package httpserver

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studentFundingProgramsHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type studentFundingProgramItem struct {
	ID                                                                                                     int64
	Student, NIM, Program, Benefit, ChargeType, Period, Reference, StartDate, EndDate, Status, StatusLabel string
	AdjustmentCount                                                                                        int
}
type studentFundingProgramMetrics struct{ Total, Active, Applied, Unapplied int }
type studentFundingProgramForm struct {
	ID                                                                           int64
	StudentID, ProgramID, PeriodID, Reference, StartDate, EndDate, Status, Notes string
}
type studentFundingProgramOption struct {
	ID    int64
	Label string
}

func (h studentFundingProgramsHandlers) base(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(loc)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "student-funding-programs"}
}
func (h studentFundingProgramsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	programID := strings.TrimSpace(c.Query("program_id"))
	periodID := strings.TrimSpace(c.Query("period_id"))
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if parsePositiveID(programID) == 0 {
		programID = ""
	}
	if parsePositiveID(periodID) == 0 {
		periodID = ""
	}
	if status != "ACTIVE" && status != "COMPLETED" && status != "CANCELLED" {
		status = ""
	}
	q := `SELECT sfp.id,s.name,s.nim,fp.name,b.benefit_type,b.benefit_value,ct.name,COALESCE(ap.code,'Semua periode'),COALESCE(sfp.reference_number,''),DATE_FORMAT(sfp.start_date,'%d %b %Y'),COALESCE(DATE_FORMAT(sfp.end_date,'%d %b %Y'),'Tidak dibatasi'),sfp.status,(SELECT COUNT(*) FROM student_invoice_adjustments a WHERE a.student_funding_program_id=sfp.id) FROM student_funding_programs sfp JOIN students s ON s.id=sfp.student_id JOIN funding_programs fp ON fp.id=sfp.funding_program_id JOIN funding_program_benefits b ON b.funding_program_id=fp.id JOIN charge_types ct ON ct.id=b.charge_type_id LEFT JOIN academic_periods ap ON ap.id=sfp.academic_period_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (s.nim LIKE ? OR s.name LIKE ? OR fp.name LIKE ? OR sfp.reference_number LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if programID != "" {
		q += ` AND sfp.funding_program_id=?`
		args = append(args, programID)
	}
	if periodID != "" {
		q += ` AND sfp.academic_period_id=?`
		args = append(args, periodID)
	}
	if status != "" {
		q += ` AND sfp.status=?`
		args = append(args, status)
	}
	q += ` ORDER BY sfp.start_date DESC,sfp.id DESC`
	rows, e := h.db.QueryContext(c.Request.Context(), q, args...)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer rows.Close()
	items := []studentFundingProgramItem{}
	for rows.Next() {
		var x studentFundingProgramItem
		var typ string
		var value float64
		if rows.Scan(&x.ID, &x.Student, &x.NIM, &x.Program, &typ, &value, &x.ChargeType, &x.Period, &x.Reference, &x.StartDate, &x.EndDate, &x.Status, &x.AdjustmentCount) != nil {
			continue
		}
		x.Benefit = fundingBenefitLabel(typ, value)
		x.StatusLabel = studentFundingStatusLabel(x.Status)
		items = append(items, x)
	}
	var m studentFundingProgramMetrics
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(status='ACTIVE'),0),COALESCE(SUM(EXISTS(SELECT 1 FROM student_invoice_adjustments a WHERE a.student_funding_program_id=student_funding_programs.id)),0),COALESCE(SUM(NOT EXISTS(SELECT 1 FROM student_invoice_adjustments a WHERE a.student_funding_program_id=student_funding_programs.id)),0) FROM student_funding_programs`).Scan(&m.Total, &m.Active, &m.Applied, &m.Unapplied)
	d := h.base(p, "Penerima Program")
	d["Items"] = items
	d["Metrics"] = m
	d["Search"] = search
	d["Programs"] = h.options(c, `SELECT id,CONCAT(code,' - ',name) FROM funding_programs ORDER BY name`)
	d["Periods"] = h.options(c, `SELECT id,CONCAT(code,' - ',semester) FROM academic_periods ORDER BY start_date DESC`)
	d["SelectedProgram"] = programID
	d["SelectedPeriod"] = periodID
	d["SelectedStatus"] = status
	d["Result"] = c.Query("result")
	d["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-funding-programs.create")
	d["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("student-funding-programs.update")
	d["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("student-funding-programs.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-funding-programs/index.html", d)
}
func (h studentFundingProgramsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Penerima Program", studentFundingProgramForm{StartDate: time.Now().Format("2006-01-02"), Status: "ACTIVE"}, "")
}
func (h studentFundingProgramsHandlers) editForm(c *gin.Context) {
	f, e := h.find(c, c.Param("id"))
	if errors.Is(e, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	h.renderForm(c, 200, "Ubah Penerima Program", f, "")
}
func (h studentFundingProgramsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := studentFundingProgramFromRequest(c)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Penerima Program", f, msg)
		return
	}
	_, e := h.db.ExecContext(c.Request.Context(), `INSERT INTO student_funding_programs(student_id,funding_program_id,academic_period_id,reference_number,start_date,end_date,status,notes,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, f.StudentID, f.ProgramID, nullableID(f.PeriodID), nullableString(f.Reference), f.StartDate, nullableString(f.EndDate), f.Status, nullableString(f.Notes))
	if e != nil {
		h.renderForm(c, 422, "Tambah Penerima Program", f, e.Error())
		return
	}
	c.Redirect(303, "/student-funding-programs?result=created")
}
func (h studentFundingProgramsHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	f := studentFundingProgramFromRequest(c)
	f.ID = id
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Penerima Program", f, msg)
		return
	}
	r, e := h.db.ExecContext(c.Request.Context(), `UPDATE student_funding_programs SET student_id=?,funding_program_id=?,academic_period_id=?,reference_number=?,start_date=?,end_date=?,status=?,notes=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, f.StudentID, f.ProgramID, nullableID(f.PeriodID), nullableString(f.Reference), f.StartDate, nullableString(f.EndDate), f.Status, nullableString(f.Notes), id)
	if e != nil {
		h.renderForm(c, 422, "Ubah Penerima Program", f, e.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/student-funding-programs?result=updated")
}
func (h studentFundingProgramsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var refs int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoice_adjustments WHERE student_funding_program_id=?`, id).Scan(&refs)
	if refs > 0 {
		c.Redirect(303, "/student-funding-programs?result=in-use")
		return
	}
	r, e := h.db.ExecContext(c.Request.Context(), `DELETE FROM student_funding_programs WHERE id=?`, id)
	if e != nil {
		c.Redirect(303, "/student-funding-programs?result=delete-failed")
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/student-funding-programs?result=deleted")
}
func (h studentFundingProgramsHandlers) find(c *gin.Context, id any) (studentFundingProgramForm, error) {
	var f studentFundingProgramForm
	var period sql.NullInt64
	var reference, notes sql.NullString
	var start time.Time
	var end sql.NullTime
	e := h.db.QueryRowContext(c.Request.Context(), `SELECT id,student_id,funding_program_id,academic_period_id,reference_number,start_date,end_date,status,notes FROM student_funding_programs WHERE id=?`, id).Scan(&f.ID, &f.StudentID, &f.ProgramID, &period, &reference, &start, &end, &f.Status, &notes)
	if period.Valid {
		f.PeriodID = strconv.FormatInt(period.Int64, 10)
	}
	f.Reference = reference.String
	f.StartDate = start.Format("2006-01-02")
	if end.Valid {
		f.EndDate = end.Time.Format("2006-01-02")
	}
	f.Notes = notes.String
	return f, e
}
func studentFundingProgramFromRequest(c *gin.Context) studentFundingProgramForm {
	return studentFundingProgramForm{StudentID: c.PostForm("student_id"), ProgramID: c.PostForm("funding_program_id"), PeriodID: c.PostForm("academic_period_id"), Reference: strings.ToUpper(strings.TrimSpace(c.PostForm("reference_number"))), StartDate: c.PostForm("start_date"), EndDate: c.PostForm("end_date"), Status: strings.ToUpper(c.PostForm("status")), Notes: strings.TrimSpace(c.PostForm("notes"))}
}
func (h studentFundingProgramsHandlers) validate(c *gin.Context, f studentFundingProgramForm, id int64) string {
	start, e1 := time.Parse("2006-01-02", f.StartDate)
	var end time.Time
	var e2 error
	if f.EndDate != "" {
		end, e2 = time.Parse("2006-01-02", f.EndDate)
	}
	switch {
	case parsePositiveID(f.StudentID) == 0:
		return "Mahasiswa wajib dipilih."
	case parsePositiveID(f.ProgramID) == 0:
		return "Program bantuan wajib dipilih."
	case f.PeriodID != "" && parsePositiveID(f.PeriodID) == 0:
		return "Periode akademik tidak valid."
	case e1 != nil || e2 != nil || (!end.IsZero() && end.Before(start)):
		return "Masa berlaku penerima tidak valid."
	case f.Status != "ACTIVE" && f.Status != "COMPLETED" && f.Status != "CANCELLED":
		return "Status penerima tidak valid."
	case utf8.RuneCountInString(f.Reference) > 100:
		return "Nomor referensi maksimal 100 karakter."
	case utf8.RuneCountInString(f.Notes) > 500:
		return "Catatan maksimal 500 karakter."
	}
	var count int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM students WHERE id=?`, f.StudentID).Scan(&count)
	if count == 0 {
		return "Mahasiswa tidak ditemukan."
	}
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM funding_programs WHERE id=?`, f.ProgramID).Scan(&count)
	if count == 0 {
		return "Program bantuan tidak ditemukan."
	}
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_funding_programs WHERE student_id=? AND funding_program_id=? AND academic_period_id <=> ? AND id<>?`, f.StudentID, f.ProgramID, nullableID(f.PeriodID), id).Scan(&count)
	if count > 0 {
		return "Program yang sama sudah ditetapkan kepada mahasiswa pada periode ini."
	}
	return ""
}
func (h studentFundingProgramsHandlers) renderForm(c *gin.Context, status int, title string, f studentFundingProgramForm, msg string) {
	d := h.base(auth.CurrentPrincipal(c), title)
	d["Form"] = f
	d["Error"] = msg
	d["IsEdit"] = f.ID > 0
	d["Students"] = h.options(c, `SELECT id,CONCAT(nim,' - ',name) FROM students ORDER BY nim`)
	d["Programs"] = h.options(c, `SELECT id,CONCAT(code,' - ',name) FROM funding_programs WHERE is_active=1 OR id=`+sqlID(f.ProgramID)+` ORDER BY name`)
	d["Periods"] = h.options(c, `SELECT id,CONCAT(code,' - ',semester) FROM academic_periods ORDER BY start_date DESC`)
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "student-funding-programs/form.html", d)
}
func (h studentFundingProgramsHandlers) options(c *gin.Context, q string) []studentFundingProgramOption {
	rows, e := h.db.QueryContext(c.Request.Context(), q)
	if e != nil {
		return nil
	}
	defer rows.Close()
	out := []studentFundingProgramOption{}
	for rows.Next() {
		var x studentFundingProgramOption
		if rows.Scan(&x.ID, &x.Label) == nil {
			out = append(out, x)
		}
	}
	return out
}
func sqlID(v string) string { n := parsePositiveID(v); return strconv.FormatInt(n, 10) }
func studentFundingStatusLabel(v string) string {
	switch v {
	case "ACTIVE":
		return "Aktif"
	case "COMPLETED":
		return "Selesai"
	case "CANCELLED":
		return "Dibatalkan"
	}
	return v
}
