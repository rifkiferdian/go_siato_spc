package httpserver

import (
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type chargeTypesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type chargeTypeItem struct {
	ID                      int64
	Code, Name, Description string
	IsActive                bool
}

func (h chargeTypesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id,code,name,COALESCE(description,''),is_active FROM charge_types WHERE (?='' OR code LIKE ? OR name LIKE ?) ORDER BY name`, search, "%"+search+"%", "%"+search+"%")
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []chargeTypeItem{}
	for rows.Next() {
		var item chargeTypeItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.IsActive); err != nil {
			c.AbortWithError(500, err)
			return
		}
		items = append(items, item)
	}
	metrics := struct{ Total, Active, Inactive, WithoutDescription int }{}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(is_active=1),0),COALESCE(SUM(is_active=0),0),COALESCE(SUM(description IS NULL OR description=''),0) FROM charge_types`).Scan(&metrics.Total, &metrics.Active, &metrics.Inactive, &metrics.WithoutDescription); err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, "Jenis Biaya")
	data["Items"], data["Metrics"], data["Search"], data["Status"] = items, metrics, search, c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("charge-types.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("charge-types.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("charge-types.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "charge-types/index.html", data)
}
func (h chargeTypesHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Jenis Biaya", 0, "", "", "", true, "")
}
func (h chargeTypesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	code, name, description, active := normalizeChargeType(c)
	if message := validateChargeType(code, name, description); message != "" {
		h.renderForm(c, 422, "Tambah Jenis Biaya", 0, code, name, description, active, message)
		return
	}
	var exists int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM charge_types WHERE code=?`, code).Scan(&exists); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if exists > 0 {
		h.renderForm(c, 422, "Tambah Jenis Biaya", 0, code, name, description, active, "Kode jenis biaya sudah digunakan.")
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(), `INSERT INTO charge_types(code,name,description,is_active,created_at,updated_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, code, name, nullableString(description), active); err != nil {
		h.renderForm(c, 422, "Tambah Jenis Biaya", 0, code, name, description, active, err.Error())
		return
	}
	c.Redirect(303, "/charge-types?status=created")
}
func (h chargeTypesHandlers) editForm(c *gin.Context) {
	var id int64
	var code, name, description string
	var active bool
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,code,name,COALESCE(description,''),is_active FROM charge_types WHERE id=?`, c.Param("id")).Scan(&id, &code, &name, &description, &active)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Jenis Biaya", id, code, name, description, active, "")
}
func (h chargeTypesHandlers) update(c *gin.Context) {
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
	code, name, description, active := normalizeChargeType(c)
	if message := validateChargeType(code, name, description); message != "" {
		h.renderForm(c, 422, "Ubah Jenis Biaya", id, code, name, description, active, message)
		return
	}
	var duplicate int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM charge_types WHERE code=? AND id<>?`, code, id).Scan(&duplicate); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if duplicate > 0 {
		h.renderForm(c, 422, "Ubah Jenis Biaya", id, code, name, description, active, "Kode jenis biaya sudah digunakan.")
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `UPDATE charge_types SET code=?,name=?,description=?,is_active=?,updated_at=UTC_TIMESTAMP(6) WHERE id=?`, code, name, nullableString(description), active, id)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/charge-types?status=updated")
}
func (h chargeTypesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var references int
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM billing_policies WHERE charge_type_id=?)+(SELECT COUNT(*) FROM funding_program_benefits WHERE charge_type_id=?)+(SELECT COUNT(*) FROM student_invoices WHERE charge_type_id=?)`, id, id, id).Scan(&references)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if references > 0 {
		c.Redirect(303, "/charge-types?status=in-use")
		return
	}
	result, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM charge_types WHERE id=?`, id)
	if err != nil {
		c.Redirect(303, "/charge-types?status=delete-failed")
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.AbortWithStatus(404)
		return
	}
	c.Redirect(303, "/charge-types?status=deleted")
}
func (h chargeTypesHandlers) renderForm(c *gin.Context, status int, title string, id int64, code, name, description string, active bool, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["EditID"], data["Code"], data["Name"], data["Description"], data["IsActive"], data["IsEdit"], data["Error"] = id, code, name, description, active, id > 0, message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "charge-types/form.html", data)
}
func (h chargeTypesHandlers) baseData(p *auth.Principal, title string) gin.H {
	roleLabel := strings.Join(p.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": roleLabel, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "charge-types"}
}
func normalizeChargeType(c *gin.Context) (string, string, string, bool) {
	return strings.ToUpper(strings.TrimSpace(c.PostForm("code"))), strings.TrimSpace(c.PostForm("name")), strings.TrimSpace(c.PostForm("description")), c.PostForm("is_active") == "1"
}
func validateChargeType(code, name, description string) string {
	switch {
	case code == "" || utf8.RuneCountInString(code) > 30:
		return "Kode jenis biaya wajib diisi dan maksimal 30 karakter."
	case name == "" || utf8.RuneCountInString(name) > 100:
		return "Nama jenis biaya wajib diisi dan maksimal 100 karakter."
	case utf8.RuneCountInString(description) > 500:
		return "Keterangan maksimal 500 karakter."
	default:
		return ""
	}
}
