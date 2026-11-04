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
	templates := template.Must(template.New("").Funcs(template.FuncMap{"dict": func() map[string]string { return map[string]string{} }}).ParseGlob(filepath.FromSlash("web/templates/*/*.html")))
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
	studyProgramsHTTP := studyProgramsHandlers{cfg: cfg, db: db}
	studyPrograms := router.Group("/study-programs", auth.RequireAuthentication())
	studyPrograms.GET("", auth.RequirePermission("study-programs.view"), studyProgramsHTTP.index)
	studyPrograms.GET("/create", auth.RequirePermission("study-programs.create"), studyProgramsHTTP.createForm)
	studyPrograms.POST("/create", auth.RequirePermission("study-programs.create"), studyProgramsHTTP.create)
	studyPrograms.GET("/:id/edit", auth.RequirePermission("study-programs.update"), studyProgramsHTTP.editForm)
	studyPrograms.POST("/:id/edit", auth.RequirePermission("study-programs.update"), studyProgramsHTTP.update)
	studyPrograms.POST("/:id/delete", auth.RequirePermission("study-programs.delete"), studyProgramsHTTP.delete)
	academicPeriodsHTTP := academicPeriodsHandlers{cfg: cfg, db: db}
	academicPeriods := router.Group("/academic-periods", auth.RequireAuthentication())
	academicPeriods.GET("", auth.RequirePermission("academic-periods.view"), academicPeriodsHTTP.index)
	academicPeriods.GET("/create", auth.RequirePermission("academic-periods.create"), academicPeriodsHTTP.createForm)
	academicPeriods.POST("/create", auth.RequirePermission("academic-periods.create"), academicPeriodsHTTP.create)
	academicPeriods.GET("/:id/edit", auth.RequirePermission("academic-periods.update"), academicPeriodsHTTP.editForm)
	academicPeriods.POST("/:id/edit", auth.RequirePermission("academic-periods.update"), academicPeriodsHTTP.update)
	academicPeriods.POST("/:id/delete", auth.RequirePermission("academic-periods.delete"), academicPeriodsHTTP.delete)
	studentCategoriesHTTP := studentCategoriesHandlers{cfg: cfg, db: db}
	studentCategories := router.Group("/student-categories", auth.RequireAuthentication())
	studentCategories.GET("", auth.RequirePermission("student-categories.view"), studentCategoriesHTTP.index)
	studentCategories.GET("/create", auth.RequirePermission("student-categories.create"), studentCategoriesHTTP.createForm)
	studentCategories.POST("/create", auth.RequirePermission("student-categories.create"), studentCategoriesHTTP.create)
	studentCategories.GET("/:id/edit", auth.RequirePermission("student-categories.update"), studentCategoriesHTTP.editForm)
	studentCategories.POST("/:id/edit", auth.RequirePermission("student-categories.update"), studentCategoriesHTTP.update)
	studentCategories.POST("/:id/delete", auth.RequirePermission("student-categories.delete"), studentCategoriesHTTP.delete)
	chargeTypesHTTP := chargeTypesHandlers{cfg: cfg, db: db}
	chargeTypes := router.Group("/charge-types", auth.RequireAuthentication())
	chargeTypes.GET("", auth.RequirePermission("charge-types.view"), chargeTypesHTTP.index)
	chargeTypes.GET("/create", auth.RequirePermission("charge-types.create"), chargeTypesHTTP.createForm)
	chargeTypes.POST("/create", auth.RequirePermission("charge-types.create"), chargeTypesHTTP.create)
	chargeTypes.GET("/:id/edit", auth.RequirePermission("charge-types.update"), chargeTypesHTTP.editForm)
	chargeTypes.POST("/:id/edit", auth.RequirePermission("charge-types.update"), chargeTypesHTTP.update)
	chargeTypes.POST("/:id/delete", auth.RequirePermission("charge-types.delete"), chargeTypesHTTP.delete)
	studentsHTTP := studentsHandlers{cfg: cfg, db: db}
	students := router.Group("/students", auth.RequireAuthentication())
	students.GET("", auth.RequirePermission("students.view"), studentsHTTP.index)
	students.GET("/create", auth.RequirePermission("students.create"), studentsHTTP.createForm)
	students.POST("/create", auth.RequirePermission("students.create"), studentsHTTP.create)
	students.GET("/:id", auth.RequirePermission("students.view"), studentsHTTP.details)
	students.GET("/:id/edit", auth.RequirePermission("students.update"), studentsHTTP.editForm)
	students.POST("/:id/edit", auth.RequirePermission("students.update"), studentsHTTP.update)
	students.POST("/:id/delete", auth.RequirePermission("students.delete"), studentsHTTP.delete)
	billingPoliciesHTTP := billingPoliciesHandlers{cfg: cfg, db: db}
	billingPolicies := router.Group("/billing-policies", auth.RequireAuthentication())
	billingPolicies.GET("", auth.RequirePermission("billing-policies.view"), billingPoliciesHTTP.index)
	billingPolicies.GET("/create", auth.RequirePermission("billing-policies.create"), billingPoliciesHTTP.createForm)
	billingPolicies.POST("/create", auth.RequirePermission("billing-policies.create"), billingPoliciesHTTP.create)
	billingPolicies.GET("/:id/edit", auth.RequirePermission("billing-policies.update"), billingPoliciesHTTP.editForm)
	billingPolicies.POST("/:id/edit", auth.RequirePermission("billing-policies.update"), billingPoliciesHTTP.update)
	billingPolicies.POST("/:id/delete", auth.RequirePermission("billing-policies.delete"), billingPoliciesHTTP.delete)
	billingPolicySchedulesHTTP := billingPolicySchedulesHandlers{cfg: cfg, db: db}
	billingPolicySchedules := router.Group("/billing-policy-schedules", auth.RequireAuthentication())
	billingPolicySchedules.GET("", auth.RequirePermission("billing-policy-schedules.view"), billingPolicySchedulesHTTP.index)
	billingPolicySchedules.GET("/create", auth.RequirePermission("billing-policy-schedules.create"), billingPolicySchedulesHTTP.createForm)
	billingPolicySchedules.POST("/create", auth.RequirePermission("billing-policy-schedules.create"), billingPolicySchedulesHTTP.create)
	billingPolicySchedules.GET("/:id/edit", auth.RequirePermission("billing-policy-schedules.update"), billingPolicySchedulesHTTP.editForm)
	billingPolicySchedules.POST("/:id/edit", auth.RequirePermission("billing-policy-schedules.update"), billingPolicySchedulesHTTP.update)
	billingPolicySchedules.POST("/:id/delete", auth.RequirePermission("billing-policy-schedules.delete"), billingPolicySchedulesHTTP.delete)
	studentBillingPoliciesHTTP := studentBillingPoliciesHandlers{cfg: cfg, db: db}
	studentBillingPolicies := router.Group("/student-billing-policies", auth.RequireAuthentication())
	studentBillingPolicies.GET("", auth.RequirePermission("student-billing-policies.view"), studentBillingPoliciesHTTP.index)
	studentBillingPolicies.GET("/create", auth.RequirePermission("student-billing-policies.create"), studentBillingPoliciesHTTP.createForm)
	studentBillingPolicies.POST("/create", auth.RequirePermission("student-billing-policies.create"), studentBillingPoliciesHTTP.create)
	studentBillingPolicies.GET("/:id/edit", auth.RequirePermission("student-billing-policies.update"), studentBillingPoliciesHTTP.editForm)
	studentBillingPolicies.POST("/:id/edit", auth.RequirePermission("student-billing-policies.update"), studentBillingPoliciesHTTP.update)
	studentBillingPolicies.POST("/:id/delete", auth.RequirePermission("student-billing-policies.delete"), studentBillingPoliciesHTTP.delete)
	studentBillingPolicyBatchesHTTP := studentBillingPolicyBatchesHandlers{cfg: cfg, db: db}
	studentBillingPolicyBatches := router.Group("/student-billing-policy-batches", auth.RequireAuthentication())
	studentBillingPolicyBatches.GET("", auth.RequirePermission("student-billing-policy-batches.view"), studentBillingPolicyBatchesHTTP.index)
	studentBillingPolicyBatches.GET("/create", auth.RequirePermission("student-billing-policy-batches.create"), studentBillingPolicyBatchesHTTP.createForm)
	studentBillingPolicyBatches.POST("/generate", auth.RequirePermission("student-billing-policy-batches.create"), studentBillingPolicyBatchesHTTP.generate)
	studentBillingPolicyBatches.GET("/:id", auth.RequirePermission("student-billing-policy-batches.view"), studentBillingPolicyBatchesHTTP.details)
	studentBillingPolicyBatches.POST("/:id/cancel", auth.RequirePermission("student-billing-policy-batches.cancel"), studentBillingPolicyBatchesHTTP.cancel)
	studentInvoicesHTTP := studentInvoicesHandlers{cfg: cfg, db: db}
	studentInvoices := router.Group("/student-invoices", auth.RequireAuthentication())
	studentInvoices.GET("", auth.RequirePermission("student-invoices.view"), studentInvoicesHTTP.index)
	studentInvoices.GET("/create", auth.RequirePermission("student-invoices.create"), studentInvoicesHTTP.createForm)
	studentInvoices.POST("/create", auth.RequirePermission("student-invoices.create"), studentInvoicesHTTP.create)
	studentInvoices.GET("/:id/print", auth.RequirePermission("student-invoices.view"), studentInvoicesHTTP.print)
	studentInvoices.GET("/:id/edit", auth.RequirePermission("student-invoices.update"), studentInvoicesHTTP.editForm)
	studentInvoices.POST("/:id/edit", auth.RequirePermission("student-invoices.update"), studentInvoicesHTTP.update)
	studentInvoices.POST("/:id/delete", auth.RequirePermission("student-invoices.delete"), studentInvoicesHTTP.delete)
	invoiceGenerationBatchesHTTP := invoiceGenerationBatchesHandlers{cfg: cfg, db: db}
	invoiceGenerationBatches := router.Group("/invoice-generation-batches", auth.RequireAuthentication())
	invoiceGenerationBatches.GET("", auth.RequirePermission("invoice-generation-batches.view"), invoiceGenerationBatchesHTTP.index)
	invoiceGenerationBatches.GET("/create", auth.RequirePermission("student-invoices.generate"), invoiceGenerationBatchesHTTP.create)
	invoiceGenerationBatches.POST("/preview", auth.RequirePermission("student-invoices.generate"), invoiceGenerationBatchesHTTP.preview)
	invoiceGenerationBatches.POST("/generate", auth.RequirePermission("student-invoices.generate"), invoiceGenerationBatchesHTTP.generate)
	invoiceGenerationBatches.GET("/:id", auth.RequirePermission("invoice-generation-batches.view"), invoiceGenerationBatchesHTTP.details)
	invoiceGenerationBatches.POST("/:id/cancel", auth.RequirePermission("invoice-generation-batches.cancel"), invoiceGenerationBatchesHTTP.cancel)
	paymentsHTTP := paymentsHandlers{cfg: cfg, db: db}
	payments := router.Group("/payments", auth.RequireAuthentication())
	payments.GET("", auth.RequirePermission("payments.view"), paymentsHTTP.index)
	payments.GET("/create", auth.RequirePermission("payments.create"), paymentsHTTP.createForm)
	payments.POST("/create", auth.RequirePermission("payments.create"), paymentsHTTP.create)
	payments.GET("/:id/print", auth.RequirePermission("payments.view"), paymentsHTTP.print)
	payments.POST("/:id/verify", auth.RequirePermission("payments.verify"), paymentsHTTP.verify)
	payments.POST("/:id/reject", auth.RequirePermission("payments.reject"), paymentsHTTP.reject)
	payments.POST("/:id/cancel", auth.RequirePermission("payments.cancel"), paymentsHTTP.cancel)
	auditLogsHTTP := auditLogsHandlers{cfg: cfg, db: db}
	auditLogs := router.Group("/audit-logs", auth.RequireAuthentication(), auth.RequirePermission("audit-logs.view"))
	auditLogs.GET("", auditLogsHTTP.index)
	auditLogs.GET("/:id", auditLogsHTTP.details)

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
