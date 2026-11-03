package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type accountHandlers struct {
	cfg config.Config
}

func (h accountHandlers) settings(c *gin.Context) {
	principal := auth.CurrentPrincipal(c)
	if principal == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	roleLabel := strings.Join(principal.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Pengguna"
	}
	initial := "A"
	if principal.UserName != "" {
		initial = strings.ToUpper(string([]rune(principal.UserName)[0]))
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")

	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "account/settings.html", gin.H{
		"Title":       "Pengaturan Akun",
		"AppName":     h.cfg.Name,
		"Year":        time.Now().Year(),
		"User":        principal,
		"RoleLabel":   roleLabel,
		"Initial":     initial,
		"CurrentDate": formatIndonesianDate(time.Now().In(jakarta)),
		"NavGroups":   buildAdminNavigation(principal),
		"ActiveMenu":  "account-settings",
	})
}
