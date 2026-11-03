package httpserver

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type usersHandlers struct {
	cfg  config.Config
	db   *sql.DB
	auth *auth.Service
}

type managedUser struct {
	ID, Email, Initial string
	Number             int
	EmailConfirmed     bool
	IsCurrent          bool
	Roles              []string
}

type managedRole struct {
	ID, Name string
	Selected bool
}

func (h usersHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	like := "%" + search + "%"
	rows, err := h.db.QueryContext(c.Request.Context(), `
SELECT u.Id, COALESCE(u.Email, u.UserName, ''), u.EmailConfirmed,
       COALESCE(GROUP_CONCAT(r.Name ORDER BY r.Name SEPARATOR '||'), '')
FROM users u
LEFT JOIN user_roles ur ON ur.UserId = u.Id
LEFT JOIN roles r ON r.Id = ur.RoleId
WHERE (? = '' OR u.Email LIKE ?)
GROUP BY u.Id, u.Email, u.UserName, u.EmailConfirmed
ORDER BY u.Email`, search, like)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	users := make([]managedUser, 0)
	for rows.Next() {
		var user managedUser
		var roleNames string
		if err := rows.Scan(&user.ID, &user.Email, &user.EmailConfirmed, &roleNames); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		user.Initial = firstInitial(user.Email)
		user.Number = len(users) + 1
		user.IsCurrent = user.ID == p.UserID
		if roleNames != "" {
			user.Roles = strings.Split(roleNames, "||")
		}
		users = append(users, user)
	}
	metrics := struct{ Total, Verified, WithRole, WithoutRole int }{}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT
        COUNT(*), SUM(CASE WHEN EmailConfirmed = 1 THEN 1 ELSE 0 END),
        COUNT(DISTINCT CASE WHEN ur.UserId IS NOT NULL THEN u.Id END),
        COUNT(DISTINCT CASE WHEN ur.UserId IS NULL THEN u.Id END)
        FROM users u LEFT JOIN user_roles ur ON ur.UserId = u.Id`).Scan(&metrics.Total, &metrics.Verified, &metrics.WithRole, &metrics.WithoutRole); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	data := h.baseData(p, "Pengguna")
	data["Users"] = users
	data["Search"] = search
	data["Metrics"] = metrics
	data["Status"] = c.Query("status")
	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "users/index.html", data)
}

func (h usersHandlers) createForm(c *gin.Context) {
	h.renderForm(c, http.StatusOK, "Tambah Pengguna", "", "", nil, "")
}

func (h usersHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Pengguna", "", email, c.PostFormArray("roles"), "Email tidak valid.")
		return
	}
	if password != c.PostForm("password_confirmation") {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Pengguna", "", email, c.PostFormArray("roles"), "Konfirmasi kata sandi tidak sama.")
		return
	}
	if err := h.auth.CreateManagedUser(c.Request.Context(), email, password, c.PostFormArray("roles")); err != nil {
		h.renderForm(c, http.StatusUnprocessableEntity, "Tambah Pengguna", "", email, c.PostFormArray("roles"), err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/users?status=created")
}

func (h usersHandlers) editForm(c *gin.Context) {
	var email string
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(Email, '') FROM users WHERE Id = ?`, c.Param("id")).Scan(&email); errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT RoleId FROM user_roles WHERE UserId = ?`, c.Param("id"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()
	selected := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			selected = append(selected, id)
		}
	}
	h.renderForm(c, http.StatusOK, "Ubah Pengguna", c.Param("id"), email, selected, "")
}

func (h usersHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	id, email, roleIDs := c.Param("id"), strings.TrimSpace(c.PostForm("email")), c.PostFormArray("roles")
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		h.renderForm(c, http.StatusUnprocessableEntity, "Ubah Pengguna", id, email, roleIDs, "Email tidak valid.")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	if id == p.UserID {
		var keepsSuperAdmin int
		if len(roleIDs) > 0 {
			placeholders := strings.TrimRight(strings.Repeat("?,", len(roleIDs)), ",")
			args := make([]any, len(roleIDs))
			for i, v := range roleIDs {
				args[i] = v
			}
			if err := tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM roles WHERE Name = 'Super Admin' AND Id IN (`+placeholders+`)`, args...).Scan(&keepsSuperAdmin); err != nil {
				h.renderForm(c, 500, "Ubah Pengguna", id, email, roleIDs, err.Error())
				return
			}
		}
		if p.HasRole("Super Admin") && keepsSuperAdmin == 0 {
			h.renderForm(c, 422, "Ubah Pengguna", id, email, roleIDs, "Role Super Admin tidak dapat dilepas dari akun yang sedang digunakan.")
			return
		}
	}
	normalized := strings.ToUpper(email)
	result, err := tx.ExecContext(c.Request.Context(), `UPDATE users SET Email=?, UserName=?, NormalizedEmail=?, NormalizedUserName=?, SecurityStamp=UUID() WHERE Id=? AND NOT EXISTS (SELECT 1 FROM (SELECT Id FROM users WHERE NormalizedEmail=? AND Id<>?) x)`, email, email, normalized, normalized, id, normalized, id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Pengguna", id, email, roleIDs, err.Error())
		return
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		h.renderForm(c, 422, "Ubah Pengguna", id, email, roleIDs, "Email sudah digunakan atau pengguna tidak ditemukan.")
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM user_roles WHERE UserId=?`, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	for _, roleID := range roleIDs {
		if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO user_roles (UserId, RoleId) SELECT ?, Id FROM roles WHERE Id=?`, id, roleID); err != nil {
			c.AbortWithError(500, err)
			return
		}
	}
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM go_auth_sessions WHERE user_id=?`, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err := tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/users?status=updated")
}

func (h usersHandlers) resetForm(c *gin.Context) {
	var email string
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(Email,'') FROM users WHERE Id=?`, c.Param("id")).Scan(&email); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	h.renderReset(c, http.StatusOK, c.Param("id"), email, "")
}

func (h usersHandlers) resetPassword(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	id := c.Param("id")
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	var email string
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COALESCE(Email,'') FROM users WHERE Id=?`, id).Scan(&email); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if c.PostForm("password") != c.PostForm("password_confirmation") {
		h.renderReset(c, 422, id, email, "Konfirmasi kata sandi tidak sama.")
		return
	}
	if err := h.auth.SetManagedUserPassword(c.Request.Context(), id, c.PostForm("password")); err != nil {
		h.renderReset(c, 422, id, email, err.Error())
		return
	}
	if id == p.UserID {
		c.Redirect(http.StatusSeeOther, "/panel-admin-stieww")
		return
	}
	c.Redirect(http.StatusSeeOther, "/users?status=password-reset")
}

func (h usersHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	id := c.Param("id")
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	if id == p.UserID {
		c.Redirect(303, "/users?status=self-delete-denied")
		return
	}
	var superAdminCount int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM user_roles ur JOIN roles r ON r.Id=ur.RoleId WHERE r.Name='Super Admin'`).Scan(&superAdminCount); err != nil {
		c.AbortWithError(500, err)
		return
	}
	var targetIsSuperAdmin int
	h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM user_roles ur JOIN roles r ON r.Id=ur.RoleId WHERE ur.UserId=? AND r.Name='Super Admin'`, id).Scan(&targetIsSuperAdmin)
	if targetIsSuperAdmin > 0 && superAdminCount <= 1 {
		c.Redirect(303, "/users?status=last-super-admin")
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(), `DELETE FROM users WHERE Id=?`, id); err != nil {
		c.Redirect(303, "/users?status=delete-failed")
		return
	}
	c.Redirect(303, "/users?status=deleted")
}

func (h usersHandlers) renderForm(c *gin.Context, status int, title, id, email string, selected []string, message string) {
	p := auth.CurrentPrincipal(c)
	roles, err := h.roles(c, selected)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, title)
	data["EditID"] = id
	data["Email"] = email
	data["Roles"] = roles
	data["Error"] = message
	data["IsEdit"] = id != ""
	c.HTML(status, "users/form.html", data)
}

func (h usersHandlers) renderReset(c *gin.Context, status int, id, email, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), "Reset Password")
	data["TargetID"] = id
	data["TargetEmail"] = email
	data["Error"] = message
	c.HTML(status, "users/reset-password.html", data)
}

func (h usersHandlers) roles(c *gin.Context, selected []string) ([]managedRole, error) {
	set := map[string]bool{}
	for _, id := range selected {
		set[id] = true
	}
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT Id, COALESCE(Name,'') FROM roles ORDER BY Name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []managedRole{}
	for rows.Next() {
		var role managedRole
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, err
		}
		role.Selected = set[role.ID]
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (h usersHandlers) baseData(p *auth.Principal, title string) gin.H {
	roleLabel := strings.Join(p.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": roleLabel, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "users"}
}

func validSessionCSRF(p *auth.Principal, value string) bool {
	if p == nil || len(p.CSRFToken) != len(value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(p.CSRFToken), []byte(value)) == 1
}

func firstInitial(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "A"
	}
	return strings.ToUpper(string([]rune(value)[0]))
}
func redirectWithError(c *gin.Context, message string) {
	c.Redirect(http.StatusSeeOther, "/users?error="+url.QueryEscape(fmt.Sprint(message)))
}
