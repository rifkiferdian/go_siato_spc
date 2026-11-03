package httpserver

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/config"
)

func New(cfg config.Config, db *sql.DB) http.Handler {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.LoadHTMLGlob("web/templates/**/*.html")
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
