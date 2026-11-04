package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type auditLogsHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type auditLogItem struct {
	ID, EntityID                                                             int64
	EntityType, EntityLabel, Action, ActionLabel, Reason, Actor, PerformedAt string
}
type auditLogMetrics struct{ Total, Today, Critical, Actors int }
type auditLogDetails struct {
	auditLogItem
	OldValues, NewValues string
}

func (h auditLogsHandlers) base(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(loc)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "audit-logs"}
}

func (h auditLogsHandlers) index(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	entity := strings.TrimSpace(c.Query("entity_type"))
	action := strings.ToUpper(strings.TrimSpace(c.Query("action")))
	from := strings.TrimSpace(c.Query("from"))
	to := strings.TrimSpace(c.Query("to"))
	q := `SELECT a.id,a.entity_type,a.entity_id,a.action,COALESCE(a.reason,''),COALESCE(u.email,'Sistem / pengguna dihapus'),DATE_FORMAT(a.performed_at,'%d %b %Y %H:%i:%s') FROM audit_logs a LEFT JOIN users u ON u.id=a.performed_by_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (a.entity_type LIKE ? OR CAST(a.entity_id AS CHAR) LIKE ? OR a.reason LIKE ? OR u.email LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if entity != "" {
		q += ` AND a.entity_type=?`
		args = append(args, entity)
	}
	if action != "" {
		q += ` AND a.action=?`
		args = append(args, action)
	}
	if _, e := time.Parse("2006-01-02", from); e == nil {
		q += ` AND a.performed_at>=?`
		args = append(args, from+" 00:00:00")
	}
	if _, e := time.Parse("2006-01-02", to); e == nil {
		q += ` AND a.performed_at<DATE_ADD(?,INTERVAL 1 DAY)`
		args = append(args, to+" 00:00:00")
	}
	q += ` ORDER BY a.performed_at DESC,a.id DESC LIMIT 500`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []auditLogItem{}
	for rows.Next() {
		var x auditLogItem
		if rows.Scan(&x.ID, &x.EntityType, &x.EntityID, &x.Action, &x.Reason, &x.Actor, &x.PerformedAt) != nil {
			continue
		}
		x.EntityLabel = auditEntityLabel(x.EntityType)
		x.ActionLabel = auditActionLabel(x.Action)
		items = append(items, x)
	}
	var m auditLogMetrics
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(DATE(performed_at)=CURDATE()),0),COALESCE(SUM(action IN ('REJECTED','CANCELLED','DELETED')),0),COUNT(DISTINCT performed_by_id) FROM audit_logs`).Scan(&m.Total, &m.Today, &m.Critical, &m.Actors)
	d := h.base(auth.CurrentPrincipal(c), "Audit Log Keuangan")
	d["Items"] = items
	d["Metrics"] = m
	d["Search"] = search
	d["SelectedEntity"] = entity
	d["SelectedAction"] = action
	d["From"] = from
	d["To"] = to
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "audit-logs/index.html", d)
}

func (h auditLogsHandlers) details(c *gin.Context) {
	var x auditLogDetails
	var oldJSON, newJSON sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT a.id,a.entity_type,a.entity_id,a.action,COALESCE(a.reason,''),COALESCE(u.email,'Sistem / pengguna dihapus'),DATE_FORMAT(a.performed_at,'%d %b %Y %H:%i:%s'),a.old_values_json,a.new_values_json FROM audit_logs a LEFT JOIN users u ON u.id=a.performed_by_id WHERE a.id=?`, c.Param("id")).Scan(&x.ID, &x.EntityType, &x.EntityID, &x.Action, &x.Reason, &x.Actor, &x.PerformedAt, &oldJSON, &newJSON)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	x.EntityLabel = auditEntityLabel(x.EntityType)
	x.ActionLabel = auditActionLabel(x.Action)
	x.OldValues = prettyAuditJSON(oldJSON.String)
	x.NewValues = prettyAuditJSON(newJSON.String)
	d := h.base(auth.CurrentPrincipal(c), "Detail Audit Log")
	d["Log"] = x
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "audit-logs/details.html", d)
}
func prettyAuditJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "—"
	}
	var v any
	if json.Unmarshal([]byte(raw), &v) != nil {
		return raw
	}
	out, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return raw
	}
	return string(out)
}
func auditEntityLabel(v string) string {
	switch v {
	case "Payment":
		return "Pembayaran"
	case "StudentInvoice":
		return "Tagihan Mahasiswa"
	case "StudentInvoiceAdjustment":
		return "Penyesuaian Tagihan"
	case "InvoiceGenerationBatch":
		return "Generate Tagihan"
	case "StudentBillingPolicy":
		return "Penetapan Mahasiswa"
	}
	return v
}
func auditActionLabel(v string) string {
	switch v {
	case "CREATED":
		return "Dibuat"
	case "VERIFIED":
		return "Diverifikasi"
	case "REJECTED":
		return "Ditolak"
	case "CANCELLED":
		return "Dibatalkan"
	case "UPDATED":
		return "Diperbarui"
	case "DELETED":
		return "Dihapus"
	}
	return v
}
