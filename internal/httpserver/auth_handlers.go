package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type authHandlers struct {
	cfg  config.Config
	auth *auth.Service
}

func (h authHandlers) showLogin(c *gin.Context) {
	if auth.CurrentPrincipal(c) != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}
	c.Header("Cache-Control", "no-store")
	existing, _ := c.Cookie(auth.LoginCSRFCookieName)
	csrfToken, err := h.auth.LoginCSRF(c.Writer, existing)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	success := ""
	if c.Query("reset") == "success" {
		success = "Kata sandi berhasil dibuat. Silakan masuk menggunakan kata sandi baru."
	}
	h.renderLogin(c, http.StatusOK, "", success, "", csrfToken, auth.SafeReturnURL(c.Query("return_url")))
}

func (h authHandlers) login(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	csrfCookie, _ := c.Cookie(auth.LoginCSRFCookieName)
	csrfToken := c.PostForm("csrf_token")
	returnURL := auth.SafeReturnURL(c.PostForm("return_url"))
	identifier := c.PostForm("email")
	if !h.auth.VerifyLoginCSRF(csrfCookie, csrfToken) {
		h.renderLogin(c, http.StatusForbidden, "Permintaan login tidak valid. Muat ulang halaman dan coba kembali.", "", identifier, csrfToken, returnURL)
		return
	}

	remember := c.PostForm("remember_me") == "on"
	rawToken, err := h.auth.Login(c.Request.Context(), identifier, c.PostForm("password"), remember, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		status := http.StatusUnauthorized
		message := "Email atau kata sandi tidak sesuai."
		if errors.Is(err, auth.ErrLockedOut) {
			status = http.StatusLocked
			message = "Akun dikunci selama 5 menit karena terlalu banyak percobaan login."
		} else if errors.Is(err, auth.ErrPasswordResetRequired) {
			status = http.StatusForbidden
			message = "Akun ini wajib membuat kata sandi baru. Hubungi administrator untuk mendapatkan tautan reset."
		} else if !errors.Is(err, auth.ErrInvalidCredentials) {
			status = http.StatusInternalServerError
			message = "Terjadi kesalahan saat memproses login."
		}
		h.renderLogin(c, status, message, "", identifier, csrfToken, returnURL)
		return
	}

	h.auth.SetSessionCookie(c.Writer, rawToken, remember)
	http.SetCookie(c.Writer, &http.Cookie{Name: auth.LoginCSRFCookieName, Value: "", Path: "/panel-admin-stieww", HttpOnly: true, MaxAge: -1, Expires: time.Unix(1, 0)})
	c.Redirect(http.StatusSeeOther, returnURL)
}

func (h authHandlers) logout(c *gin.Context) {
	principal := auth.CurrentPrincipal(c)
	if principal == nil {
		c.Redirect(http.StatusSeeOther, "/panel-admin-stieww")
		return
	}
	if !h.auth.VerifyLoginCSRF(principal.CSRFToken, c.PostForm("csrf_token")) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	rawToken, _ := c.Cookie(auth.SessionCookieName)
	_ = h.auth.Logout(c.Request.Context(), rawToken)
	h.auth.ClearSessionCookie(c.Writer)
	c.Redirect(http.StatusSeeOther, "/panel-admin-stieww")
}

func (h authHandlers) showPasswordReset(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	token := c.Query("token")
	if !h.auth.PasswordResetTokenValid(c.Request.Context(), token) {
		c.HTML(http.StatusBadRequest, "auth/reset-password.html", gin.H{"Title": "Buat Kata Sandi Baru", "Error": "Tautan reset tidak valid atau sudah kedaluwarsa."})
		return
	}
	c.HTML(http.StatusOK, "auth/reset-password.html", gin.H{"Title": "Buat Kata Sandi Baru", "Token": token})
}

func (h authHandlers) resetPassword(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	token := c.PostForm("token")
	password := c.PostForm("password")
	if password != c.PostForm("password_confirmation") {
		c.HTML(http.StatusUnprocessableEntity, "auth/reset-password.html", gin.H{"Title": "Buat Kata Sandi Baru", "Token": token, "Error": "Konfirmasi kata sandi tidak sama."})
		return
	}
	if err := h.auth.ResetPassword(c.Request.Context(), token, password); err != nil {
		status := http.StatusUnprocessableEntity
		message := err.Error()
		if errors.Is(err, auth.ErrInvalidResetToken) {
			status = http.StatusBadRequest
			message = "Tautan reset tidak valid atau sudah kedaluwarsa."
		}
		c.HTML(status, "auth/reset-password.html", gin.H{"Title": "Buat Kata Sandi Baru", "Token": token, "Error": message})
		return
	}
	c.Redirect(http.StatusSeeOther, "/panel-admin-stieww?reset=success")
}

func (h authHandlers) renderLogin(c *gin.Context, status int, message, success, identifier, csrfToken, returnURL string) {
	c.HTML(status, "auth/login.html", gin.H{
		"Title":     "Masuk Sistem",
		"AppName":   h.cfg.Name,
		"Year":      time.Now().Year(),
		"Error":     message,
		"Success":   success,
		"Email":     identifier,
		"CSRFToken": csrfToken,
		"ReturnURL": returnURL,
	})
}
