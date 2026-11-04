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

type studentBillingPoliciesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type studentBillingPolicyItem struct {
	ID                                                                                                                                                                                         int64
	NIM, StudentName, ProgramCode, CategoryName, PolicyCode, PolicyName, PeriodCode, AssignmentType, AssignmentLabel, BaseAmount, EffectiveAmount, Status, StatusLabel, AssignedAt, ApprovedBy string
}
type studentBillingPolicyMetrics struct{ Total, Active, Automatic, Manual int }
type studentBillingPolicyForm struct {
	ID                                                                              int64
	StudentID, PolicyID, EffectiveAmount, AssignmentType, Status, AssignedAt, Notes string
}

func (h studentBillingPoliciesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if !validAssignmentStatus(status) {
		status = ""
	}
	q := `SELECT sbp.id,s.nim,s.name,sp.code,COALESCE(sc.name,''),bp.code,bp.name,ap.code,sbp.assignment_type,bp.total_amount,sbp.effective_amount,sbp.status,DATE_FORMAT(sbp.assigned_at,'%d %b %Y'),COALESCE(u.Email,IF(sbp.assignment_type='AUTOMATIC','Oleh sistem','-')) FROM student_billing_policies sbp JOIN students s ON s.id=sbp.student_id JOIN study_programs sp ON sp.id=s.study_program_id LEFT JOIN student_categories sc ON sc.id=s.student_category_id JOIN billing_policies bp ON bp.id=sbp.billing_policy_id JOIN academic_periods ap ON ap.id=bp.academic_period_id LEFT JOIN users u ON u.Id=sbp.approved_by_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (s.nim LIKE ? OR s.name LIKE ? OR bp.code LIKE ? OR bp.name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if status != "" {
		q += ` AND sbp.status=?`
		args = append(args, status)
	}
	q += ` ORDER BY sbp.assigned_at DESC,s.name`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []studentBillingPolicyItem{}
	for rows.Next() {
		var x studentBillingPolicyItem
		var base, effective float64
		if err = rows.Scan(&x.ID, &x.NIM, &x.StudentName, &x.ProgramCode, &x.CategoryName, &x.PolicyCode, &x.PolicyName, &x.PeriodCode, &x.AssignmentType, &base, &effective, &x.Status, &x.AssignedAt, &x.ApprovedBy); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.AssignmentLabel = assignmentTypeLabel(x.AssignmentType)
		x.StatusLabel = assignmentStatusLabel(x.Status)
		x.BaseAmount = formatRupiah(base)
		x.EffectiveAmount = formatRupiah(effective)
		items = append(items, x)
	}
	var metrics studentBillingPolicyMetrics
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(status='ACTIVE'),0),COALESCE(SUM(assignment_type='AUTOMATIC'),0),COALESCE(SUM(assignment_type='MANUAL'),0) FROM student_billing_policies`).Scan(&metrics.Total, &metrics.Active, &metrics.Automatic, &metrics.Manual); err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, "Penetapan Mahasiswa")
	data["Items"] = items
	data["Metrics"] = metrics
	data["Search"] = search
	data["SelectedStatus"] = status
	data["Status"] = c.Query("result")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-billing-policies.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("student-billing-policies.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("student-billing-policies.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-billing-policies/index.html", data)
}
func (h studentBillingPoliciesHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Penetapan Mahasiswa", studentBillingPolicyForm{StudentID: c.Query("student_id"), AssignmentType: "AUTOMATIC", Status: "ACTIVE", AssignedAt: time.Now().Format("2006-01-02")}, "")
}
func (h studentBillingPoliciesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := studentBillingPolicyFormFromRequest(c)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Penetapan Mahasiswa", f, msg)
		return
	}
	var approved any
	if f.AssignmentType == "MANUAL" {
		approved = p.UserID
	}
	_, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO student_billing_policies(student_id,billing_policy_id,effective_amount,assignment_type,status,approved_by_id,assigned_at,notes,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, f.StudentID, f.PolicyID, f.EffectiveAmount, f.AssignmentType, f.Status, approved, f.AssignedAt, nullableString(f.Notes))
	if err != nil {
		h.renderForm(c, 422, "Tambah Penetapan Mahasiswa", f, err.Error())
		return
	}
	c.Redirect(303, "/student-billing-policies?result=created")
}
func (h studentBillingPoliciesHandlers) editForm(c *gin.Context) {
	f, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Penetapan Mahasiswa", f, "")
}
func (h studentBillingPoliciesHandlers) update(c *gin.Context) {
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
	f := studentBillingPolicyFormFromRequest(c)
	f.ID = id
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Penetapan Mahasiswa", f, msg)
		return
	}
	var approved any
	if f.AssignmentType == "MANUAL" {
		var existing sql.NullString
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT approved_by_id FROM student_billing_policies WHERE id=?`, id).Scan(&existing)
		if existing.Valid {
			approved = existing.String
		} else {
			approved = p.UserID
		}
	}
	r, err := h.db.ExecContext(c.Request.Context(), `UPDATE student_billing_policies SET student_id=?,billing_policy_id=?,effective_amount=?,assignment_type=?,status=?,approved_by_id=?,assigned_at=?,notes=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, f.StudentID, f.PolicyID, f.EffectiveAmount, f.AssignmentType, f.Status, approved, f.AssignedAt, nullableString(f.Notes), id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Penetapan Mahasiswa", f, err.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		var exists int
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_billing_policies WHERE id=?`, id).Scan(&exists)
		if exists == 0 {
			c.AbortWithStatus(404)
			return
		}
	}
	c.Redirect(303, "/student-billing-policies?result=updated")
}
func (h studentBillingPoliciesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var refs int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE student_billing_policy_id=?`, id).Scan(&refs); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if refs > 0 {
		c.Redirect(303, "/student-billing-policies?result=in-use")
		return
	}
	r, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM student_billing_policies WHERE id=?`, id)
	if err != nil {
		c.Redirect(303, "/student-billing-policies?result=delete-failed")
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/student-billing-policies?result=deleted")
}

func (h studentBillingPoliciesHandlers) find(c *gin.Context, id any) (studentBillingPolicyForm, error) {
	var f studentBillingPolicyForm
	var amount float64
	var assigned time.Time
	var notes sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,student_id,billing_policy_id,effective_amount,assignment_type,status,assigned_at,notes FROM student_billing_policies WHERE id=?`, id).Scan(&f.ID, &f.StudentID, &f.PolicyID, &amount, &f.AssignmentType, &f.Status, &assigned, &notes)
	f.EffectiveAmount = strconv.FormatFloat(amount, 'f', 2, 64)
	f.AssignedAt = assigned.Format("2006-01-02")
	f.Notes = notes.String
	return f, err
}
func studentBillingPolicyFormFromRequest(c *gin.Context) studentBillingPolicyForm {
	return studentBillingPolicyForm{StudentID: strings.TrimSpace(c.PostForm("student_id")), PolicyID: strings.TrimSpace(c.PostForm("billing_policy_id")), EffectiveAmount: strings.TrimSpace(c.PostForm("effective_amount")), AssignmentType: strings.ToUpper(strings.TrimSpace(c.PostForm("assignment_type"))), Status: strings.ToUpper(strings.TrimSpace(c.PostForm("status"))), AssignedAt: strings.TrimSpace(c.PostForm("assigned_at")), Notes: strings.TrimSpace(c.PostForm("notes"))}
}
func (h studentBillingPoliciesHandlers) validate(c *gin.Context, f studentBillingPolicyForm, id int64) string {
	amount, e1 := strconv.ParseFloat(f.EffectiveAmount, 64)
	_, e2 := time.Parse("2006-01-02", f.AssignedAt)
	switch {
	case parsePositiveID(f.StudentID) == 0:
		return "Mahasiswa wajib dipilih."
	case parsePositiveID(f.PolicyID) == 0:
		return "Kebijakan pembayaran wajib dipilih."
	case e1 != nil || amount < 0:
		return "Nominal efektif harus berupa angka nol atau lebih."
	case f.AssignmentType != "AUTOMATIC" && f.AssignmentType != "MANUAL":
		return "Jenis penetapan tidak valid."
	case !validAssignmentStatus(f.Status):
		return "Status penetapan tidak valid."
	case e2 != nil:
		return "Tanggal penetapan tidak valid."
	case utf8.RuneCountInString(f.Notes) > 500:
		return "Catatan maksimal 500 karakter."
	}
	var studentProgram int64
	var studentCategory sql.NullInt64
	var studentYear int
	var studentClass sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT study_program_id,student_category_id,cohort_year,cohort_year_class FROM students WHERE id=?`, f.StudentID).Scan(&studentProgram, &studentCategory, &studentYear, &studentClass)
	if errors.Is(err, sql.ErrNoRows) {
		return "Mahasiswa tidak ditemukan."
	}
	if err != nil {
		return "Gagal memvalidasi mahasiswa."
	}
	var policyProgram, policyCategory, policyYear sql.NullInt64
	var policyClass sql.NullString
	var policyTotal float64
	var periodID, chargeTypeID int64
	err = h.db.QueryRowContext(c.Request.Context(), `SELECT study_program_id,student_category_id,cohort_year,cohort_year_class,total_amount,academic_period_id,charge_type_id FROM billing_policies WHERE id=?`, f.PolicyID).Scan(&policyProgram, &policyCategory, &policyYear, &policyClass, &policyTotal, &periodID, &chargeTypeID)
	if errors.Is(err, sql.ErrNoRows) {
		return "Kebijakan pembayaran tidak ditemukan."
	}
	if err != nil {
		return "Gagal memvalidasi kebijakan pembayaran."
	}
	var duplicate int
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_billing_policies WHERE student_id=? AND billing_policy_id=? AND id<>?`, f.StudentID, f.PolicyID, id).Scan(&duplicate); err != nil {
		return "Gagal memvalidasi penetapan."
	}
	if duplicate > 0 {
		return "Kebijakan tersebut sudah ditetapkan kepada mahasiswa ini."
	}
	if f.Status == "ACTIVE" {
		if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_billing_policies sbp JOIN billing_policies bp ON bp.id=sbp.billing_policy_id WHERE sbp.student_id=? AND sbp.status='ACTIVE' AND sbp.id<>? AND bp.academic_period_id=? AND bp.charge_type_id=?`, f.StudentID, id, periodID, chargeTypeID).Scan(&duplicate); err != nil {
			return "Gagal memvalidasi penetapan aktif."
		}
		if duplicate > 0 {
			return "Mahasiswa sudah memiliki penetapan aktif untuk semester dan jenis biaya yang sama."
		}
	}
	if amount > policyTotal {
		return fmt.Sprintf("Nominal efektif tidak boleh melebihi tarif dasar %s.", formatRupiah(policyTotal))
	}
	if f.AssignmentType == "AUTOMATIC" {
		matches := (!policyProgram.Valid || policyProgram.Int64 == studentProgram) && (!policyCategory.Valid || (studentCategory.Valid && policyCategory.Int64 == studentCategory.Int64)) && (!policyYear.Valid || policyYear.Int64 == int64(studentYear)) && (!policyClass.Valid || strings.EqualFold(policyClass.String, studentClass.String))
		if !matches {
			return "Kebijakan tidak cocok dengan prodi, golongan, angkatan, atau kelas mahasiswa. Gunakan penetapan manual untuk pengecualian."
		}
	}
	return ""
}
func (h studentBillingPoliciesHandlers) studentOptions(c *gin.Context) []billingPolicyOption {
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id,CONCAT(nim,' - ',name,' (',COALESCE(cohort_year_class,cohort_year),')') FROM students ORDER BY status<>'ACTIVE',name`)
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
func (h studentBillingPoliciesHandlers) policyOptions(c *gin.Context) []billingPolicyOption {
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id,CONCAT(code,' - ',name,' - Rp ',FORMAT(total_amount,0)) FROM billing_policies ORDER BY is_active DESC,created_at DESC,name`)
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
func (h studentBillingPoliciesHandlers) renderForm(c *gin.Context, status int, title string, f studentBillingPolicyForm, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["Form"] = f
	data["Students"] = h.studentOptions(c)
	data["Policies"] = h.policyOptions(c)
	data["IsEdit"] = f.ID > 0
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "student-billing-policies/form.html", data)
}
func (h studentBillingPoliciesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "student-billing-policies"}
}
func validAssignmentStatus(v string) bool {
	return v == "DRAFT" || v == "ACTIVE" || v == "COMPLETED" || v == "CANCELLED"
}
func assignmentTypeLabel(v string) string {
	if v == "AUTOMATIC" {
		return "Otomatis"
	}
	return "Manual"
}
func assignmentStatusLabel(v string) string {
	switch v {
	case "DRAFT":
		return "Draft"
	case "ACTIVE":
		return "Aktif"
	case "COMPLETED":
		return "Selesai"
	case "CANCELLED":
		return "Dibatalkan"
	}
	return v
}
