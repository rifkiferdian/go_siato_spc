package httpserver

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type dashboardHandlers struct {
	cfg config.Config
	db  *sql.DB
}

type dashboardMetric struct {
	Label string
	Value string
	Note  string
	Icon  string
	Color string
}

func (h dashboardHandlers) index(c *gin.Context) {
	principal := auth.CurrentPrincipal(c)
	if principal == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	title, description, allowed := dashboardIdentity(principal)
	if !allowed {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	counts := struct{ Users, Students, PendingPayments, OutstandingInvoices int64 }{}
	queries := []struct {
		target *int64
		query  string
	}{
		{&counts.Users, `SELECT COUNT(*) FROM users`},
		{&counts.Students, `SELECT COUNT(*) FROM students WHERE status = 'ACTIVE'`},
		{&counts.PendingPayments, `SELECT COUNT(*) FROM payments WHERE status = 'PENDING'`},
		{&counts.OutstandingInvoices, `SELECT COUNT(*) FROM student_invoices WHERE status IN ('UNPAID','PARTIALLY_PAID','OVERDUE')`},
	}
	for _, item := range queries {
		if err := h.db.QueryRowContext(c.Request.Context(), item.query).Scan(item.target); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("memuat dashboard: %w", err))
			return
		}
	}

	metrics := []dashboardMetric{
		{"Pengguna Sistem", fmt.Sprint(counts.Users), "Akun terdaftar", "fa-users", "blue"},
		{"Mahasiswa Aktif", fmt.Sprint(counts.Students), "Data mahasiswa aktif", "fa-user-graduate", "violet"},
		{"Pembayaran Pending", fmt.Sprint(counts.PendingPayments), "Perlu diverifikasi", "fa-clock", "amber"},
		{"Tagihan Berjalan", fmt.Sprint(counts.OutstandingInvoices), "Belum diselesaikan", "fa-file-invoice-dollar", "red"},
	}

	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "dashboard/index.html", gin.H{
		"Title":       title,
		"Description": description,
		"AppName":     h.cfg.Name,
		"Year":        time.Now().Year(),
		"User":        principal,
		"Metrics":     metrics,
	})
}

func dashboardIdentity(p *auth.Principal) (string, string, bool) {
	switch {
	case p.HasRole("Super Admin"):
		return "Dashboard Super Admin", "Ringkasan sistem, akses pengguna, dan kesiapan operasional.", true
	case p.HasRole("Manager Keuangan"):
		return "Dashboard Manager Keuangan", "Pemantauan piutang, persetujuan, dan tren keuangan.", true
	case p.HasRole("Rektor"):
		return "Dashboard Eksekutif", "Ringkasan indikator keuangan dan capaian institusi.", true
	case p.HasRole("Admin Keuangan") || p.HasPermission("dashboard.finance.view"):
		return "Dashboard Admin Keuangan", "Operasional pembayaran, tagihan, dan verifikasi harian.", true
	case p.HasRole("Staff Akademik") || p.HasPermission("dashboard.academic.view"):
		return "Dashboard Staff Akademik", "Pengelolaan data mahasiswa dan master akademik.", true
	case p.HasRole("Marketing / PMB") || p.HasPermission("dashboard.marketing.view"):
		return "Dashboard Marketing / PMB", "Pemantauan mahasiswa dan program bantuan marketing.", true
	case p.HasPermission("dashboard.executive.view"):
		return "Dashboard Eksekutif", "Ringkasan indikator keuangan dan capaian institusi.", true
	default:
		return "", "", false
	}
}
