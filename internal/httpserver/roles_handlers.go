package httpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type rolesHandlers struct {
	cfg config.Config
	db  *sql.DB
}

type managedRoleItem struct {
	ID, Name, Initial string
	UserCount         int
	PermissionCount   int
	Protected         bool
}

type permissionOption struct {
	ID                   int64
	Name, Group, Display string
	Selected             bool
}

type permissionGroup struct {
	Name        string
	Permissions []permissionOption
}

func (h rolesHandlers) index(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT r.Id, COALESCE(r.Name, ''),
       COUNT(DISTINCT ur.UserId), COUNT(DISTINCT rp.permission_id)
FROM roles r
LEFT JOIN user_roles ur ON ur.RoleId = r.Id
LEFT JOIN role_permissions rp ON rp.role_id = r.Id
WHERE (? = '' OR r.Name LIKE ?)
GROUP BY r.Id, r.Name
ORDER BY r.Name`, search, "%"+search+"%")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	roles := make([]managedRoleItem, 0)
	for rows.Next() {
		var role managedRoleItem
		if err := rows.Scan(&role.ID, &role.Name, &role.UserCount, &role.PermissionCount); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		role.Initial = firstInitial(role.Name)
		role.Protected = strings.EqualFold(role.Name, "Super Admin")
		roles = append(roles, role)
	}
	metrics := struct{ TotalRoles, UsedRoles, TotalPermissions, Assignments int }{}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT
        (SELECT COUNT(*) FROM roles),
        (SELECT COUNT(DISTINCT RoleId) FROM user_roles),
        (SELECT COUNT(*) FROM permissions),
        (SELECT COUNT(*) FROM role_permissions)`).Scan(&metrics.TotalRoles, &metrics.UsedRoles, &metrics.TotalPermissions, &metrics.Assignments); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	data := h.baseData(auth.CurrentPrincipal(c), "Role & Hak Akses")
	data["Roles"] = roles
	data["Metrics"] = metrics
	data["Search"] = search
	data["Status"] = c.Query("status")
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "roles/index.html", data)
}

func (h rolesHandlers) createForm(c *gin.Context) {
	h.renderForm(c, http.StatusOK, "Tambah Role", "", "", nil, "")
}

func (h rolesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	name := strings.TrimSpace(c.PostForm("name"))
	permissionIDs, err := parsePermissionIDs(c.PostFormArray("permissions"))
	if err != nil || name == "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Role", "", name, permissionIDs, "Nama role wajib diisi dan permission harus valid.")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM roles WHERE NormalizedName = ?`, strings.ToUpper(name)).Scan(&exists); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if exists > 0 {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Role", "", name, permissionIDs, "Nama role sudah digunakan.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO roles (Id, Name, NormalizedName, ConcurrencyStamp) VALUES (UUID(), ?, ?, UUID())`, name, strings.ToUpper(name)); err != nil {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Role", "", name, permissionIDs, err.Error())
		return
	}
	var roleID string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT Id FROM roles WHERE NormalizedName = ?`, strings.ToUpper(name)).Scan(&roleID); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := saveRolePermissions(c, tx, roleID, permissionIDs); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/roles?status=created")
}

func (h rolesHandlers) editForm(c *gin.Context) {
	var name string
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(Name, '') FROM roles WHERE Id = ?`, c.Param("id")).Scan(&name); errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	selected, err := h.selectedPermissions(c, c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	h.renderForm(c, http.StatusOK, "Ubah Role", c.Param("id"), name, selected, "")
}

func (h rolesHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id, name := c.Param("id"), strings.TrimSpace(c.PostForm("name"))
	permissionIDs, err := parsePermissionIDs(c.PostFormArray("permissions"))
	if err != nil || name == "" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Role", id, name, permissionIDs, "Nama role wajib diisi dan permission harus valid.")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()
	var currentName string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT COALESCE(Name, '') FROM roles WHERE Id = ? FOR UPDATE`, id).Scan(&currentName); errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if strings.EqualFold(currentName, "Super Admin") && name != "Super Admin" {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Role", id, name, permissionIDs, "Nama role Super Admin tidak dapat diubah.")
		return
	}
	var duplicate int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM roles WHERE NormalizedName = ? AND Id <> ?`, strings.ToUpper(name), id).Scan(&duplicate); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if duplicate > 0 {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Role", id, name, permissionIDs, "Nama role sudah digunakan.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE roles SET Name = ?, NormalizedName = ?, ConcurrencyStamp = UUID() WHERE Id = ?`, name, strings.ToUpper(name), id); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := saveRolePermissions(c, tx, id, permissionIDs); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if err := tx.Commit(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/roles?status=updated")
}

func (h rolesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id := c.Param("id")
	var name string
	var userCount int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(r.Name, ''), COUNT(ur.UserId) FROM roles r LEFT JOIN user_roles ur ON ur.RoleId = r.Id WHERE r.Id = ? GROUP BY r.Id, r.Name`, id).Scan(&name, &userCount); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if strings.EqualFold(name, "Super Admin") {
		c.Redirect(http.StatusSeeOther, "/roles?status=protected")
		return
	}
	if userCount > 0 {
		c.Redirect(http.StatusSeeOther, "/roles?status=in-use")
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM roles WHERE Id = ?`, id); err != nil {
		c.Redirect(http.StatusSeeOther, "/roles?status=delete-failed")
		return
	}
	c.Redirect(http.StatusSeeOther, "/roles?status=deleted")
}

func (h rolesHandlers) renderForm(c *gin.Context, status int, title, id, name string, selected []int64, message string) {
	groups, err := h.permissionGroups(c, selected)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["EditID"], data["Name"], data["PermissionGroups"], data["Error"], data["IsEdit"] = id, name, groups, message, id != ""
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "roles/form.html", data)
}

func (h rolesHandlers) permissionGroups(c *gin.Context, selected []int64) ([]permissionGroup, error) {
	set := make(map[int64]bool, len(selected))
	for _, id := range selected {
		set[id] = true
	}
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT id, name FROM permissions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groupOrder := []string{}
	groupMap := map[string][]permissionOption{}
	for rows.Next() {
		var option permissionOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return nil, err
		}
		parts := strings.Split(option.Name, ".")
		option.Group = titleWords(parts[0])
		if len(parts) > 1 {
			option.Display = titleWords(strings.Join(parts[1:], " "))
		} else {
			option.Display = titleWords(option.Name)
		}
		option.Selected = set[option.ID]
		if _, exists := groupMap[option.Group]; !exists {
			groupOrder = append(groupOrder, option.Group)
		}
		groupMap[option.Group] = append(groupMap[option.Group], option)
	}
	groups := make([]permissionGroup, 0, len(groupOrder))
	for _, name := range groupOrder {
		groups = append(groups, permissionGroup{Name: name, Permissions: groupMap[name]})
	}
	return groups, rows.Err()
}

func (h rolesHandlers) selectedPermissions(c *gin.Context, roleID string) ([]int64, error) {
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT permission_id FROM role_permissions WHERE role_id = ?`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (h rolesHandlers) baseData(p *auth.Principal, title string) gin.H {
	roleLabel := strings.Join(p.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": roleLabel, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "roles"}
}

func saveRolePermissions(c *gin.Context, tx *sql.Tx, roleID string, permissionIDs []int64) error {
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, permissionID := range permissionIDs {
		if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO role_permissions (role_id, permission_id) SELECT ?, id FROM permissions WHERE id = ?`, roleID, permissionID); err != nil {
			return err
		}
	}
	return nil
}

func parsePermissionIDs(values []string) ([]int64, error) {
	seen := map[int64]bool{}
	ids := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.New("permission tidak valid")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func titleWords(value string) string {
	value = strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(value)
	words := strings.Fields(value)
	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}
