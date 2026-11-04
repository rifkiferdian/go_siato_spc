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

type billingPoliciesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type billingPolicyItem struct {
	ID                                                                                                  int64
	Code, Name, ChargeType, Period, Program, Category, CohortClass, PaymentMethod, PaymentLabel, Amount string
	CohortYear, InstallmentCount, ScheduleCount                                                         int
	IsActive                                                                                            bool
}
type billingPolicyMetrics struct{ Total, Active, Monthly, Custom int }
type billingPolicyOption struct {
	ID    int64
	Label string
}
type billingPolicyForm struct {
	ID                                                                                                                                      int64
	Code, Name, ChargeTypeID, PeriodID, ProgramID, CategoryID, CohortYear, CohortClass, PaymentMethod, TotalAmount, InstallmentCount, Notes string
	IsActive                                                                                                                                bool
}

func (h billingPoliciesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	period := c.Query("academic_period_id")
	program := c.Query("study_program_id")
	category := c.Query("student_category_id")
	year := c.Query("cohort_year")
	class := strings.TrimSpace(c.Query("cohort_year_class"))
	q := `SELECT bp.id,bp.code,bp.name,ct.name,ap.code,COALESCE(sp.name,''),COALESCE(sc.name,''),COALESCE(bp.cohort_year,0),COALESCE(bp.cohort_year_class,''),bp.payment_method,bp.total_amount,bp.installment_count,COUNT(bps.id),bp.is_active FROM billing_policies bp JOIN charge_types ct ON ct.id=bp.charge_type_id JOIN academic_periods ap ON ap.id=bp.academic_period_id LEFT JOIN study_programs sp ON sp.id=bp.study_program_id LEFT JOIN student_categories sc ON sc.id=bp.student_category_id LEFT JOIN billing_policy_schedules bps ON bps.billing_policy_id=bp.id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (bp.code LIKE ? OR bp.name LIKE ? OR bp.payment_method LIKE ? OR ct.name LIKE ? OR ap.code LIKE ? OR sp.name LIKE ? OR sc.name LIKE ? OR bp.cohort_year_class LIKE ?)`
		like := "%" + search + "%"
		for i := 0; i < 8; i++ {
			args = append(args, like)
		}
	}
	if parsePositiveID(period) > 0 {
		q += ` AND bp.academic_period_id=?`
		args = append(args, period)
	}
	if parsePositiveID(program) > 0 {
		q += ` AND bp.study_program_id=?`
		args = append(args, program)
	}
	if parsePositiveID(category) > 0 {
		q += ` AND bp.student_category_id=?`
		args = append(args, category)
	}
	if parsePositiveID(year) > 0 {
		q += ` AND bp.cohort_year=?`
		args = append(args, year)
	}
	if class != "" {
		q += ` AND bp.cohort_year_class=?`
		args = append(args, class)
	}
	q += ` GROUP BY bp.id,bp.code,bp.name,ct.name,ap.code,sp.name,sc.name,bp.cohort_year,bp.cohort_year_class,bp.payment_method,bp.total_amount,bp.installment_count,bp.is_active ORDER BY ap.start_date DESC,bp.name`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []billingPolicyItem{}
	for rows.Next() {
		var x billingPolicyItem
		var amount float64
		if err = rows.Scan(&x.ID, &x.Code, &x.Name, &x.ChargeType, &x.Period, &x.Program, &x.Category, &x.CohortYear, &x.CohortClass, &x.PaymentMethod, &amount, &x.InstallmentCount, &x.ScheduleCount, &x.IsActive); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Amount = formatRupiah(amount)
		x.PaymentLabel = paymentMethodLabel(x.PaymentMethod)
		items = append(items, x)
	}
	var metrics billingPolicyMetrics
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(is_active=1),0),COALESCE(SUM(payment_method='MONTHLY'),0),COALESCE(SUM(payment_method='CUSTOM'),0) FROM billing_policies`).Scan(&metrics.Total, &metrics.Active, &metrics.Monthly, &metrics.Custom); err != nil {
		c.AbortWithError(500, err)
		return
	}
	chargeTypes, periods, programs, categories := h.options(c)
	years, classes := h.filterOptions(c)
	data := h.baseData(p, "Tarif & Kebijakan")
	data["Items"] = items
	data["Metrics"] = metrics
	data["ChargeTypes"] = chargeTypes
	data["Periods"] = periods
	data["Programs"] = programs
	data["Categories"] = categories
	data["Years"] = years
	data["Classes"] = classes
	data["Search"] = search
	data["SelectedPeriod"] = period
	data["SelectedProgram"] = program
	data["SelectedCategory"] = category
	data["SelectedYear"] = year
	data["SelectedClass"] = class
	data["Status"] = c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("billing-policies.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("billing-policies.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("billing-policies.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "billing-policies/index.html", data)
}
func (h billingPoliciesHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Kebijakan", billingPolicyForm{PaymentMethod: "MONTHLY", InstallmentCount: "1", IsActive: true}, "")
}
func (h billingPoliciesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := billingPolicyFormFromRequest(c)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Kebijakan", f, msg)
		return
	}
	_, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO billing_policies(code,name,charge_type_id,academic_period_id,study_program_id,student_category_id,cohort_year,cohort_year_class,payment_method,total_amount,installment_count,notes,is_active,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, f.Code, f.Name, f.ChargeTypeID, f.PeriodID, nullableID(f.ProgramID), nullableID(f.CategoryID), nullableInt(f.CohortYear), nullableString(f.CohortClass), f.PaymentMethod, f.TotalAmount, f.InstallmentCount, nullableString(f.Notes), f.IsActive)
	if err != nil {
		h.renderForm(c, 422, "Tambah Kebijakan", f, err.Error())
		return
	}
	c.Redirect(303, "/billing-policies?status=created")
}
func (h billingPoliciesHandlers) editForm(c *gin.Context) {
	f, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Kebijakan", f, "")
}
func (h billingPoliciesHandlers) update(c *gin.Context) {
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
	f := billingPolicyFormFromRequest(c)
	f.ID = id
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Kebijakan", f, msg)
		return
	}
	r, err := h.db.ExecContext(c.Request.Context(), `UPDATE billing_policies SET code=?,name=?,charge_type_id=?,academic_period_id=?,study_program_id=?,student_category_id=?,cohort_year=?,cohort_year_class=?,payment_method=?,total_amount=?,installment_count=?,notes=?,is_active=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, f.Code, f.Name, f.ChargeTypeID, f.PeriodID, nullableID(f.ProgramID), nullableID(f.CategoryID), nullableInt(f.CohortYear), nullableString(f.CohortClass), f.PaymentMethod, f.TotalAmount, f.InstallmentCount, nullableString(f.Notes), f.IsActive, id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Kebijakan", f, err.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		var exists int
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM billing_policies WHERE id=?`, id).Scan(&exists)
		if exists == 0 {
			c.AbortWithStatus(404)
			return
		}
	}
	c.Redirect(303, "/billing-policies?status=updated")
}
func (h billingPoliciesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var refs int
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM student_billing_policies WHERE billing_policy_id=?)+(SELECT COUNT(*) FROM student_invoices WHERE student_billing_policy_id=?)+(SELECT COUNT(*) FROM student_billing_policy_batches WHERE billing_policy_id=?)`, id, id, id).Scan(&refs)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if refs > 0 {
		c.Redirect(303, "/billing-policies?status=in-use")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `DELETE FROM billing_policy_schedules WHERE billing_policy_id=?`, id); err == nil {
		_, err = tx.ExecContext(c.Request.Context(), `DELETE FROM billing_policies WHERE id=?`, id)
	}
	if err != nil {
		_ = tx.Rollback()
		c.Redirect(303, "/billing-policies?status=delete-failed")
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, "/billing-policies?status=deleted")
}

func (h billingPoliciesHandlers) find(c *gin.Context, id any) (billingPolicyForm, error) {
	var f billingPolicyForm
	var program, category, year sql.NullInt64
	var class, notes sql.NullString
	var amount float64
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,code,name,charge_type_id,academic_period_id,study_program_id,student_category_id,cohort_year,cohort_year_class,payment_method,total_amount,installment_count,notes,is_active FROM billing_policies WHERE id=?`, id).Scan(&f.ID, &f.Code, &f.Name, &f.ChargeTypeID, &f.PeriodID, &program, &category, &year, &class, &f.PaymentMethod, &amount, &f.InstallmentCount, &notes, &f.IsActive)
	if program.Valid {
		f.ProgramID = strconv.FormatInt(program.Int64, 10)
	}
	if category.Valid {
		f.CategoryID = strconv.FormatInt(category.Int64, 10)
	}
	if year.Valid {
		f.CohortYear = strconv.FormatInt(year.Int64, 10)
	}
	f.CohortClass = class.String
	f.Notes = notes.String
	f.TotalAmount = strconv.FormatFloat(amount, 'f', 2, 64)
	return f, err
}
func billingPolicyFormFromRequest(c *gin.Context) billingPolicyForm {
	return billingPolicyForm{Code: strings.ToUpper(strings.TrimSpace(c.PostForm("code"))), Name: strings.TrimSpace(c.PostForm("name")), ChargeTypeID: c.PostForm("charge_type_id"), PeriodID: c.PostForm("academic_period_id"), ProgramID: c.PostForm("study_program_id"), CategoryID: c.PostForm("student_category_id"), CohortYear: strings.TrimSpace(c.PostForm("cohort_year")), CohortClass: strings.ToUpper(strings.TrimSpace(c.PostForm("cohort_year_class"))), PaymentMethod: strings.ToUpper(strings.TrimSpace(c.PostForm("payment_method"))), TotalAmount: strings.TrimSpace(c.PostForm("total_amount")), InstallmentCount: strings.TrimSpace(c.PostForm("installment_count")), Notes: strings.TrimSpace(c.PostForm("notes")), IsActive: c.PostForm("is_active") == "1"}
}
func (h billingPoliciesHandlers) validate(c *gin.Context, f billingPolicyForm, id int64) string {
	amount, e1 := strconv.ParseFloat(f.TotalAmount, 64)
	installments, e2 := strconv.Atoi(f.InstallmentCount)
	year := 0
	var e3 error
	if f.CohortYear != "" {
		year, e3 = strconv.Atoi(f.CohortYear)
	}
	switch {
	case f.Code == "" || utf8.RuneCountInString(f.Code) > 50:
		return "Kode kebijakan wajib diisi dan maksimal 50 karakter."
	case f.Name == "" || utf8.RuneCountInString(f.Name) > 150:
		return "Nama kebijakan wajib diisi dan maksimal 150 karakter."
	case parsePositiveID(f.ChargeTypeID) == 0 || parsePositiveID(f.PeriodID) == 0:
		return "Jenis biaya dan periode akademik wajib dipilih."
	case f.ProgramID != "" && parsePositiveID(f.ProgramID) == 0:
		return "Program studi tidak valid."
	case f.CategoryID != "" && parsePositiveID(f.CategoryID) == 0:
		return "Golongan mahasiswa tidak valid."
	case e3 != nil || (f.CohortYear != "" && (year < 2000 || year > 2100)):
		return "Tahun angkatan harus berada antara 2000 dan 2100."
	case utf8.RuneCountInString(f.CohortClass) > 30:
		return "Kelas angkatan maksimal 30 karakter."
	case !validPaymentMethod(f.PaymentMethod):
		return "Metode pembayaran tidak valid."
	case e1 != nil || amount < 0:
		return "Total tarif harus berupa angka nol atau lebih."
	case e2 != nil || installments < 1 || installments > 120:
		return "Jumlah termin harus antara 1 dan 120."
	case utf8.RuneCountInString(f.Notes) > 500:
		return "Catatan maksimal 500 karakter."
	}
	var count int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM billing_policies WHERE code=? AND id<>?`, f.Code, id).Scan(&count); err != nil {
		return "Gagal memvalidasi kode kebijakan."
	}
	if count > 0 {
		return "Kode kebijakan sudah digunakan."
	}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM charge_types WHERE id=?)+(SELECT COUNT(*) FROM academic_periods WHERE id=?)`, f.ChargeTypeID, f.PeriodID).Scan(&count); err != nil || count != 2 {
		return "Jenis biaya atau periode akademik tidak ditemukan."
	}
	for _, check := range []struct{ id, table string }{{f.ProgramID, "study_programs"}, {f.CategoryID, "student_categories"}} {
		if check.id != "" {
			if err := h.db.QueryRowContext(c.Request.Context(), fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE id=?`, check.table), check.id).Scan(&count); err != nil || count == 0 {
				return "Sasaran kebijakan tidak ditemukan."
			}
		}
	}
	if id > 0 {
		var scheduleCount, maxSequence int
		var scheduleAmount float64
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(MAX(sequence_no),0),COALESCE(SUM(amount),0) FROM billing_policy_schedules WHERE billing_policy_id=?`, id).Scan(&scheduleCount, &maxSequence, &scheduleAmount)
		if installments < scheduleCount || installments < maxSequence {
			return fmt.Sprintf("Jumlah termin tidak boleh kurang dari %d karena jadwal sudah tersedia.", max(scheduleCount, maxSequence))
		}
		if amount < scheduleAmount {
			return fmt.Sprintf("Total tarif tidak boleh kurang dari total jadwal (%s).", formatRupiah(scheduleAmount))
		}
	}
	return ""
}
func (h billingPoliciesHandlers) options(c *gin.Context) ([]billingPolicyOption, []billingPolicyOption, []billingPolicyOption, []billingPolicyOption) {
	load := func(q string) []billingPolicyOption {
		rows, err := h.db.QueryContext(c.Request.Context(), q)
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
	return load(`SELECT id,CONCAT(code,' - ',name) FROM charge_types ORDER BY name`), load(`SELECT id,CONCAT(academic_year,' - ',semester) FROM academic_periods ORDER BY start_date DESC`), load(`SELECT id,CONCAT(code,' - ',name) FROM study_programs ORDER BY name`), load(`SELECT id,CONCAT(code,' - ',name) FROM student_categories ORDER BY name`)
}
func (h billingPoliciesHandlers) filterOptions(c *gin.Context) ([]string, []string) {
	load := func(q string) []string {
		rows, err := h.db.QueryContext(c.Request.Context(), q)
		if err != nil {
			return nil
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var x string
			if rows.Scan(&x) == nil {
				out = append(out, x)
			}
		}
		return out
	}
	return load(`SELECT DISTINCT cohort_year FROM billing_policies WHERE cohort_year IS NOT NULL ORDER BY cohort_year DESC`), load(`SELECT DISTINCT cohort_year_class FROM billing_policies WHERE cohort_year_class IS NOT NULL AND cohort_year_class<>'' ORDER BY cohort_year_class`)
}
func (h billingPoliciesHandlers) renderForm(c *gin.Context, status int, title string, f billingPolicyForm, message string) {
	chargeTypes, periods, programs, categories := h.options(c)
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["Form"] = f
	data["ChargeTypes"] = chargeTypes
	data["Periods"] = periods
	data["Programs"] = programs
	data["Categories"] = categories
	data["IsEdit"] = f.ID > 0
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "billing-policies/form.html", data)
}
func (h billingPoliciesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "billing-policies"}
}
func nullableID(v string) any {
	if parsePositiveID(v) == 0 {
		return nil
	}
	return v
}
func nullableInt(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func validPaymentMethod(v string) bool {
	return v == "MONTHLY" || v == "MILESTONE" || v == "INSTALLMENT" || v == "ONE_TIME" || v == "CUSTOM"
}
func paymentMethodLabel(v string) string {
	switch v {
	case "MONTHLY":
		return "Bulanan"
	case "MILESTONE":
		return "Tahapan UTS/UAS"
	case "INSTALLMENT":
		return "Cicilan"
	case "ONE_TIME":
		return "Sekali Bayar"
	case "CUSTOM":
		return "Custom"
	}
	return v
}
