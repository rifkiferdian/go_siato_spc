package httpserver

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studentInvoiceAdjustmentsHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type adjustmentItem struct {
	ID                                                                                                        int64
	Date, Invoice, NIM, Student, ChargeType, Period, Program, Type, TypeLabel, Amount, Remaining, Description string
}
type adjustmentMetrics struct {
	TotalAmount, ScholarshipAmount, OtherAmount string
	InvoiceCount                                int
}
type adjustmentForm struct{ InvoiceID, AssignmentID, Description, Reason string }
type adjustmentOption struct {
	ID    int64
	Label string
}

func (h studentInvoiceAdjustmentsHandlers) base(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(loc)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "student-invoice-adjustments"}
}
func (h studentInvoiceAdjustmentsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	typ := strings.ToUpper(strings.TrimSpace(c.Query("adjustment_type")))
	period := strings.TrimSpace(c.Query("period_id"))
	from := c.Query("date_from")
	to := c.Query("date_to")
	if typ != "SCHOLARSHIP" && typ != "EMPLOYEE_SUBSIDY" && typ != "MARKETING_DISCOUNT" {
		typ = ""
	}
	if parsePositiveID(period) == 0 {
		period = ""
	}
	q := `SELECT a.id,DATE_FORMAT(a.created_at,'%d %b %Y %H:%i'),i.invoice_number,s.nim,s.name,ct.name,COALESCE(ap.code,'Tanpa periode'),fp.name,a.adjustment_type,a.amount,GREATEST(i.final_amount-i.paid_amount,0),a.description FROM student_invoice_adjustments a JOIN student_invoices i ON i.id=a.student_invoice_id JOIN students s ON s.id=i.student_id JOIN charge_types ct ON ct.id=i.charge_type_id LEFT JOIN academic_periods ap ON ap.id=i.academic_period_id JOIN student_funding_programs sfp ON sfp.id=a.student_funding_program_id JOIN funding_programs fp ON fp.id=sfp.funding_program_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (i.invoice_number LIKE ? OR s.nim LIKE ? OR s.name LIKE ? OR fp.name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if typ != "" {
		q += ` AND a.adjustment_type=?`
		args = append(args, typ)
	}
	if period != "" {
		q += ` AND i.academic_period_id=?`
		args = append(args, period)
	}
	if _, e := time.Parse("2006-01-02", from); e == nil {
		q += ` AND a.created_at>=?`
		args = append(args, from+" 00:00:00")
	}
	if _, e := time.Parse("2006-01-02", to); e == nil {
		q += ` AND a.created_at<DATE_ADD(?,INTERVAL 1 DAY)`
		args = append(args, to+" 00:00:00")
	}
	q += ` ORDER BY a.created_at DESC,a.id DESC`
	rows, e := h.db.QueryContext(c.Request.Context(), q, args...)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer rows.Close()
	items := []adjustmentItem{}
	for rows.Next() {
		var x adjustmentItem
		var amount, remaining float64
		if rows.Scan(&x.ID, &x.Date, &x.Invoice, &x.NIM, &x.Student, &x.ChargeType, &x.Period, &x.Program, &x.Type, &amount, &remaining, &x.Description) != nil {
			continue
		}
		x.TypeLabel = adjustmentTypeLabel(x.Type)
		x.Amount = formatRupiah(amount)
		x.Remaining = formatRupiah(remaining)
		items = append(items, x)
	}
	var total, scholarship, other float64
	var count int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(SUM(amount),0),COALESCE(SUM(IF(adjustment_type='SCHOLARSHIP',amount,0)),0),COALESCE(SUM(IF(adjustment_type<>'SCHOLARSHIP',amount,0)),0),COUNT(DISTINCT student_invoice_id) FROM student_invoice_adjustments`).Scan(&total, &scholarship, &other, &count)
	d := h.base(p, "Penyesuaian Tagihan")
	d["Items"] = items
	d["Metrics"] = adjustmentMetrics{formatRupiah(total), formatRupiah(scholarship), formatRupiah(other), count}
	d["Search"] = search
	d["SelectedType"] = typ
	d["SelectedPeriod"] = period
	d["DateFrom"] = from
	d["DateTo"] = to
	d["Periods"] = h.options(c, `SELECT id,code FROM academic_periods ORDER BY start_date DESC`)
	d["Result"] = c.Query("result")
	d["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-invoice-adjustments.create")
	d["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("student-invoice-adjustments.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-invoice-adjustments/index.html", d)
}
func (h studentInvoiceAdjustmentsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, adjustmentForm{}, "")
}
func (h studentInvoiceAdjustmentsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := adjustmentForm{InvoiceID: c.PostForm("student_invoice_id"), AssignmentID: c.PostForm("student_funding_program_id"), Description: strings.TrimSpace(c.PostForm("description")), Reason: strings.TrimSpace(c.PostForm("reason"))}
	amount, typ, msg := h.calculate(c, f)
	if msg != "" {
		h.renderForm(c, 422, f, msg)
		return
	}
	if utf8.RuneCountInString(f.Description) > 255 || f.Reason == "" || utf8.RuneCountInString(f.Reason) > 500 {
		h.renderForm(c, 422, f, "Alasan wajib diisi maksimal 500 karakter dan keterangan maksimal 255 karakter.")
		return
	}
	if f.Description == "" {
		f.Description = "Penyesuaian otomatis dari program bantuan."
	}
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer tx.Rollback()
	r, e := tx.ExecContext(c.Request.Context(), `INSERT INTO student_invoice_adjustments(student_invoice_id,student_funding_program_id,adjustment_type,amount,description,created_at) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6))`, f.InvoiceID, f.AssignmentID, typ, amount, f.Description)
	if e != nil {
		h.renderForm(c, 422, f, e.Error())
		return
	}
	id, _ := r.LastInsertId()
	if e = updateAdjustedInvoice(tx, f.InvoiceID, amount); e != nil {
		c.AbortWithError(500, e)
		return
	}
	_, e = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,new_values_json,reason,performed_by_id,performed_at) VALUES('StudentInvoiceAdjustment',?,'CREATED',JSON_OBJECT('student_invoice_id',?,'student_funding_program_id',?,'adjustment_type',?,'amount',?),?,?,UTC_TIMESTAMP(6))`, id, f.InvoiceID, f.AssignmentID, typ, amount, f.Reason, p.UserID)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/student-invoice-adjustments?result=created")
}
func (h studentInvoiceAdjustmentsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	reason := strings.TrimSpace(c.PostForm("reason"))
	if reason == "" {
		c.Redirect(303, "/student-invoice-adjustments?result=reason-required")
		return
	}
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer tx.Rollback()
	var invoiceID int64
	var amount float64
	if e = tx.QueryRowContext(c.Request.Context(), `SELECT student_invoice_id,amount FROM student_invoice_adjustments WHERE id=? FOR UPDATE`, c.Param("id")).Scan(&invoiceID, &amount); e != nil {
		c.AbortWithStatus(404)
		return
	}
	if _, e = tx.ExecContext(c.Request.Context(), `DELETE FROM student_invoice_adjustments WHERE id=?`, c.Param("id")); e != nil {
		c.AbortWithError(500, e)
		return
	}
	if e = updateAdjustedInvoice(tx, strconv.FormatInt(invoiceID, 10), -amount); e != nil {
		c.AbortWithError(500, e)
		return
	}
	_, e = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,old_values_json,reason,performed_by_id,performed_at) VALUES('StudentInvoiceAdjustment',?,'DELETED',JSON_OBJECT('student_invoice_id',?,'amount',?),?,?,UTC_TIMESTAMP(6))`, c.Param("id"), invoiceID, amount, reason, p.UserID)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/student-invoice-adjustments?result=deleted")
}
func (h studentInvoiceAdjustmentsHandlers) calculate(c *gin.Context, f adjustmentForm) (float64, string, string) {
	var invoiceStudent, chargeID, assignmentStudent int64
	var original, discount, penalty, paid, value float64
	var benefitType, programType string
	var maximum sql.NullFloat64
	e := h.db.QueryRowContext(c.Request.Context(), `SELECT i.student_id,i.charge_type_id,i.original_amount,i.discount_amount,i.penalty_amount,i.paid_amount,sfp.student_id,b.benefit_type,b.benefit_value,b.maximum_amount,fp.program_type FROM student_invoices i JOIN student_funding_programs sfp ON sfp.id=? AND sfp.status='ACTIVE' JOIN funding_programs fp ON fp.id=sfp.funding_program_id AND fp.is_active=1 JOIN funding_program_benefits b ON b.funding_program_id=fp.id AND b.charge_type_id=i.charge_type_id WHERE i.id=? AND i.status NOT IN ('PAID','CANCELLED','WAIVED') LIMIT 1`, f.AssignmentID, f.InvoiceID).Scan(&invoiceStudent, &chargeID, &original, &discount, &penalty, &paid, &assignmentStudent, &benefitType, &value, &maximum, &programType)
	_ = chargeID
	if e != nil || invoiceStudent != assignmentStudent {
		return 0, "", "Tagihan dan program tidak cocok atau sudah tidak aktif."
	}
	available := math.Max(0, original-discount+penalty-paid)
	amount := 0.0
	switch benefitType {
	case "PERCENTAGE":
		amount = original * value / 100
	case "FIXED_AMOUNT":
		amount = value
	case "SPECIAL_RATE":
		amount = math.Max(0, original-discount-value)
	case "WAIVER":
		amount = available
	default:
		return 0, "", "Manfaat pembayaran pihak ketiga harus dicatat melalui menu Pembayaran."
	}
	if maximum.Valid && maximum.Float64 > 0 {
		amount = math.Min(amount, maximum.Float64)
	}
	amount = math.Min(math.Round(amount*100)/100, available)
	if amount <= 0 {
		return 0, "", "Tidak ada nominal tagihan yang dapat disesuaikan."
	}
	typ := "MARKETING_DISCOUNT"
	if programType == "SCHOLARSHIP" {
		typ = "SCHOLARSHIP"
	} else if programType == "EMPLOYEE_SUBSIDY" {
		typ = "EMPLOYEE_SUBSIDY"
	}
	return amount, typ, ""
}
func updateAdjustedInvoice(tx *sql.Tx, id string, delta float64) error {
	// MySQL mengevaluasi assignment UPDATE dari kiri ke kanan. Diskon ditulis
	// terakhir agar rumus total/status memakai nilai lama + delta tepat satu kali.
	_, e := tx.Exec(`UPDATE student_invoices SET final_amount=GREATEST(original_amount-GREATEST(discount_amount+?,0)+penalty_amount,0),status=CASE WHEN GREATEST(original_amount-GREATEST(discount_amount+?,0)+penalty_amount,0)=0 THEN 'WAIVED' WHEN paid_amount>=GREATEST(original_amount-GREATEST(discount_amount+?,0)+penalty_amount,0) THEN 'PAID' WHEN paid_amount>0 THEN 'PARTIALLY_PAID' WHEN due_date<CURDATE() THEN 'OVERDUE' ELSE 'UNPAID' END,discount_amount=GREATEST(discount_amount+?,0),updated_at=UTC_TIMESTAMP(6),version=version+1 WHERE id=?`, delta, delta, delta, delta, id)
	return e
}
func (h studentInvoiceAdjustmentsHandlers) renderForm(c *gin.Context, status int, f adjustmentForm, msg string) {
	d := h.base(auth.CurrentPrincipal(c), "Tambah Penyesuaian Tagihan")
	d["Form"] = f
	d["Error"] = msg
	d["Invoices"] = h.options(c, `SELECT i.id,CONCAT(i.invoice_number,' · ',s.nim,' - ',s.name,' · Sisa ',FORMAT(GREATEST(i.final_amount-i.paid_amount,0),0,'id_ID')) FROM student_invoices i JOIN students s ON s.id=i.student_id WHERE i.status IN ('UNPAID','PARTIALLY_PAID','OVERDUE') ORDER BY s.name,i.due_date`)
	d["Assignments"] = h.options(c, `SELECT sfp.id,CONCAT(s.nim,' - ',s.name,' · ',fp.name) FROM student_funding_programs sfp JOIN students s ON s.id=sfp.student_id JOIN funding_programs fp ON fp.id=sfp.funding_program_id WHERE sfp.status='ACTIVE' AND fp.is_active=1 ORDER BY s.name,fp.name`)
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "student-invoice-adjustments/form.html", d)
}
func (h studentInvoiceAdjustmentsHandlers) options(c *gin.Context, q string) []adjustmentOption {
	rows, e := h.db.QueryContext(c.Request.Context(), q)
	if e != nil {
		return nil
	}
	defer rows.Close()
	out := []adjustmentOption{}
	for rows.Next() {
		var x adjustmentOption
		if rows.Scan(&x.ID, &x.Label) == nil {
			out = append(out, x)
		}
	}
	return out
}
func adjustmentTypeLabel(v string) string {
	switch v {
	case "SCHOLARSHIP":
		return "Beasiswa"
	case "EMPLOYEE_SUBSIDY":
		return "Subsidi Karyawan"
	case "MARKETING_DISCOUNT":
		return "Diskon Marketing"
	}
	return v
}
func adjustmentAmountPreview(v float64) string { return fmt.Sprintf("Rp %.0f", v) }
