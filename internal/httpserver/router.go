package httpserver

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

func New(cfg config.Config, db *sql.DB) http.Handler {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	authService := auth.NewService(db, cfg.Environment == "production", cfg.PasswordPepper)
	router.Use(gin.Logger(), gin.Recovery(), authService.LoadPrincipal())
	templates := template.Must(template.ParseGlob(filepath.FromSlash("web/templates/*/*.html")))
	template.Must(templates.ParseGlob(filepath.FromSlash("web/templates/partials/*/*.html")))
	router.SetHTMLTemplate(templates)
	router.Static("/static", "./web/static")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home/index.html", gin.H{
			"Title":   "Beranda",
			"AppName": cfg.Name,
			"Year":    time.Now().Year(),
		})
	})

	router.GET("/privacy", func(c *gin.Context) {
		c.HTML(http.StatusOK, "home/privacy.html", gin.H{
			"Title":   "Kebijakan Privasi",
			"AppName": cfg.Name,
			"Year":    time.Now().Year(),
		})
	})

	authHTTP := authHandlers{cfg: cfg, auth: authService}
	router.GET("/panel-admin-stieww", authHTTP.showLogin)
	router.POST("/panel-admin-stieww", authHTTP.login)
	router.GET("/reset-password", authHTTP.showPasswordReset)
	router.POST("/reset-password", authHTTP.resetPassword)
	router.POST("/logout", auth.RequireAuthentication(), authHTTP.logout)

	dashboardHTTP := dashboardHandlers{cfg: cfg, db: db}
	router.GET("/dashboard", auth.RequireAuthentication(), dashboardHTTP.index)
	accountHTTP := accountHandlers{cfg: cfg}
	router.GET("/account/settings", auth.RequireAuthentication(), accountHTTP.settings)
	usersHTTP := usersHandlers{cfg: cfg, db: db, auth: authService}
	users := router.Group("/users", auth.RequireAuthentication())
	users.GET("", auth.RequirePermission("users.view"), usersHTTP.index)
	users.GET("/create", auth.RequirePermission("users.create"), usersHTTP.createForm)
	users.POST("/create", auth.RequirePermission("users.create"), usersHTTP.create)
	users.GET("/:id/edit", auth.RequirePermission("users.update"), usersHTTP.editForm)
	users.POST("/:id/edit", auth.RequirePermission("users.update"), usersHTTP.update)
	users.GET("/:id/reset-password", auth.RequirePermission("users.reset-password"), usersHTTP.resetForm)
	users.POST("/:id/reset-password", auth.RequirePermission("users.reset-password"), usersHTTP.resetPassword)
	users.POST("/:id/delete", auth.RequirePermission("users.update"), usersHTTP.delete)
	rolesHTTP := rolesHandlers{cfg: cfg, db: db}
	roles := router.Group("/roles", auth.RequireAuthentication())
	roles.GET("", auth.RequirePermission("roles.view"), rolesHTTP.index)
	roles.GET("/create", auth.RequirePermission("roles.manage"), rolesHTTP.createForm)
	roles.POST("/create", auth.RequirePermission("roles.manage"), rolesHTTP.create)
	roles.GET("/:id/edit", auth.RequirePermission("roles.manage"), rolesHTTP.editForm)
	roles.POST("/:id/edit", auth.RequirePermission("roles.manage"), rolesHTTP.update)
	roles.POST("/:id/delete", auth.RequirePermission("roles.manage"), rolesHTTP.delete)

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := contextWithTimeout(c, 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
