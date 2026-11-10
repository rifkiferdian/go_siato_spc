package httpserver

import (
	"database/sql"
	"errors"
	"fmt"
	stdhtml "html"
	"html/template"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studentsHandlers struct {
	cfg config.Config
	db  *sql.DB
}

type studentItem struct {
	ID                                                                                                int64
	NIM, Name, ProgramCode, ProgramName, CategoryName, CohortClass, Status, StatusLabel, Email, Phone string
	CohortYear                                                                                        int
}
type studentOption struct {
	ID    int64
	Label string
}
type studentMetrics struct{ Total, Active, OnLeave, Graduated int }
type studentInvoice struct {
	Number, Description, DueDate, Status, StatusLabel string
	Amount, Paid                                      float64
}
type studentPaymentHistory struct {
	ID                                                           int64
	Number, Date, Method, Reference, Amount, Status, StatusLabel string
	Purpose                                                      template.HTML
	AllocationCount                                              int
}
type studentFundingHistory struct {
	ID                                                                                int64
	Program, Code, Benefit, ChargeType, Period, Reference, Dates, Status, StatusLabel string
	AdjustmentCount                                                                   int
	AdjustmentTotal                                                                   string
}
type studentForm struct {
	ID                                                                              int64
	NIM, Name, ProgramID, CategoryID, CohortYear, CohortClass, Status, Email, Phone string
}

func (h studentsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	year := strings.TrimSpace(c.Query("cohort_year"))
	class := strings.TrimSpace(c.Query("cohort_class"))
	program := strings.TrimSpace(c.Query("study_program_id"))
	category := strings.TrimSpace(c.Query("student_category_id"))
	query := `SELECT s.id,s.nim,s.name,sp.code,sp.name,COALESCE(sc.name,''),s.cohort_year,COALESCE(s.cohort_year_class,''),s.status,COALESCE(s.email,''),COALESCE(s.phone_number,'') FROM students s JOIN study_programs sp ON sp.id=s.study_program_id LEFT JOIN student_categories sc ON sc.id=s.student_category_id WHERE 1=1`
	args := []any{}
	if search != "" {
		query += ` AND (s.nim LIKE ? OR s.name LIKE ?)`
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if parsePositiveID(year) > 0 {
		query += ` AND s.cohort_year=?`
		args = append(args, year)
	}
	if class != "" {
		query += ` AND s.cohort_year_class=?`
		args = append(args, class)
	}
	if parsePositiveID(program) > 0 {
		query += ` AND s.study_program_id=?`
		args = append(args, program)
	}
	if parsePositiveID(category) > 0 {
		query += ` AND s.student_category_id=?`
		args = append(args, category)
	}
	query += ` ORDER BY s.cohort_year DESC,s.nim`
	rows, err := h.db.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []studentItem{}
	for rows.Next() {
		var x studentItem
		if err := rows.Scan(&x.ID, &x.NIM, &x.Name, &x.ProgramCode, &x.ProgramName, &x.CategoryName, &x.CohortYear, &x.CohortClass, &x.Status, &x.Email, &x.Phone); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.StatusLabel = studentStatusLabel(x.Status)
		items = append(items, x)
	}
	var metrics studentMetrics
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(status='ACTIVE'),0),COALESCE(SUM(status='ON_LEAVE'),0),COALESCE(SUM(status='GRADUATED'),0) FROM students`).Scan(&metrics.Total, &metrics.Active, &metrics.OnLeave, &metrics.Graduated); err != nil {
		c.AbortWithError(500, err)
		return
	}
	programs, categories := h.options(c)
	years, classes := h.filterOptions(c)
	data := h.baseData(p, "Mahasiswa")
	data["Items"] = items
	data["Metrics"] = metrics
	data["Programs"] = programs
	data["Categories"] = categories
	data["Years"] = years
	data["Classes"] = classes
	data["Search"] = search
	data["SelectedYear"] = year
	data["SelectedClass"] = class
	data["SelectedProgram"] = program
	data["SelectedCategory"] = category
	data["Status"] = c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("students.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("students.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("students.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "students/index.html", data)
}

func (h studentsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Mahasiswa", studentForm{Status: "ACTIVE"}, "")
}
func (h studentsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	form := studentFormFromRequest(c)
	if msg := h.validate(c, form, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Mahasiswa", form, msg)
		return
	}
	_, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO students(nim,name,study_program_id,cohort_year,status,email,phone_number,created_at,updated_at,student_category_id,cohort_year_class) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),?,?)`, form.NIM, form.Name, form.ProgramID, form.CohortYear, form.Status, nullableString(form.Email), nullableString(form.Phone), form.CategoryID, nullableString(form.CohortClass))
	if err != nil {
		h.renderForm(c, 422, "Tambah Mahasiswa", form, err.Error())
		return
	}
	c.Redirect(303, "/students?status=created")
}
func (h studentsHandlers) editForm(c *gin.Context) {
	form, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Mahasiswa", form, "")
}
func (h studentsHandlers) update(c *gin.Context) {
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
	form := studentFormFromRequest(c)
	form.ID = id
	if msg := h.validate(c, form, id); msg != "" {
		h.renderForm(c, 422, "Ubah Mahasiswa", form, msg)
		return
	}
	r, err := h.db.ExecContext(c.Request.Context(), `UPDATE students SET nim=?,name=?,study_program_id=?,cohort_year=?,status=?,email=?,phone_number=?,student_category_id=?,cohort_year_class=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, form.NIM, form.Name, form.ProgramID, form.CohortYear, form.Status, nullableString(form.Email), nullableString(form.Phone), form.CategoryID, nullableString(form.CohortClass), id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Mahasiswa", form, err.Error())
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, fmt.Sprintf("/students/%d?status=updated", id))
}
func (h studentsHandlers) details(c *gin.Context) {
	form, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	var item studentItem
	err = h.db.QueryRowContext(c.Request.Context(), `SELECT s.id,s.nim,s.name,sp.code,sp.name,COALESCE(sc.name,''),s.cohort_year,COALESCE(s.cohort_year_class,''),s.status,COALESCE(s.email,''),COALESCE(s.phone_number,'') FROM students s JOIN study_programs sp ON sp.id=s.study_program_id LEFT JOIN student_categories sc ON sc.id=s.student_category_id WHERE s.id=?`, form.ID).Scan(&item.ID, &item.NIM, &item.Name, &item.ProgramCode, &item.ProgramName, &item.CategoryName, &item.CohortYear, &item.CohortClass, &item.Status, &item.Email, &item.Phone)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	item.StatusLabel = studentStatusLabel(item.Status)
	tab := strings.ToLower(strings.TrimSpace(c.Query("tab")))
	if tab != "payments" && tab != "funding" {
		tab = "summary"
	}
	selectedPeriod := strings.TrimSpace(c.Query("academic_period_id"))
	var periodID int64
	if selectedPeriod == "all" {
		selectedPeriod = "all"
	} else if parsed := parsePositiveID(selectedPeriod); parsed > 0 {
		var exists int
		if h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM academic_periods WHERE id=?`, parsed).Scan(&exists) == nil && exists == 1 {
			periodID = parsed
		} else {
			selectedPeriod = ""
		}
	}
	if selectedPeriod == "" {
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(MAX(id),0) FROM academic_periods WHERE is_active=1`).Scan(&periodID)
		if periodID > 0 {
			selectedPeriod = strconv.FormatInt(periodID, 10)
		} else {
			selectedPeriod = "all"
		}
	}
	periods := []studentOption{}
	periodRows, periodErr := h.db.QueryContext(c.Request.Context(), `SELECT id,CONCAT(academic_year,' / ',semester,' · ',code) FROM academic_periods ORDER BY start_date DESC`)
	if periodErr != nil {
		c.AbortWithError(500, periodErr)
		return
	}
	for periodRows.Next() {
		var option studentOption
		if periodRows.Scan(&option.ID, &option.Label) == nil {
			periods = append(periods, option)
		}
	}
	periodRows.Close()

	periodWhere := ""
	periodArgs := []any{item.ID}
	if periodID > 0 {
		periodWhere = " AND academic_period_id=?"
		periodArgs = append(periodArgs, periodID)
	}
	var invoiceCount, paymentCount, fundingCount int
	var billed, paid float64
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(final_amount),0),COALESCE(SUM(paid_amount),0) FROM student_invoices WHERE student_id=?`+periodWhere, periodArgs...).Scan(&invoiceCount, &billed, &paid); err != nil {
		c.AbortWithError(500, err)
		return
	}
	paymentCountQuery := `SELECT COUNT(DISTINCT p.id) FROM payments p JOIN payment_allocations pa ON pa.payment_id=p.id JOIN student_invoices i ON i.id=pa.student_invoice_id WHERE p.student_id=?`
	fundingCountQuery := `SELECT COUNT(*) FROM student_funding_programs WHERE student_id=?`
	countArgs := []any{item.ID}
	if periodID > 0 {
		paymentCountQuery += ` AND i.academic_period_id=?`
		fundingCountQuery += ` AND academic_period_id=?`
		countArgs = append(countArgs, periodID)
	}
	_ = h.db.QueryRowContext(c.Request.Context(), paymentCountQuery, countArgs...).Scan(&paymentCount)
	_ = h.db.QueryRowContext(c.Request.Context(), fundingCountQuery, countArgs...).Scan(&fundingCount)

	invoiceQuery := `SELECT invoice_number,description,DATE_FORMAT(due_date,'%d/%m/%Y'),status,final_amount,paid_amount FROM student_invoices WHERE student_id=?` + periodWhere + ` ORDER BY due_date DESC,created_at DESC LIMIT 50`
	rows, err := h.db.QueryContext(c.Request.Context(), invoiceQuery, periodArgs...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	invoices := []studentInvoice{}
	for rows.Next() {
		var x studentInvoice
		if err = rows.Scan(&x.Number, &x.Description, &x.DueDate, &x.Status, &x.Amount, &x.Paid); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.StatusLabel = invoiceStatusLabel(x.Status)
		invoices = append(invoices, x)
	}
	rows.Close()

	paymentQuery := `SELECT p.id,p.payment_number,GROUP_CONCAT(DISTINCT CONCAT(i.description,' — ',ct.name) ORDER BY i.due_date SEPARATOR '||'),DATE_FORMAT(p.payment_date,'%d/%m/%Y %H:%i'),p.payment_method,COALESCE(p.reference_number,''),SUM(pa.amount),p.status,COUNT(pa.id) FROM payments p JOIN payment_allocations pa ON pa.payment_id=p.id JOIN student_invoices i ON i.id=pa.student_invoice_id JOIN charge_types ct ON ct.id=i.charge_type_id WHERE p.student_id=?`
	paymentArgs := []any{item.ID}
	if periodID > 0 {
		paymentQuery += ` AND i.academic_period_id=?`
		paymentArgs = append(paymentArgs, periodID)
	}
	paymentQuery += ` GROUP BY p.id,p.payment_number,p.payment_date,p.payment_method,p.reference_number,p.status ORDER BY p.payment_date DESC,p.id DESC`
	paymentRows, err := h.db.QueryContext(c.Request.Context(), paymentQuery, paymentArgs...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	payments := []studentPaymentHistory{}
	for paymentRows.Next() {
		var x studentPaymentHistory
		var rawPurpose string
		var amount float64
		if err = paymentRows.Scan(&x.ID, &x.Number, &rawPurpose, &x.Date, &x.Method, &x.Reference, &amount, &x.Status, &x.AllocationCount); err != nil {
			paymentRows.Close()
			c.AbortWithError(500, err)
			return
		}
		x.Method = paymentMethodLabel(x.Method)
		x.Purpose = template.HTML(strings.ReplaceAll(stdhtml.EscapeString(rawPurpose), "||", "<br>"))
		x.Amount = formatRupiah(amount)
		x.StatusLabel = paymentStatusLabel(x.Status)
		payments = append(payments, x)
	}
	paymentRows.Close()

	fundingQuery := `SELECT sfp.id,fp.name,fp.code,b.benefit_type,b.benefit_value,ct.name,COALESCE(CONCAT(ap.academic_year,' / ',ap.semester),'Semua semester'),COALESCE(sfp.reference_number,''),CONCAT(DATE_FORMAT(sfp.start_date,'%d/%m/%Y'),' – ',COALESCE(DATE_FORMAT(sfp.end_date,'%d/%m/%Y'),'seterusnya')),sfp.status,COUNT(DISTINCT a.id),COALESCE(SUM(a.amount),0) FROM student_funding_programs sfp JOIN funding_programs fp ON fp.id=sfp.funding_program_id JOIN funding_program_benefits b ON b.funding_program_id=fp.id JOIN charge_types ct ON ct.id=b.charge_type_id LEFT JOIN academic_periods ap ON ap.id=sfp.academic_period_id LEFT JOIN student_invoice_adjustments a ON a.student_funding_program_id=sfp.id WHERE sfp.student_id=?`
	fundingArgs := []any{item.ID}
	if periodID > 0 {
		fundingQuery += ` AND sfp.academic_period_id=?`
		fundingArgs = append(fundingArgs, periodID)
	}
	fundingQuery += ` GROUP BY sfp.id,fp.name,fp.code,b.benefit_type,b.benefit_value,ct.name,ap.academic_year,ap.semester,sfp.reference_number,sfp.start_date,sfp.end_date,sfp.status ORDER BY sfp.start_date DESC,sfp.id DESC`
	fundingRows, err := h.db.QueryContext(c.Request.Context(), fundingQuery, fundingArgs...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	fundingPrograms := []studentFundingHistory{}
	for fundingRows.Next() {
		var x studentFundingHistory
		var benefitType string
		var benefitValue, adjustmentTotal float64
		if err = fundingRows.Scan(&x.ID, &x.Program, &x.Code, &benefitType, &benefitValue, &x.ChargeType, &x.Period, &x.Reference, &x.Dates, &x.Status, &x.AdjustmentCount, &adjustmentTotal); err != nil {
			fundingRows.Close()
			c.AbortWithError(500, err)
			return
		}
		x.Benefit = fundingBenefitLabel(benefitType, benefitValue)
		x.StatusLabel = studentFundingStatusLabel(x.Status)
		x.AdjustmentTotal = formatRupiah(adjustmentTotal)
		fundingPrograms = append(fundingPrograms, x)
	}
	fundingRows.Close()
	p := auth.CurrentPrincipal(c)
	data := h.baseData(p, "Detail Mahasiswa")
	data["Student"] = item
	data["Invoices"] = invoices
	data["Payments"] = payments
	data["FundingPrograms"] = fundingPrograms
	data["Periods"] = periods
	data["SelectedPeriod"] = selectedPeriod
	data["ActiveTab"] = tab
	data["InvoiceCount"] = invoiceCount
	data["PaymentCount"] = paymentCount
	data["FundingCount"] = fundingCount
	data["Billed"] = formatRupiah(billed)
	data["Paid"] = formatRupiah(paid)
	data["Remaining"] = formatRupiah(max(billed-paid, 0))
	data["Status"] = c.Query("status")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("students.update")
	c.HTML(200, "students/details.html", data)
}
func (h studentsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var refs int
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM student_billing_policies WHERE student_id=?)+(SELECT COUNT(*) FROM student_invoices WHERE student_id=?)+(SELECT COUNT(*) FROM payments WHERE student_id=?)+(SELECT COUNT(*) FROM student_funding_programs WHERE student_id=?)`, id, id, id, id).Scan(&refs)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if refs > 0 {
		c.Redirect(303, "/students?status=in-use")
		return
	}
	r, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM students WHERE id=?`, id)
	if err != nil {
		c.Redirect(303, "/students?status=delete-failed")
		return
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/students?status=deleted")
}

func (h studentsHandlers) find(c *gin.Context, id any) (studentForm, error) {
	var f studentForm
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,nim,name,study_program_id,COALESCE(student_category_id,0),cohort_year,COALESCE(cohort_year_class,''),status,COALESCE(email,''),COALESCE(phone_number,'') FROM students WHERE id=?`, id).Scan(&f.ID, &f.NIM, &f.Name, &f.ProgramID, &f.CategoryID, &f.CohortYear, &f.CohortClass, &f.Status, &f.Email, &f.Phone)
	return f, err
}
func studentFormFromRequest(c *gin.Context) studentForm {
	return studentForm{NIM: strings.ToUpper(strings.TrimSpace(c.PostForm("nim"))), Name: strings.TrimSpace(c.PostForm("name")), ProgramID: strings.TrimSpace(c.PostForm("study_program_id")), CategoryID: strings.TrimSpace(c.PostForm("student_category_id")), CohortYear: strings.TrimSpace(c.PostForm("cohort_year")), CohortClass: strings.ToUpper(strings.TrimSpace(c.PostForm("cohort_year_class"))), Status: strings.ToUpper(strings.TrimSpace(c.PostForm("status"))), Email: strings.ToLower(strings.TrimSpace(c.PostForm("email"))), Phone: strings.TrimSpace(c.PostForm("phone_number"))}
}
func (h studentsHandlers) validate(c *gin.Context, f studentForm, id int64) string {
	year, _ := strconv.Atoi(f.CohortYear)
	switch {
	case f.NIM == "" || utf8.RuneCountInString(f.NIM) > 30:
		return "NIM wajib diisi dan maksimal 30 karakter."
	case f.Name == "" || utf8.RuneCountInString(f.Name) > 150:
		return "Nama mahasiswa wajib diisi dan maksimal 150 karakter."
	case parsePositiveID(f.ProgramID) == 0:
		return "Program studi wajib dipilih."
	case parsePositiveID(f.CategoryID) == 0:
		return "Golongan mahasiswa wajib dipilih."
	case year < 1900 || year > 2100:
		return "Tahun angkatan harus berada antara 1900 dan 2100."
	case utf8.RuneCountInString(f.CohortClass) > 30:
		return "Kelompok angkatan maksimal 30 karakter."
	case !validStudentStatus(f.Status):
		return "Status mahasiswa tidak valid."
	case f.Email != "" && utf8.RuneCountInString(f.Email) > 150:
		return "Email maksimal 150 karakter."
	case f.Email != "":
		if _, err := mail.ParseAddress(f.Email); err != nil {
			return "Format email tidak valid."
		}
	case utf8.RuneCountInString(f.Phone) > 30:
		return "Nomor telepon maksimal 30 karakter."
	}
	var n int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM students WHERE nim=? AND id<>?`, f.NIM, id).Scan(&n); err != nil {
		return "Gagal memvalidasi NIM."
	}
	if n > 0 {
		return "NIM sudah digunakan."
	}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM study_programs WHERE id=?)+(SELECT COUNT(*) FROM student_categories WHERE id=?)`, f.ProgramID, f.CategoryID).Scan(&n); err != nil || n != 2 {
		return "Program studi atau golongan mahasiswa tidak valid."
	}
	return ""
}
func (h studentsHandlers) options(c *gin.Context) ([]studentOption, []studentOption) {
	load := func(q string) []studentOption {
		rows, err := h.db.QueryContext(c.Request.Context(), q)
		if err != nil {
			return nil
		}
		defer rows.Close()
		out := []studentOption{}
		for rows.Next() {
			var x studentOption
			if rows.Scan(&x.ID, &x.Label) == nil {
				out = append(out, x)
			}
		}
		return out
	}
	return load(`SELECT id,CONCAT(code,' - ',name) FROM study_programs ORDER BY name`), load(`SELECT id,name FROM student_categories ORDER BY name`)
}
func (h studentsHandlers) filterOptions(c *gin.Context) ([]string, []string) {
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
	return load(`SELECT DISTINCT cohort_year FROM students ORDER BY cohort_year DESC`), load(`SELECT DISTINCT cohort_year_class FROM students WHERE cohort_year_class IS NOT NULL AND cohort_year_class<>'' ORDER BY cohort_year_class DESC`)
}
func (h studentsHandlers) renderForm(c *gin.Context, status int, title string, f studentForm, message string) {
	programs, categories := h.options(c)
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["Form"] = f
	data["Programs"] = programs
	data["Categories"] = categories
	data["IsEdit"] = f.ID > 0
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "students/form.html", data)
}
func (h studentsHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "students"}
}
func validStudentStatus(s string) bool {
	return s == "ACTIVE" || s == "ON_LEAVE" || s == "GRADUATED" || s == "INACTIVE"
}
func studentStatusLabel(s string) string {
	switch s {
	case "ACTIVE":
		return "Aktif"
	case "ON_LEAVE":
		return "Cuti"
	case "GRADUATED":
		return "Lulus"
	default:
		return "Nonaktif"
	}
}
func invoiceStatusLabel(s string) string {
	switch s {
	case "PAID":
		return "Lunas"
	case "PARTIALLY_PAID":
		return "Dibayar Sebagian"
	case "OVERDUE":
		return "Jatuh Tempo"
	case "WAIVED":
		return "Dibebaskan"
	default:
		return "Belum Dibayar"
	}
}
func formatRupiah(v float64) string {
	raw := strconv.FormatInt(int64(v+0.5), 10)
	for i := len(raw) - 3; i > 0; i -= 3 {
		raw = raw[:i] + "." + raw[i:]
	}
	return "Rp" + raw
}
