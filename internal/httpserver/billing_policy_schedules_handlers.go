package httpserver

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type billingPolicySchedulesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type billingPolicyScheduleItem struct {
	ID, PolicyID                                                     int64
	PolicyCode, PolicyName, Description, DueDate, Percentage, Amount string
	SequenceNo                                                       int
}
type billingPolicyScheduleMetrics struct{ Total, Policies, Upcoming, ThisMonth int }
type billingPolicyScheduleForm struct {
	ID                                                             int64
	PolicyID, SequenceNo, Description, DueDate, Amount, Percentage string
}

func (h billingPolicySchedulesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	policy := c.Query("billing_policy_id")
	q := `SELECT bps.id,bps.billing_policy_id,bp.code,bp.name,bps.sequence_no,bps.description,DATE_FORMAT(bps.due_date,'%d %b %Y'),bps.amount,bps.percentage FROM billing_policy_schedules bps JOIN billing_policies bp ON bp.id=bps.billing_policy_id WHERE 1=1`
	args := []any{}
	if parsePositiveID(policy) > 0 {
		q += ` AND bps.billing_policy_id=?`
		args = append(args, policy)
	}
	if search != "" {
		q += ` AND (bps.description LIKE ? OR bp.code LIKE ? OR bp.name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY bps.due_date,bps.sequence_no`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []billingPolicyScheduleItem{}
	for rows.Next() {
		var x billingPolicyScheduleItem
		var amount float64
		var percentage sql.NullFloat64
		if err = rows.Scan(&x.ID, &x.PolicyID, &x.PolicyCode, &x.PolicyName, &x.SequenceNo, &x.Description, &x.DueDate, &amount, &percentage); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Amount = formatRupiah(amount)
		if percentage.Valid {
			x.Percentage = strconv.FormatFloat(percentage.Float64, 'f', 2, 64) + "%"
		} else {
			x.Percentage = "—"
		}
		items = append(items, x)
	}
	var metrics billingPolicyScheduleMetrics
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COUNT(DISTINCT billing_policy_id),COALESCE(SUM(due_date>=CURDATE()),0),COALESCE(SUM(due_date>=DATE_FORMAT(CURDATE(),'%Y-%m-01') AND due_date<DATE_ADD(DATE_FORMAT(CURDATE(),'%Y-%m-01'),INTERVAL 1 MONTH)),0) FROM billing_policy_schedules`).Scan(&metrics.Total, &metrics.Policies, &metrics.Upcoming, &metrics.ThisMonth); err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, "Jadwal Pembayaran")
	data["Items"] = items
	data["Metrics"] = metrics
	data["Policies"] = h.policyOptions(c)
	data["Search"] = search
	data["SelectedPolicy"] = policy
	data["Status"] = c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("billing-policy-schedules.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("billing-policy-schedules.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("billing-policy-schedules.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "billing-policy-schedules/index.html", data)
}
func (h billingPolicySchedulesHandlers) createForm(c *gin.Context) {
	f := billingPolicyScheduleForm{PolicyID: c.Query("billing_policy_id"), DueDate: time.Now().Format("2006-01-02")}
	if parsePositiveID(f.PolicyID) > 0 {
		var count, installments int
		var total float64
		if h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM billing_policy_schedules WHERE billing_policy_id=bp.id),bp.installment_count,bp.total_amount FROM billing_policies bp WHERE bp.id=?`, f.PolicyID).Scan(&count, &installments, &total) == nil {
			f.SequenceNo = strconv.Itoa(count + 1)
			if installments > 0 {
				f.Amount = strconv.FormatFloat(total/float64(installments), 'f', 2, 64)
			}
		}
	}
	h.renderForm(c, 200, "Tambah Jadwal Pembayaran", f, "")
}
func (h billingPolicySchedulesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := billingScheduleFormFromRequest(c)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Jadwal Pembayaran", f, msg)
		return
	}
	_, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO billing_policy_schedules(billing_policy_id,sequence_no,description,due_date,amount,percentage,created_at,updated_at) VALUES(?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, f.PolicyID, f.SequenceNo, f.Description, f.DueDate, f.Amount, nullableDecimal(f.Percentage))
	if err != nil {
		h.renderForm(c, 422, "Tambah Jadwal Pembayaran", f, err.Error())
		return
	}
	c.Redirect(303, "/billing-policy-schedules?billing_policy_id="+f.PolicyID+"&status=created")
}
func (h billingPolicySchedulesHandlers) editForm(c *gin.Context) {
	f, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Jadwal Pembayaran", f, "")
}
func (h billingPolicySchedulesHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	if id == 0 {
		c.AbortWithStatus(404)
		return
	}
	f := billingScheduleFormFromRequest(c)
	f.ID = id
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Jadwal Pembayaran", f, msg)
		return
	}
	r, err := h.db.ExecContext(c.Request.Context(), `UPDATE billing_policy_schedules SET billing_policy_id=?,sequence_no=?,description=?,due_date=?,amount=?,percentage=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, f.PolicyID, f.SequenceNo, f.Description, f.DueDate, f.Amount, nullableDecimal(f.Percentage), id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Jadwal Pembayaran", f, err.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		var exists int
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM billing_policy_schedules WHERE id=?`, id).Scan(&exists)
		if exists == 0 {
			c.AbortWithStatus(404)
			return
		}
	}
	c.Redirect(303, "/billing-policy-schedules?billing_policy_id="+f.PolicyID+"&status=updated")
}
func (h billingPolicySchedulesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var policyID int64
	var refs int
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT bps.billing_policy_id,(SELECT COUNT(*) FROM student_invoices WHERE billing_policy_schedule_id=bps.id) FROM billing_policy_schedules bps WHERE bps.id=?`, id).Scan(&policyID, &refs)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if refs > 0 {
		c.Redirect(303, fmt.Sprintf("/billing-policy-schedules?billing_policy_id=%d&status=in-use", policyID))
		return
	}
	_, err = h.db.ExecContext(c.Request.Context(), `DELETE FROM billing_policy_schedules WHERE id=?`, id)
	if err != nil {
		c.Redirect(303, fmt.Sprintf("/billing-policy-schedules?billing_policy_id=%d&status=delete-failed", policyID))
		return
	}
	c.Redirect(303, fmt.Sprintf("/billing-policy-schedules?billing_policy_id=%d&status=deleted", policyID))
}

func (h billingPolicySchedulesHandlers) find(c *gin.Context, id any) (billingPolicyScheduleForm, error) {
	var f billingPolicyScheduleForm
	var due time.Time
	var amount float64
	var percentage sql.NullFloat64
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,billing_policy_id,sequence_no,description,due_date,amount,percentage FROM billing_policy_schedules WHERE id=?`, id).Scan(&f.ID, &f.PolicyID, &f.SequenceNo, &f.Description, &due, &amount, &percentage)
	f.DueDate = due.Format("2006-01-02")
	f.Amount = strconv.FormatFloat(amount, 'f', 2, 64)
	if percentage.Valid {
		f.Percentage = strconv.FormatFloat(percentage.Float64, 'f', 2, 64)
	}
	return f, err
}
func billingScheduleFormFromRequest(c *gin.Context) billingPolicyScheduleForm {
	return billingPolicyScheduleForm{PolicyID: strings.TrimSpace(c.PostForm("billing_policy_id")), SequenceNo: strings.TrimSpace(c.PostForm("sequence_no")), Description: strings.TrimSpace(c.PostForm("description")), DueDate: strings.TrimSpace(c.PostForm("due_date")), Amount: strings.TrimSpace(c.PostForm("amount")), Percentage: strings.TrimSpace(c.PostForm("percentage"))}
}
func (h billingPolicySchedulesHandlers) validate(c *gin.Context, f billingPolicyScheduleForm, id int64) string {
	sequence, e1 := strconv.Atoi(f.SequenceNo)
	amount, e2 := strconv.ParseFloat(f.Amount, 64)
	due, e3 := time.Parse("2006-01-02", f.DueDate)
	var percentage float64
	var e4 error
	if f.Percentage != "" {
		percentage, e4 = strconv.ParseFloat(f.Percentage, 64)
	}
	switch {
	case parsePositiveID(f.PolicyID) == 0:
		return "Kebijakan pembayaran wajib dipilih."
	case e1 != nil || sequence < 1 || sequence > 120:
		return "Urutan termin harus antara 1 dan 120."
	case f.Description == "" || utf8.RuneCountInString(f.Description) > 150:
		return "Keterangan wajib diisi dan maksimal 150 karakter."
	case e3 != nil:
		return "Tanggal jatuh tempo tidak valid."
	case e2 != nil || amount < 0:
		return "Nominal harus berupa angka nol atau lebih."
	case f.Percentage != "" && (e4 != nil || percentage < 0 || percentage > 100):
		return "Persentase harus berada antara 0 dan 100."
	}
	var installmentCount int
	var total float64
	var start, end time.Time
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT installment_count,total_amount,ap.start_date,ap.end_date FROM billing_policies bp JOIN academic_periods ap ON ap.id=bp.academic_period_id WHERE bp.id=?`, f.PolicyID).Scan(&installmentCount, &total, &start, &end)
	if errors.Is(err, sql.ErrNoRows) {
		return "Kebijakan pembayaran tidak ditemukan."
	}
	if err != nil {
		return "Gagal memvalidasi kebijakan pembayaran."
	}
	if sequence > installmentCount {
		return fmt.Sprintf("Urutan termin maksimal %d sesuai kebijakan.", installmentCount)
	}
	var duplicate int
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM billing_policy_schedules WHERE billing_policy_id=? AND sequence_no=? AND id<>?`, f.PolicyID, sequence, id).Scan(&duplicate); err != nil {
		return "Gagal memvalidasi urutan termin."
	}
	if duplicate > 0 {
		return "Urutan termin tersebut sudah digunakan pada kebijakan ini."
	}
	var scheduled float64
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(SUM(amount),0) FROM billing_policy_schedules WHERE billing_policy_id=? AND id<>?`, f.PolicyID, id).Scan(&scheduled); err != nil {
		return "Gagal menghitung total jadwal."
	}
	if scheduled+amount > total {
		return fmt.Sprintf("Total jadwal melebihi tarif kebijakan %s.", formatRupiah(total))
	}
	if due.Before(start) || due.After(end) {
		return "Tanggal jatuh tempo harus berada dalam periode akademik kebijakan."
	}
	return ""
}
func (h billingPolicySchedulesHandlers) policyOptions(c *gin.Context) []billingPolicyOption {
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT bp.id,CONCAT(bp.code,' · ',bp.name,' · ',ct.name,' · ',ap.academic_year,' ',ap.semester) FROM billing_policies bp JOIN charge_types ct ON ct.id=bp.charge_type_id JOIN academic_periods ap ON ap.id=bp.academic_period_id ORDER BY ap.start_date DESC,bp.name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []billingPolicyOption{}
	for rows.Next() {
		var x billingPolicyOption
		if rows.Scan(&x.ID, &x.Label) == nil {
			out = append(out, x)
		}
	}
	return out
}
func (h billingPolicySchedulesHandlers) renderForm(c *gin.Context, status int, title string, f billingPolicyScheduleForm, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["Form"] = f
	data["Policies"] = h.policyOptions(c)
	data["IsEdit"] = f.ID > 0
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "billing-policy-schedules/form.html", data)
}
func (h billingPolicySchedulesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "billing-policy-schedules"}
}
func nullableDecimal(v string) any {
	if v == "" {
		return nil
	}
	return v
}
