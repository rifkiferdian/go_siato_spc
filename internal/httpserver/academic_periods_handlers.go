package httpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

var academicYearPattern = regexp.MustCompile(`^\d{4}/\d{4}$`)

type academicPeriodsHandlers struct {
	cfg config.Config
	db  *sql.DB
}

type academicPeriodItem struct {
	ID                                   int64
	Code, AcademicYear, Semester         string
	StartDate, EndDate, UTSDate, UASDate string
	IsActive                             bool
}

type academicPeriodForm struct {
	Code, AcademicYear, Semester         string
	StartDate, EndDate, UTSDate, UASDate string
	IsActive                             bool
}

func (h academicPeriodsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id, code, academic_year, semester, start_date, end_date, uts_date, uas_date, is_active FROM academic_periods WHERE (? = '' OR code LIKE ? OR academic_year LIKE ? OR semester LIKE ?) ORDER BY start_date DESC`, search, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	items := make([]academicPeriodItem, 0)
	for rows.Next() {
		var item academicPeriodItem
		var start, end time.Time
		var uts, uas sql.NullTime
		if err := rows.Scan(&item.ID, &item.Code, &item.AcademicYear, &item.Semester, &start, &end, &uts, &uas, &item.IsActive); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		item.StartDate, item.EndDate = formatDisplayDate(start), formatDisplayDate(end)
		if uts.Valid {
			item.UTSDate = formatDisplayDate(uts.Time)
		} else {
			item.UTSDate = "-"
		}
		if uas.Valid {
			item.UASDate = formatDisplayDate(uas.Time)
		} else {
			item.UASDate = "-"
		}
		items = append(items, item)
	}
	metrics := struct{ Total, Active, Odd, Even int }{}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_active=1 THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN semester='GANJIL' THEN 1 ELSE 0 END),0), COALESCE(SUM(CASE WHEN semester='GENAP' THEN 1 ELSE 0 END),0) FROM academic_periods`).Scan(&metrics.Total, &metrics.Active, &metrics.Odd, &metrics.Even); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	data := h.baseData(p, "Periode Akademik")
	data["Items"], data["Metrics"], data["Search"], data["Status"] = items, metrics, search, c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("academic-periods.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("academic-periods.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("academic-periods.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "academic-periods/index.html", data)
}

func (h academicPeriodsHandlers) createForm(c *gin.Context) {
	now := time.Now()
	h.renderForm(c, http.StatusOK, "Tambah Periode Akademik", 0, academicPeriodForm{Semester: "GANJIL", StartDate: now.Format("2006-01-02"), EndDate: now.AddDate(0, 6, 0).Format("2006-01-02")}, "")
}

func (h academicPeriodsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	form, dates, message := parseAcademicPeriodForm(c)
	if message != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Periode Akademik", 0, form, message)
		return
	}
	duplicateMessage, err := h.validateUnique(c, form, 0)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if duplicateMessage != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Periode Akademik", 0, form, duplicateMessage)
		return
	}
	_, err = h.db.ExecContext(c.Request.Context(), `INSERT INTO academic_periods (code, academic_year, semester, start_date, end_date, uts_date, uas_date, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))`, form.Code, form.AcademicYear, form.Semester, dates.start, dates.end, nullDate(dates.uts), nullDate(dates.uas), form.IsActive)
	if err != nil {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Periode Akademik", 0, form, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/academic-periods?status=created")
}

func (h academicPeriodsHandlers) editForm(c *gin.Context) {
	var id int64
	var form academicPeriodForm
	var start, end time.Time
	var uts, uas sql.NullTime
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id, code, academic_year, semester, start_date, end_date, uts_date, uas_date, is_active FROM academic_periods WHERE id=?`, c.Param("id")).Scan(&id, &form.Code, &form.AcademicYear, &form.Semester, &start, &end, &uts, &uas, &form.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	form.StartDate, form.EndDate = start.Format("2006-01-02"), end.Format("2006-01-02")
	if uts.Valid {
		form.UTSDate = uts.Time.Format("2006-01-02")
	}
	if uas.Valid {
		form.UASDate = uas.Time.Format("2006-01-02")
	}
	h.renderForm(c, http.StatusOK, "Ubah Periode Akademik", id, form, "")
}

func (h academicPeriodsHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id := parsePositiveID(c.Param("id"))
	if id == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	form, dates, message := parseAcademicPeriodForm(c)
	if message != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Periode Akademik", id, form, message)
		return
	}
	duplicateMessage, err := h.validateUnique(c, form, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if duplicateMessage != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Periode Akademik", id, form, duplicateMessage)
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `UPDATE academic_periods SET code=?, academic_year=?, semester=?, start_date=?, end_date=?, uts_date=?, uas_date=?, is_active=?, updated_at=UTC_TIMESTAMP(6) WHERE id=?`, form.Code, form.AcademicYear, form.Semester, dates.start, dates.end, nullDate(dates.uts), nullDate(dates.uas), form.IsActive, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusSeeOther, "/academic-periods?status=updated")
}

func (h academicPeriodsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id := c.Param("id")
	var references int
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT
        (SELECT COUNT(*) FROM billing_policies WHERE academic_period_id=?) +
        (SELECT COUNT(*) FROM invoice_generation_batches WHERE academic_period_id=?) +
        (SELECT COUNT(*) FROM student_funding_programs WHERE academic_period_id=?) +
        (SELECT COUNT(*) FROM student_invoices WHERE academic_period_id=?)`, id, id, id, id).Scan(&references)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if references > 0 {
		c.Redirect(http.StatusSeeOther, "/academic-periods?status=in-use")
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM academic_periods WHERE id=?`, id)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/academic-periods?status=delete-failed")
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusSeeOther, "/academic-periods?status=deleted")
}

func (h academicPeriodsHandlers) validateUnique(c *gin.Context, form academicPeriodForm, excludedID int64) (string, error) {
	var codeCount, combinationCount int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(SUM(code=? AND id<>?),0), COALESCE(SUM(academic_year=? AND semester=? AND id<>?),0) FROM academic_periods`, form.Code, excludedID, form.AcademicYear, form.Semester, excludedID).Scan(&codeCount, &combinationCount); err != nil {
		return "", err
	}
	if codeCount > 0 {
		return "Kode periode sudah digunakan.", nil
	}
	if combinationCount > 0 {
		return "Kombinasi tahun akademik dan semester sudah tersedia.", nil
	}
	return "", nil
}

func (h academicPeriodsHandlers) renderForm(c *gin.Context, status int, title string, id int64, form academicPeriodForm, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["EditID"], data["Form"], data["IsEdit"], data["Error"] = id, form, id > 0, message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "academic-periods/form.html", data)
}

func (h academicPeriodsHandlers) baseData(p *auth.Principal, title string) gin.H {
	roleLabel := strings.Join(p.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": roleLabel, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "academic-periods"}
}

type academicPeriodDates struct {
	start, end time.Time
	uts, uas   *time.Time
}

func parseAcademicPeriodForm(c *gin.Context) (academicPeriodForm, academicPeriodDates, string) {
	form := academicPeriodForm{Code: strings.ToUpper(strings.TrimSpace(c.PostForm("code"))), AcademicYear: strings.TrimSpace(c.PostForm("academic_year")), Semester: strings.ToUpper(strings.TrimSpace(c.PostForm("semester"))), StartDate: c.PostForm("start_date"), EndDate: c.PostForm("end_date"), UTSDate: c.PostForm("uts_date"), UASDate: c.PostForm("uas_date"), IsActive: c.PostForm("is_active") == "1"}
	dates := academicPeriodDates{}
	if form.Code == "" || utf8.RuneCountInString(form.Code) > 30 {
		return form, dates, "Kode periode wajib diisi dan maksimal 30 karakter."
	}
	if !academicYearPattern.MatchString(form.AcademicYear) {
		return form, dates, "Gunakan format tahun akademik 2026/2027."
	}
	if form.Semester != "GANJIL" && form.Semester != "GENAP" && form.Semester != "PENDEK" {
		return form, dates, "Semester tidak valid."
	}
	var err error
	if dates.start, err = time.Parse("2006-01-02", form.StartDate); err != nil {
		return form, dates, "Tanggal mulai wajib diisi."
	}
	if dates.end, err = time.Parse("2006-01-02", form.EndDate); err != nil {
		return form, dates, "Tanggal selesai wajib diisi."
	}
	if dates.end.Before(dates.start) {
		return form, dates, "Tanggal selesai tidak boleh sebelum tanggal mulai."
	}
	if form.UTSDate != "" {
		value, parseErr := time.Parse("2006-01-02", form.UTSDate)
		if parseErr != nil {
			return form, dates, "Tanggal UTS tidak valid."
		}
		dates.uts = &value
	}
	if form.UASDate != "" {
		value, parseErr := time.Parse("2006-01-02", form.UASDate)
		if parseErr != nil {
			return form, dates, "Tanggal UAS tidak valid."
		}
		dates.uas = &value
	}
	if dates.uts != nil && (dates.uts.Before(dates.start) || dates.uts.After(dates.end)) {
		return form, dates, "Tanggal UTS harus berada dalam periode akademik."
	}
	if dates.uas != nil && (dates.uas.Before(dates.start) || dates.uas.After(dates.end)) {
		return form, dates, "Tanggal UAS harus berada dalam periode akademik."
	}
	if dates.uts != nil && dates.uas != nil && dates.uas.Before(*dates.uts) {
		return form, dates, "Tanggal UAS tidak boleh sebelum tanggal UTS."
	}
	return form, dates, ""
}

func nullDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func formatDisplayDate(value time.Time) string {
	months := [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	return value.Format("02") + " " + months[value.Month()-1] + " " + value.Format("2006")
}
