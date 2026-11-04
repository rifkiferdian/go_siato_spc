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

type fundingProgramsHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type fundingProgramItem struct {
	ID                                                                                                       int64
	Code, Name, ProgramType, ProgramTypeLabel, FundingSource, Benefit, ChargeType, Scope, StartDate, EndDate string
	MaximumUses                                                                                              int
	Active                                                                                                   bool
}
type fundingProgramMetrics struct{ Total, Active, Scholarship, Marketing int }
type fundingProgramForm struct {
	ID                                                                                                                                                                 int64
	Code, Name, ProgramType, FundingSource, StartDate, EndDate, Notes, ChargeTypeID, BenefitType, BenefitValue, MaximumAmount, ApplicationScope, MaximumUses, Priority string
	Active                                                                                                                                                             bool
}
type fundingProgramOption struct {
	ID    int64
	Label string
}

func (h fundingProgramsHandlers) base(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(loc)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "funding-programs"}
}
func (h fundingProgramsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	q := `SELECT fp.id,fp.code,fp.name,fp.program_type,fp.funding_source,DATE_FORMAT(fp.start_date,'%d %b %Y'),COALESCE(DATE_FORMAT(fp.end_date,'%d %b %Y'),'Tidak dibatasi'),fp.is_active,b.benefit_type,b.benefit_value,ct.name,b.application_scope,COALESCE(b.maximum_uses,0) FROM funding_programs fp JOIN funding_program_benefits b ON b.funding_program_id=fp.id JOIN charge_types ct ON ct.id=b.charge_type_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (fp.code LIKE ? OR fp.name LIKE ? OR fp.funding_source LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY fp.name`
	rows, e := h.db.QueryContext(c.Request.Context(), q, args...)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer rows.Close()
	items := []fundingProgramItem{}
	for rows.Next() {
		var x fundingProgramItem
		var typ string
		var value float64
		if rows.Scan(&x.ID, &x.Code, &x.Name, &x.ProgramType, &x.FundingSource, &x.StartDate, &x.EndDate, &x.Active, &typ, &value, &x.ChargeType, &x.Scope, &x.MaximumUses) != nil {
			continue
		}
		x.ProgramTypeLabel = fundingProgramTypeLabel(x.ProgramType)
		x.Benefit = fundingBenefitLabel(typ, value)
		x.Scope = fundingScopeLabel(x.Scope)
		items = append(items, x)
	}
	var m fundingProgramMetrics
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(is_active=1),0),COALESCE(SUM(program_type='SCHOLARSHIP'),0),COALESCE(SUM(program_type='MARKETING'),0) FROM funding_programs`).Scan(&m.Total, &m.Active, &m.Scholarship, &m.Marketing)
	d := h.base(p, "Program & Manfaat")
	d["Items"] = items
	d["Metrics"] = m
	d["Search"] = search
	d["Result"] = c.Query("result")
	d["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("funding-programs.create")
	d["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("funding-programs.update")
	d["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("funding-programs.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "funding-programs/index.html", d)
}
func (h fundingProgramsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Program & Manfaat", fundingProgramForm{ProgramType: "SCHOLARSHIP", StartDate: time.Now().Format("2006-01-02"), BenefitType: "PERCENTAGE", BenefitValue: "0", ApplicationScope: "PER_INVOICE", Priority: "100", Active: true}, "")
}
func (h fundingProgramsHandlers) editForm(c *gin.Context) {
	f, e := h.find(c, c.Param("id"))
	if errors.Is(e, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	h.renderForm(c, 200, "Ubah Program & Manfaat", f, "")
}
func (h fundingProgramsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := fundingProgramFromRequest(c)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Program & Manfaat", f, msg)
		return
	}
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer tx.Rollback()
	r, e := tx.ExecContext(c.Request.Context(), `INSERT INTO funding_programs(code,name,program_type,funding_source,start_date,end_date,is_active,notes,created_at,updated_at) VALUES(?,?,?,?,?,?,?, ?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, f.Code, f.Name, f.ProgramType, f.FundingSource, f.StartDate, nullableString(f.EndDate), f.Active, nullableString(f.Notes))
	if e != nil {
		h.renderForm(c, 422, "Tambah Program & Manfaat", f, e.Error())
		return
	}
	id, _ := r.LastInsertId()
	_, e = tx.ExecContext(c.Request.Context(), `INSERT INTO funding_program_benefits(funding_program_id,charge_type_id,benefit_type,benefit_value,maximum_amount,created_at,updated_at,application_scope,maximum_uses,priority) VALUES(?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?,?,?)`, id, f.ChargeTypeID, f.BenefitType, f.BenefitValue, nullableDecimal(f.MaximumAmount), f.ApplicationScope, nullableInt(f.MaximumUses), f.Priority)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/funding-programs?result=created")
}
func (h fundingProgramsHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	f := fundingProgramFromRequest(c)
	f.ID = id
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Program & Manfaat", f, msg)
		return
	}
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer tx.Rollback()
	r, e := tx.ExecContext(c.Request.Context(), `UPDATE funding_programs SET code=?,name=?,program_type=?,funding_source=?,start_date=?,end_date=?,is_active=?,notes=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, f.Code, f.Name, f.ProgramType, f.FundingSource, f.StartDate, nullableString(f.EndDate), f.Active, nullableString(f.Notes), id)
	if e != nil {
		h.renderForm(c, 422, "Ubah Program & Manfaat", f, e.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	_, e = tx.ExecContext(c.Request.Context(), `UPDATE funding_program_benefits SET charge_type_id=?,benefit_type=?,benefit_value=?,maximum_amount=?,application_scope=?,maximum_uses=?,priority=?,updated_at=UTC_TIMESTAMP(6) WHERE funding_program_id=?`, f.ChargeTypeID, f.BenefitType, f.BenefitValue, nullableDecimal(f.MaximumAmount), f.ApplicationScope, nullableInt(f.MaximumUses), f.Priority, id)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/funding-programs?result=updated")
}
func (h fundingProgramsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	tx, e := h.db.BeginTx(c.Request.Context(), nil)
	if e != nil {
		c.AbortWithError(500, e)
		return
	}
	defer tx.Rollback()
	id := c.Param("id")
	var refs int
	_ = tx.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM student_funding_programs WHERE funding_program_id=?)+(SELECT COUNT(*) FROM payments WHERE funding_program_id=?)`, id, id).Scan(&refs)
	if refs > 0 {
		c.Redirect(303, "/funding-programs?result=in-use")
		return
	}
	if _, e = tx.ExecContext(c.Request.Context(), `DELETE FROM funding_program_benefits WHERE funding_program_id=?`, id); e == nil {
		_, e = tx.ExecContext(c.Request.Context(), `DELETE FROM funding_programs WHERE id=?`, id)
	}
	if e != nil {
		c.Redirect(303, "/funding-programs?result=delete-failed")
		return
	}
	if tx.Commit() != nil {
		c.AbortWithStatus(500)
		return
	}
	c.Redirect(303, "/funding-programs?result=deleted")
}
func (h fundingProgramsHandlers) find(c *gin.Context, id any) (fundingProgramForm, error) {
	var f fundingProgramForm
	var end, notes, maxAmount sql.NullString
	var maxUses sql.NullInt64
	var value float64
	var priority int
	var start time.Time
	var endDate sql.NullTime
	e := h.db.QueryRowContext(c.Request.Context(), `SELECT fp.id,fp.code,fp.name,fp.program_type,fp.funding_source,fp.start_date,fp.end_date,fp.is_active,fp.notes,b.charge_type_id,b.benefit_type,b.benefit_value,b.maximum_amount,b.application_scope,b.maximum_uses,b.priority FROM funding_programs fp JOIN funding_program_benefits b ON b.funding_program_id=fp.id WHERE fp.id=? LIMIT 1`, id).Scan(&f.ID, &f.Code, &f.Name, &f.ProgramType, &f.FundingSource, &start, &endDate, &f.Active, &notes, &f.ChargeTypeID, &f.BenefitType, &value, &maxAmount, &f.ApplicationScope, &maxUses, &priority)
	_ = end
	f.StartDate = start.Format("2006-01-02")
	if endDate.Valid {
		f.EndDate = endDate.Time.Format("2006-01-02")
	}
	f.Notes = notes.String
	f.BenefitValue = strconv.FormatFloat(value, 'f', 2, 64)
	f.MaximumAmount = maxAmount.String
	if maxUses.Valid {
		f.MaximumUses = strconv.FormatInt(maxUses.Int64, 10)
	}
	f.Priority = strconv.Itoa(priority)
	return f, e
}
func fundingProgramFromRequest(c *gin.Context) fundingProgramForm {
	return fundingProgramForm{Code: strings.ToUpper(strings.TrimSpace(c.PostForm("code"))), Name: strings.TrimSpace(c.PostForm("name")), ProgramType: strings.ToUpper(c.PostForm("program_type")), FundingSource: strings.TrimSpace(c.PostForm("funding_source")), StartDate: c.PostForm("start_date"), EndDate: c.PostForm("end_date"), Notes: strings.TrimSpace(c.PostForm("notes")), ChargeTypeID: c.PostForm("charge_type_id"), BenefitType: strings.ToUpper(c.PostForm("benefit_type")), BenefitValue: c.PostForm("benefit_value"), MaximumAmount: c.PostForm("maximum_amount"), ApplicationScope: strings.ToUpper(c.PostForm("application_scope")), MaximumUses: c.PostForm("maximum_uses"), Priority: c.PostForm("priority"), Active: c.PostForm("is_active") == "1"}
}
func (h fundingProgramsHandlers) validate(c *gin.Context, f fundingProgramForm, id int64) string {
	value, e := strconv.ParseFloat(f.BenefitValue, 64)
	_, eStart := time.Parse("2006-01-02", f.StartDate)
	var end time.Time
	var eEnd error
	if f.EndDate != "" {
		end, eEnd = time.Parse("2006-01-02", f.EndDate)
	}
	start, _ := time.Parse("2006-01-02", f.StartDate)
	validProgram := f.ProgramType == "SCHOLARSHIP" || f.ProgramType == "EMPLOYEE_SUBSIDY" || f.ProgramType == "MARKETING"
	validBenefit := f.BenefitType == "PERCENTAGE" || f.BenefitType == "FIXED_AMOUNT" || f.BenefitType == "SPECIAL_RATE" || f.BenefitType == "WAIVER" || f.BenefitType == "THIRD_PARTY_PAYMENT"
	validScope := f.ApplicationScope == "PER_INVOICE" || f.ApplicationScope == "PER_PERIOD" || f.ApplicationScope == "ONCE"
	switch {
	case f.Code == "" || utf8.RuneCountInString(f.Code) > 50:
		return "Kode program wajib diisi dan maksimal 50 karakter."
	case f.Name == "" || utf8.RuneCountInString(f.Name) > 150:
		return "Nama program wajib diisi dan maksimal 150 karakter."
	case !validProgram:
		return "Jenis program tidak valid."
	case f.FundingSource == "":
		return "Sumber dana wajib diisi."
	case eStart != nil || eEnd != nil || (!end.IsZero() && end.Before(start)):
		return "Masa berlaku program tidak valid."
	case parsePositiveID(f.ChargeTypeID) == 0:
		return "Jenis biaya wajib dipilih."
	case !validBenefit || e != nil || value < 0:
		return "Bentuk atau nilai manfaat tidak valid."
	case f.BenefitType == "PERCENTAGE" && value > 100:
		return "Persentase manfaat maksimal 100%."
	case !validScope:
		return "Cakupan penerapan tidak valid."
	case parsePositiveID(f.Priority) == 0:
		return "Prioritas harus lebih dari nol."
	case utf8.RuneCountInString(f.Notes) > 500:
		return "Catatan maksimal 500 karakter."
	}
	var count int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM funding_programs WHERE code=? AND id<>?`, f.Code, id).Scan(&count)
	if count > 0 {
		return "Kode program sudah digunakan."
	}
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM charge_types WHERE id=?`, f.ChargeTypeID).Scan(&count)
	if count == 0 {
		return "Jenis biaya tidak ditemukan."
	}
	return ""
}
func (h fundingProgramsHandlers) renderForm(c *gin.Context, status int, title string, f fundingProgramForm, msg string) {
	d := h.base(auth.CurrentPrincipal(c), title)
	d["Form"] = f
	d["Error"] = msg
	d["IsEdit"] = f.ID > 0
	rows, e := h.db.QueryContext(c.Request.Context(), `SELECT id,CONCAT(code,' - ',name) FROM charge_types WHERE is_active=1 OR id=? ORDER BY name`, nullableID(f.ChargeTypeID))
	opts := []fundingProgramOption{}
	if e == nil {
		defer rows.Close()
		for rows.Next() {
			var x fundingProgramOption
			if rows.Scan(&x.ID, &x.Label) == nil {
				opts = append(opts, x)
			}
		}
	}
	d["ChargeTypes"] = opts
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "funding-programs/form.html", d)
}
func fundingProgramTypeLabel(v string) string {
	switch v {
	case "SCHOLARSHIP":
		return "Beasiswa"
	case "EMPLOYEE_SUBSIDY":
		return "Anak Karyawan"
	case "MARKETING":
		return "Marketing"
	}
	return v
}
func fundingBenefitLabel(t string, v float64) string {
	switch t {
	case "PERCENTAGE":
		return strconv.FormatFloat(v, 'f', 0, 64) + "%"
	case "FIXED_AMOUNT":
		return "Potongan " + formatRupiah(v)
	case "SPECIAL_RATE":
		return "Tarif " + formatRupiah(v)
	case "WAIVER":
		return "Dibebaskan 100%"
	case "THIRD_PARTY_PAYMENT":
		return "Ditanggung " + formatRupiah(v)
	}
	return t
}
func fundingScopeLabel(v string) string {
	switch v {
	case "PER_INVOICE":
		return "Setiap tagihan"
	case "PER_PERIOD":
		return "Per semester"
	case "ONCE":
		return "Sekali saja"
	}
	return v
}
