package httpserver

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type paymentsHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type paymentItem struct {
	ID                                                                                                          int64
	Number, Reference, NIM, Student, Date, Method, MethodLabel, Amount, Status, StatusLabel, Receiver, Verifier string
	AllocationCount                                                                                             int
	CanVerify                                                                                                   bool
}
type paymentMetrics struct{ Total, Pending, Verified, Closed int }
type paymentOption struct {
	ID    int64
	Label string
}
type paymentForm struct{ Number, StudentID, InvoiceID, Amount, Method, Reference, PayerType, PayerName, Notes string }

func (h paymentsHandlers) base(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(loc)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "payments"}
}

func (h paymentsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.ToUpper(c.Query("status"))
	method := strings.ToUpper(c.Query("payment_method"))
	q := `SELECT p.id,p.payment_number,COALESCE(p.reference_number,''),s.nim,s.name,DATE_FORMAT(p.payment_date,'%d %b %Y %H:%i'),p.payment_method,p.amount,p.status,u.email,COALESCE(v.email,''),(SELECT COUNT(*) FROM payment_allocations a WHERE a.payment_id=p.id) FROM payments p JOIN students s ON s.id=p.student_id JOIN users u ON u.id=p.received_by_id LEFT JOIN users v ON v.id=p.verified_by_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (p.payment_number LIKE ? OR s.nim LIKE ? OR s.name LIKE ? OR p.reference_number LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if status != "" {
		q += ` AND p.status=?`
		args = append(args, status)
	}
	if method != "" {
		q += ` AND p.payment_method=?`
		args = append(args, method)
	}
	q += ` ORDER BY p.payment_date DESC,p.id DESC`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []paymentItem{}
	for rows.Next() {
		var x paymentItem
		var amount float64
		if rows.Scan(&x.ID, &x.Number, &x.Reference, &x.NIM, &x.Student, &x.Date, &x.Method, &amount, &x.Status, &x.Receiver, &x.Verifier, &x.AllocationCount) != nil {
			continue
		}
		x.Amount = formatRupiah(amount)
		x.MethodLabel = paymentMethodLabel(x.Method)
		x.StatusLabel = paymentStatusLabel(x.Status)
		x.CanVerify = x.Status == "PENDING" && x.Receiver != p.Email
		items = append(items, x)
	}
	var m paymentMetrics
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(status='PENDING'),0),COALESCE(SUM(status='VERIFIED'),0),COALESCE(SUM(status IN ('REJECTED','CANCELLED')),0) FROM payments`).Scan(&m.Total, &m.Pending, &m.Verified, &m.Closed)
	d := h.base(p, "Pembayaran")
	d["Items"] = items
	d["Metrics"] = m
	d["Search"] = search
	d["SelectedStatus"] = status
	d["SelectedMethod"] = method
	d["Result"] = c.Query("result")
	d["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("payments.create")
	d["CanVerify"] = p.HasRole("Super Admin") || p.HasPermission("payments.verify")
	d["CanReject"] = p.HasRole("Super Admin") || p.HasPermission("payments.reject")
	d["CanCancel"] = p.HasRole("Super Admin") || p.HasPermission("payments.cancel")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "payments/index.html", d)
}

func (h paymentsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, paymentForm{Number: h.nextNumber(c), Method: "BANK_TRANSFER", PayerType: "STUDENT"}, "")
}
func (h paymentsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := paymentForm{Number: c.PostForm("payment_number"), StudentID: c.PostForm("student_id"), InvoiceID: c.PostForm("invoice_id"), Amount: c.PostForm("amount"), Method: c.PostForm("payment_method"), Reference: strings.TrimSpace(c.PostForm("reference_number")), PayerType: c.PostForm("payer_type"), PayerName: strings.TrimSpace(c.PostForm("payer_name")), Notes: strings.TrimSpace(c.PostForm("notes"))}
	amount, e := strconv.ParseFloat(f.Amount, 64)
	if e != nil || amount <= 0 || parsePositiveID(f.StudentID) == 0 || parsePositiveID(f.InvoiceID) == 0 {
		h.renderForm(c, 422, f, "Mahasiswa, tagihan, dan nominal pembayaran wajib diisi dengan benar.")
		return
	}
	var owner int64
	var remaining float64
	if h.db.QueryRowContext(c, `SELECT student_id,GREATEST(final_amount-paid_amount,0) FROM student_invoices WHERE id=?`, f.InvoiceID).Scan(&owner, &remaining) != nil || owner != parsePositiveID(f.StudentID) || amount > remaining {
		h.renderForm(c, 422, f, "Tagihan tidak sesuai dengan mahasiswa atau nominal melebihi sisa tagihan.")
		return
	}
	tx, err := h.db.BeginTx(c, nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(c, `INSERT INTO payments(payment_number,student_id,payment_date,amount,payment_method,reference_number,status,received_by_id,notes,created_at,updated_at,payer_name,payer_type) VALUES(?,?,NOW(),?,?,?,?,?, ?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?,?)`, f.Number, f.StudentID, amount, f.Method, nullableString(f.Reference), "PENDING", p.UserID, nullableString(f.Notes), nullableString(f.PayerName), f.PayerType)
	if err != nil {
		h.renderForm(c, 422, f, err.Error())
		return
	}
	id, _ := r.LastInsertId()
	_, err = tx.ExecContext(c, `INSERT INTO payment_allocations(payment_id,student_invoice_id,amount,created_at) VALUES(?,?,?,UTC_TIMESTAMP(6))`, id, f.InvoiceID, amount)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	_, err = tx.ExecContext(c, `INSERT INTO audit_logs(entity_type,entity_id,action,new_values_json,performed_by_id,performed_at) VALUES('Payment',?,'CREATED',JSON_OBJECT('payment_number',?,'student_id',?,'amount',?,'payment_method',?,'status','PENDING'),?,UTC_TIMESTAMP(6))`, id, f.Number, f.StudentID, amount, f.Method, p.UserID)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/payments?result=created")
}

func (h paymentsHandlers) changeStatus(c *gin.Context, target string) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var status, receiver string
	if h.db.QueryRowContext(c, `SELECT status,received_by_id FROM payments WHERE id=?`, id).Scan(&status, &receiver) != nil {
		c.AbortWithStatus(404)
		return
	}
	if target == "VERIFIED" && receiver == p.UserID {
		c.Redirect(303, "/payments?result=maker-checker")
		return
	}
	if status != "PENDING" && !(target == "CANCELLED" && status == "VERIFIED") {
		c.Redirect(303, "/payments?result=invalid-status")
		return
	}
	reason := strings.TrimSpace(c.PostForm("reason"))
	if (target == "REJECTED" || target == "CANCELLED") && reason == "" {
		c.Redirect(303, "/payments?result=reason-required")
		return
	}
	tx, _ := h.db.BeginTx(c, nil)
	defer tx.Rollback()
	if target == "VERIFIED" {
		_, _ = tx.ExecContext(c, `UPDATE payments SET status='VERIFIED',verified_by_id=?,verified_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=?`, p.UserID, id)
	} else if target == "REJECTED" {
		_, _ = tx.ExecContext(c, `UPDATE payments SET status='REJECTED',verified_by_id=?,verified_at=UTC_TIMESTAMP(6),rejected_reason=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, p.UserID, reason, id)
	} else {
		_, _ = tx.ExecContext(c, `UPDATE payments SET status='CANCELLED',cancelled_by_id=?,cancelled_at=UTC_TIMESTAMP(6),cancelled_reason=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, p.UserID, reason, id)
	}
	_, err := tx.ExecContext(c, `INSERT INTO audit_logs(entity_type,entity_id,action,old_values_json,new_values_json,reason,performed_by_id,performed_at) VALUES('Payment',?,?,JSON_OBJECT('status',?),JSON_OBJECT('status',?),?,?,UTC_TIMESTAMP(6))`, id, target, status, target, nullableString(reason), p.UserID)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	_ = h.recalculate(tx, id)
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/payments?result="+strings.ToLower(target))
}
func (h paymentsHandlers) verify(c *gin.Context) { h.changeStatus(c, "VERIFIED") }
func (h paymentsHandlers) reject(c *gin.Context) { h.changeStatus(c, "REJECTED") }
func (h paymentsHandlers) cancel(c *gin.Context) { h.changeStatus(c, "CANCELLED") }
func (h paymentsHandlers) recalculate(tx *sql.Tx, paymentID string) error {
	rows, e := tx.Query(`SELECT student_invoice_id FROM payment_allocations WHERE payment_id=?`, paymentID)
	if e != nil {
		return e
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		_, e = tx.Exec(`UPDATE student_invoices i SET paid_amount=COALESCE((SELECT SUM(a.amount) FROM payment_allocations a JOIN payments p ON p.id=a.payment_id WHERE a.student_invoice_id=i.id AND p.status='VERIFIED'),0),status=CASE WHEN final_amount=0 THEN 'WAIVED' WHEN COALESCE((SELECT SUM(a.amount) FROM payment_allocations a JOIN payments p ON p.id=a.payment_id WHERE a.student_invoice_id=i.id AND p.status='VERIFIED'),0)>=final_amount THEN 'PAID' WHEN COALESCE((SELECT SUM(a.amount) FROM payment_allocations a JOIN payments p ON p.id=a.payment_id WHERE a.student_invoice_id=i.id AND p.status='VERIFIED'),0)>0 THEN 'PARTIALLY_PAID' WHEN due_date<CURDATE() THEN 'OVERDUE' ELSE 'UNPAID' END,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, id)
		if e != nil {
			return e
		}
	}
	return nil
}

func (h paymentsHandlers) print(c *gin.Context) {
	var x paymentItem
	var amount float64
	err := h.db.QueryRowContext(c, `SELECT p.id,p.payment_number,COALESCE(p.reference_number,''),s.nim,s.name,DATE_FORMAT(p.payment_date,'%d %b %Y %H:%i'),p.payment_method,p.amount,p.status,u.email,COALESCE(v.email,''),(SELECT COUNT(*) FROM payment_allocations a WHERE a.payment_id=p.id) FROM payments p JOIN students s ON s.id=p.student_id JOIN users u ON u.id=p.received_by_id LEFT JOIN users v ON v.id=p.verified_by_id WHERE p.id=?`, c.Param("id")).Scan(&x.ID, &x.Number, &x.Reference, &x.NIM, &x.Student, &x.Date, &x.Method, &amount, &x.Status, &x.Receiver, &x.Verifier, &x.AllocationCount)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}
	x.Amount = formatRupiah(amount)
	x.MethodLabel = paymentMethodLabel(x.Method)
	x.StatusLabel = paymentStatusLabel(x.Status)
	c.HTML(200, "payments/print.html", gin.H{"Title": "Kuitansi Pembayaran", "AppName": h.cfg.Name, "Year": time.Now().Year(), "Payment": x})
}
func (h paymentsHandlers) renderForm(c *gin.Context, status int, f paymentForm, msg string) {
	d := h.base(auth.CurrentPrincipal(c), "Tambah Pembayaran")
	d["Form"] = f
	d["Error"] = msg
	d["Students"] = h.paymentOptions(c, `SELECT id,CONCAT(nim,' - ',name) FROM students WHERE status='ACTIVE' ORDER BY name`)
	d["Invoices"] = h.paymentOptions(c, `SELECT i.id,CONCAT(s.nim,' | ',i.invoice_number,' | ',ct.name,' | Sisa Rp ',FORMAT(GREATEST(i.final_amount-i.paid_amount,0),0,'id_ID')) FROM student_invoices i JOIN students s ON s.id=i.student_id JOIN charge_types ct ON ct.id=i.charge_type_id WHERE i.status IN ('UNPAID','PARTIALLY_PAID','OVERDUE') ORDER BY s.name,i.due_date`)
	c.HTML(status, "payments/form.html", d)
}
func (h paymentsHandlers) paymentOptions(c *gin.Context, q string) []paymentOption {
	rows, e := h.db.QueryContext(c, q)
	if e != nil {
		return nil
	}
	defer rows.Close()
	out := []paymentOption{}
	for rows.Next() {
		var x paymentOption
		if rows.Scan(&x.ID, &x.Label) == nil {
			out = append(out, x)
		}
	}
	return out
}
func (h paymentsHandlers) nextNumber(c *gin.Context) string {
	prefix := fmt.Sprintf("PAY-%d-", time.Now().Year())
	var n int
	_ = h.db.QueryRowContext(c, `SELECT COALESCE(MAX(CAST(SUBSTRING(payment_number,10) AS UNSIGNED)),0)+1 FROM payments WHERE payment_number LIKE ?`, prefix+"%").Scan(&n)
	return fmt.Sprintf("%s%04d", prefix, n)
}
func paymentStatusLabel(v string) string {
	switch v {
	case "PENDING":
		return "Pending"
	case "VERIFIED":
		return "Terverifikasi"
	case "REJECTED":
		return "Ditolak"
	case "CANCELLED":
		return "Dibatalkan"
	}
	return v
}
