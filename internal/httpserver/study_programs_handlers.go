package httpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studyProgramsHandlers struct {
	cfg config.Config
	db  *sql.DB
}

type studyProgramItem struct {
	ID                         int64
	Code, Name, EducationLevel string
	IsActive                   bool
	StudentCount               int
}

func (h studyProgramsHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT sp.id, sp.code, sp.name, sp.education_level, sp.is_active, COUNT(s.id)
FROM study_programs sp
LEFT JOIN students s ON s.study_program_id = sp.id
WHERE (? = '' OR sp.code LIKE ? OR sp.name LIKE ?)
GROUP BY sp.id, sp.code, sp.name, sp.education_level, sp.is_active
ORDER BY sp.name`, search, "%"+search+"%", "%"+search+"%")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	items := make([]studyProgramItem, 0)
	for rows.Next() {
		var item studyProgramItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.EducationLevel, &item.IsActive, &item.StudentCount); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		items = append(items, item)
	}
	metrics := struct{ Total, Active, Inactive, Students int }{}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT
        (SELECT COUNT(*) FROM study_programs),
        (SELECT COUNT(*) FROM study_programs WHERE is_active = 1),
        (SELECT COUNT(*) FROM study_programs WHERE is_active = 0),
        (SELECT COUNT(*) FROM students)`).Scan(&metrics.Total, &metrics.Active, &metrics.Inactive, &metrics.Students); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	data := h.baseData(p, "Program Studi")
	data["Items"], data["Metrics"], data["Search"], data["Status"] = items, metrics, search, c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("study-programs.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("study-programs.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("study-programs.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "study-programs/index.html", data)
}

func (h studyProgramsHandlers) createForm(c *gin.Context) {
	h.renderForm(c, http.StatusOK, "Tambah Program Studi", 0, "", "", "", true, "")
}

func (h studyProgramsHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	code, name, level, active := normalizeStudyProgram(c)
	if message := validateStudyProgram(code, name, level); message != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Program Studi", 0, code, name, level, active, message)
		return
	}
	var exists int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM study_programs WHERE code = ?`, code).Scan(&exists); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if exists > 0 {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Program Studi", 0, code, name, level, active, "Kode program studi sudah digunakan.")
		return
	}
	_, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO study_programs (code, name, education_level, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))`, code, name, level, active)
	if err != nil {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Program Studi", 0, code, name, level, active, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/study-programs?status=created")
}

func (h studyProgramsHandlers) editForm(c *gin.Context) {
	var id int64
	var code, name, level string
	var active bool
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id, code, name, education_level, is_active FROM study_programs WHERE id = ?`, c.Param("id")).Scan(&id, &code, &name, &level, &active)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	h.renderForm(c, http.StatusOK, "Ubah Program Studi", id, code, name, level, active, "")
}

func (h studyProgramsHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id, code, name, level, active := c.Param("id"), "", "", "", false
	code, name, level, active = normalizeStudyProgram(c)
	if message := validateStudyProgram(code, name, level); message != "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Program Studi", parsePositiveID(id), code, name, level, active, message)
		return
	}
	var duplicate int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM study_programs WHERE code = ? AND id <> ?`, code, id).Scan(&duplicate); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if duplicate > 0 {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Program Studi", parsePositiveID(id), code, name, level, active, "Kode program studi sudah digunakan.")
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `UPDATE study_programs SET code = ?, name = ?, education_level = ?, is_active = ?, updated_at = UTC_TIMESTAMP(6) WHERE id = ?`, code, name, level, active, id)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusSeeOther, "/study-programs?status=updated")
}

func (h studyProgramsHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	var references int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT
        (SELECT COUNT(*) FROM students WHERE study_program_id = ?) +
        (SELECT COUNT(*) FROM billing_policies WHERE study_program_id = ?) +
        (SELECT COUNT(*) FROM student_billing_policy_batches WHERE study_program_id = ?)`, c.Param("id"), c.Param("id"), c.Param("id")).Scan(&references); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if references > 0 {
		c.Redirect(http.StatusSeeOther, "/study-programs?status=in-use")
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM study_programs WHERE id = ?`, c.Param("id"))
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/study-programs?status=delete-failed")
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Redirect(http.StatusSeeOther, "/study-programs?status=deleted")
}

func (h studyProgramsHandlers) renderForm(c *gin.Context, status int, title string, id int64, code, name, level string, active bool, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["EditID"], data["Code"], data["Name"], data["EducationLevel"] = id, code, name, level
	data["IsActive"], data["IsEdit"], data["Error"] = active, id > 0, message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "study-programs/form.html", data)
}

func (h studyProgramsHandlers) baseData(p *auth.Principal, title string) gin.H {
	roleLabel := strings.Join(p.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": roleLabel, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "study-programs"}
}

func normalizeStudyProgram(c *gin.Context) (string, string, string, bool) {
	return strings.ToUpper(strings.TrimSpace(c.PostForm("code"))), strings.TrimSpace(c.PostForm("name")), strings.ToUpper(strings.TrimSpace(c.PostForm("education_level"))), c.PostForm("is_active") == "1"
}

func validateStudyProgram(code, name, level string) string {
	switch {
	case code == "":
		return "Kode program studi wajib diisi."
	case utf8.RuneCountInString(code) > 20:
		return "Kode program studi maksimal 20 karakter."
	case name == "":
		return "Nama program studi wajib diisi."
	case utf8.RuneCountInString(name) > 150:
		return "Nama program studi maksimal 150 karakter."
	case level == "":
		return "Jenjang pendidikan wajib diisi."
	case utf8.RuneCountInString(level) > 20:
		return "Jenjang pendidikan maksimal 20 karakter."
	default:
		return ""
	}
}

func parsePositiveID(value string) int64 {
	var id int64
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0
		}
		id = id*10 + int64(char-'0')
	}
	return id
}
