package httpserver

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
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

type adminNavItem struct {
	Label string
	Icon  string
	Href  string
}

type adminNavGroup struct {
	Label string
	Icon  string
	Items []adminNavItem
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

	counts := struct{ Users, Roles, Students, PendingPayments, OutstandingInvoices int64 }{}
	queries := []struct {
		target *int64
		query  string
	}{
		{&counts.Users, `SELECT COUNT(*) FROM users`},
		{&counts.Roles, `SELECT COUNT(*) FROM roles`},
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
		{"Pengguna Aktif", fmt.Sprint(counts.Users), "Akun dengan akses sistem", "", "blue"},
		{"Role Terdaftar", fmt.Sprint(counts.Roles), "Role dan hak akses", "", "violet"},
		{"Pembayaran Hari Ini", fmt.Sprint(counts.PendingPayments), "Transaksi masuk", "", "green"},
		{"Perlu Perhatian", fmt.Sprint(counts.OutstandingInvoices), "Konfigurasi belum lengkap", "", "amber"},
	}
	roleLabel := strings.Join(principal.Roles, ", ")
	if roleLabel == "" {
		roleLabel = "Admin Keuangan"
	}
	initial := "A"
	if principal.UserName != "" {
		initial = strings.ToUpper(string([]rune(principal.UserName)[0]))
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	currentDate := formatIndonesianDate(time.Now().In(jakarta))

	c.Header("Cache-Control", "no-store")
	c.HTML(http.StatusOK, "dashboard/index.html", gin.H{
		"Title":       title,
		"Description": description,
		"AppName":     h.cfg.Name,
		"Year":        time.Now().Year(),
		"User":        principal,
		"Metrics":     metrics,
		"RoleLabel":   roleLabel,
		"Initial":     initial,
		"CurrentDate": currentDate,
		"NavGroups":   buildAdminNavigation(principal),
		"ActiveMenu":  "dashboard",
	})
}

func buildAdminNavigation(p *auth.Principal) []adminNavGroup {
	can := func(permission string) bool { return p.HasRole("Super Admin") || p.HasPermission(permission) }
	groups := make([]adminNavGroup, 0, 5)
	appendGroup := func(label, icon string, candidates []struct{ label, permission string }) {
		group := adminNavGroup{Label: label, Icon: icon}
		for _, candidate := range candidates {
			if can(candidate.permission) {
				group.Items = append(group.Items, adminNavItem{Label: candidate.label, Href: "#", Icon: "fa-circle"})
			}
		}
		if len(group.Items) > 0 {
			groups = append(groups, group)
		}
	}
	appendGroup("Master Akademik", "fa-folder-open", []struct{ label, permission string }{{"Program Studi", "study-programs.view"}, {"Periode Akademik", "academic-periods.view"}, {"Golongan Mahasiswa", "student-categories.view"}, {"Jenis Biaya", "charge-types.view"}, {"Mahasiswa", "students.view"}})
	appendGroup("Kebijakan Tagihan", "fa-file-invoice", []struct{ label, permission string }{{"Tarif & Kebijakan", "billing-policies.view"}, {"Jadwal Pembayaran", "billing-policy-schedules.view"}, {"Penetapan Mahasiswa", "student-billing-policies.view"}, {"Penetapan Massal", "student-billing-policy-batches.view"}})
	appendGroup("Transaksi Keuangan", "fa-receipt", []struct{ label, permission string }{{"Tagihan Mahasiswa", "student-invoices.view"}, {"Generate Tagihan", "invoice-generation-batches.view"}, {"Pembayaran", "payments.view"}, {"Audit Log Keuangan", "audit-logs.view"}})
	if p.HasRole("Super Admin") {
		groups = append(groups, adminNavGroup{Label: "Manajemen Pengguna", Icon: "fa-users-gear", Items: []adminNavItem{{Label: "Pengguna", Href: "/users", Icon: "fa-circle"}, {Label: "Role & Akses", Href: "/roles", Icon: "fa-circle"}}})
	}
	appendGroup("Program Bantuan", "fa-hand-holding-heart", []struct{ label, permission string }{{"Program Bantuan", "funding-programs.view"}, {"Penerima Program", "student-funding-programs.view"}, {"Penyesuaian Tagihan", "student-invoice-adjustments.view"}})
	return groups
}

func formatIndonesianDate(value time.Time) string {
	months := [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	return fmt.Sprintf("%02d %s %d", value.Day(), months[value.Month()-1], value.Year())
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
