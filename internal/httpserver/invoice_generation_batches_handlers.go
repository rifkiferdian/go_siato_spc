package httpserver

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type invoiceGenerationBatchesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type invoiceBatchRow struct {
	ID                                                       int64
	Number, Period, StartedAt, Status, StatusLabel, Amount   string
	TotalCandidates, TotalCreated, TotalSkipped, TotalFailed int
}
type invoiceAssignmentRow struct {
	ID                                                        int64
	NIM, StudentName, Program, PolicyCode, PolicyName, Amount string
	CohortYear                                                int
}
type invoicePreviewRow struct {
	AssignmentID, ScheduleID                   int64
	Student, Policy, Schedule, Amount, Message string
	Sequence                                   int
	Ready                                      bool
}
type invoiceBatchDetail struct {
	ID                                                                         int64
	Number, Period, Status, StatusLabel, Amount, CancellationReason, StartedAt string
	TotalCandidates, TotalCreated, TotalSkipped, TotalFailed                   int
}
type invoiceBatchDetailItem struct{ Status, StatusLabel, Amount, Message, InvoiceNumber string }

func (h invoiceGenerationBatchesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT b.id,b.batch_number,COALESCE(ap.code,'-'),DATE_FORMAT(CONVERT_TZ(b.started_at,'+00:00','+07:00'),'%d %b %Y %H:%i'),b.status,b.total_candidates,b.total_created,b.total_skipped,b.total_failed,b.total_amount FROM invoice_generation_batches b LEFT JOIN academic_periods ap ON ap.id=b.academic_period_id ORDER BY b.started_at DESC LIMIT 100`)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []invoiceBatchRow{}
	metrics := struct {
		Total, Created int
		Amount         string
	}{}
	var totalAmount float64
	for rows.Next() {
		var x invoiceBatchRow
		var amount float64
		if err = rows.Scan(&x.ID, &x.Number, &x.Period, &x.StartedAt, &x.Status, &x.TotalCandidates, &x.TotalCreated, &x.TotalSkipped, &x.TotalFailed, &amount); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.StatusLabel = invoiceBatchStatusLabel(x.Status)
		x.Amount = formatRupiah(amount)
		items = append(items, x)
		metrics.Total++
		metrics.Created += x.TotalCreated
		totalAmount += amount
	}
	metrics.Amount = formatRupiah(totalAmount)
	data := h.baseData(p, "Generate Tagihan")
	data["Items"] = items
	data["Metrics"] = metrics
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-invoices.generate")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "invoice-generation-batches/index.html", data)
}
func (h invoiceGenerationBatchesHandlers) create(c *gin.Context) {
	program := c.Query("study_program_id")
	year := c.Query("cohort_year")
	policy := c.Query("billing_policy_id")
	q := `SELECT sbp.id,s.nim,s.name,sp.code,s.cohort_year,bp.code,bp.name,sbp.effective_amount FROM student_billing_policies sbp JOIN students s ON s.id=sbp.student_id JOIN study_programs sp ON sp.id=s.study_program_id JOIN billing_policies bp ON bp.id=sbp.billing_policy_id WHERE sbp.status='ACTIVE' AND bp.is_active=1`
	args := []any{}
	if parsePositiveID(program) > 0 {
		q += ` AND s.study_program_id=?`
		args = append(args, program)
	}
	if parsePositiveID(year) > 0 {
		q += ` AND s.cohort_year=?`
		args = append(args, year)
	}
	if parsePositiveID(policy) > 0 {
		q += ` AND sbp.billing_policy_id=?`
		args = append(args, policy)
	}
	q += ` ORDER BY s.nim`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []invoiceAssignmentRow{}
	for rows.Next() {
		var x invoiceAssignmentRow
		var amount float64
		if err = rows.Scan(&x.ID, &x.NIM, &x.StudentName, &x.Program, &x.CohortYear, &x.PolicyCode, &x.PolicyName, &amount); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Amount = formatRupiah(amount)
		items = append(items, x)
	}
	data := h.baseData(auth.CurrentPrincipal(c), "Pilih Target Tagihan")
	data["Items"] = items
	data["Programs"] = h.options(c, `SELECT id,name FROM study_programs ORDER BY name`)
	data["Years"] = h.strings(c, `SELECT DISTINCT cohort_year FROM students ORDER BY cohort_year DESC`)
	data["Policies"] = h.options(c, `SELECT id,CONCAT(code,' - ',name) FROM billing_policies WHERE is_active=1 ORDER BY name`)
	data["SelectedProgram"] = program
	data["SelectedYear"] = year
	data["SelectedPolicy"] = policy
	data["Error"] = c.Query("error")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "invoice-generation-batches/create.html", data)
}
func (h invoiceGenerationBatchesHandlers) preview(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	ids := positiveIDs(c.PostFormArray("assignment_ids"))
	if len(ids) == 0 {
		c.Redirect(303, "/invoice-generation-batches/create?error=select")
		return
	}
	items, err := h.buildPreview(c, ids)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, "Preview Generate Tagihan")
	data["Items"] = items
	data["AssignmentIDs"] = ids
	ready := 0
	var amount float64
	for _, x := range items {
		if x.Ready {
			ready++
			v, _ := parseRupiahRaw(x.Amount)
			amount += v
		}
	}
	data["ReadyCount"] = ready
	data["SkippedCount"] = len(items) - ready
	data["TotalAmount"] = formatRupiah(amount)
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "invoice-generation-batches/preview.html", data)
}
func (h invoiceGenerationBatchesHandlers) generate(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	ids := positiveIDs(c.PostFormArray("assignment_ids"))
	items, err := h.buildPreview(c, ids)
	if err != nil || len(items) == 0 {
		c.Redirect(303, "/invoice-generation-batches/create?error=select")
		return
	}
	var periodID int64
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT bp.academic_period_id FROM student_billing_policies sbp JOIN billing_policies bp ON bp.id=sbp.billing_policy_id WHERE sbp.id=?`, ids[0]).Scan(&periodID)
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(c.Request.Context(), `INSERT INTO invoice_generation_batches(batch_number,academic_period_id,status,total_candidates,total_created,total_skipped,total_failed,total_amount,initiated_by_id,started_at) VALUES(?,?,'PROCESSING',?,0,0,0,0,?,UTC_TIMESTAMP(6))`, newInvoiceBatchNumber(), periodID, len(items), p.UserID)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	batchID, _ := r.LastInsertId()
	created, skipped := 0, 0
	var total float64
	next := h.nextSequence(c)
	for _, item := range items {
		amount, _ := parseRupiahRaw(item.Amount)
		status := "SKIPPED"
		var invoiceID any
		if item.Ready {
			var studentID, period, charge int64
			var description string
			var due time.Time
			if err = tx.QueryRowContext(c.Request.Context(), `SELECT sbp.student_id,bp.academic_period_id,bp.charge_type_id,bps.description,bps.due_date FROM student_billing_policies sbp JOIN billing_policies bp ON bp.id=sbp.billing_policy_id JOIN billing_policy_schedules bps ON bps.id=? WHERE sbp.id=?`, item.ScheduleID, item.AssignmentID).Scan(&studentID, &period, &charge, &description, &due); err != nil {
				c.AbortWithError(500, err)
				return
			}
			number := fmt.Sprintf("INV-%d-%04d", time.Now().Year(), next)
			next++
			invoiceStatus := "UNPAID"
			if due.Before(time.Now().Truncate(24 * time.Hour)) {
				invoiceStatus = "OVERDUE"
			}
			inv, err := tx.ExecContext(c.Request.Context(), `INSERT INTO student_invoices(invoice_number,student_id,academic_period_id,charge_type_id,student_billing_policy_id,billing_policy_schedule_id,sequence_no,description,original_amount,discount_amount,penalty_amount,final_amount,paid_amount,due_date,status,source_type,created_at,updated_at,version,invoice_generation_batch_id) VALUES(?,?,?,?,?,?,?,?,?,0,0,?,0,?,?,'AUTOMATIC',UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),1,?)`, number, studentID, period, charge, item.AssignmentID, item.ScheduleID, item.Sequence, description, amount, amount, due, invoiceStatus, batchID)
			if err != nil {
				c.AbortWithError(500, err)
				return
			}
			id, _ := inv.LastInsertId()
			invoiceID = id
			status = "CREATED"
			created++
			total += amount
		} else {
			skipped++
		}
		if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO invoice_generation_batch_items(invoice_generation_batch_id,student_billing_policy_id,billing_policy_schedule_id,student_invoice_id,status,proposed_amount,message,created_at) VALUES(?,?,?,?,?,?,?,UTC_TIMESTAMP(6))`, batchID, item.AssignmentID, item.ScheduleID, invoiceID, status, amount, item.Message); err != nil {
			c.AbortWithError(500, err)
			return
		}
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE invoice_generation_batches SET status='COMPLETED',total_created=?,total_skipped=?,total_amount=?,completed_at=UTC_TIMESTAMP(6) WHERE id=?`, created, skipped, total, batchID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, fmt.Sprintf("/invoice-generation-batches/%d?status=created", batchID))
}
func (h invoiceGenerationBatchesHandlers) details(c *gin.Context) {
	id := parsePositiveID(c.Param("id"))
	var d invoiceBatchDetail
	var amount float64
	var reason sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT b.id,b.batch_number,COALESCE(ap.code,'-'),b.status,b.total_candidates,b.total_created,b.total_skipped,b.total_failed,b.total_amount,b.cancellation_reason,DATE_FORMAT(CONVERT_TZ(b.started_at,'+00:00','+07:00'),'%d %b %Y %H:%i') FROM invoice_generation_batches b LEFT JOIN academic_periods ap ON ap.id=b.academic_period_id WHERE b.id=?`, id).Scan(&d.ID, &d.Number, &d.Period, &d.Status, &d.TotalCandidates, &d.TotalCreated, &d.TotalSkipped, &d.TotalFailed, &amount, &reason, &d.StartedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	d.Amount = formatRupiah(amount)
	d.StatusLabel = invoiceBatchStatusLabel(d.Status)
	d.CancellationReason = reason.String
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT i.status,i.proposed_amount,COALESCE(i.message,''),COALESCE(si.invoice_number,'') FROM invoice_generation_batch_items i LEFT JOIN student_invoices si ON si.id=i.student_invoice_id WHERE i.invoice_generation_batch_id=? ORDER BY i.id`, id)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []invoiceBatchDetailItem{}
	for rows.Next() {
		var x invoiceBatchDetailItem
		var value float64
		if err = rows.Scan(&x.Status, &value, &x.Message, &x.InvoiceNumber); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Amount = formatRupiah(value)
		if x.Status == "CREATED" {
			x.StatusLabel = "Dibuat"
		} else {
			x.StatusLabel = "Dilewati"
		}
		items = append(items, x)
	}
	p := auth.CurrentPrincipal(c)
	data := h.baseData(p, "Detail Generate Tagihan")
	data["Batch"] = d
	data["Items"] = items
	data["Status"] = c.Query("status")
	data["CanCancel"] = (p.HasRole("Super Admin") || p.HasPermission("invoice-generation-batches.cancel")) && d.Status != "CANCELLED"
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "invoice-generation-batches/details.html", data)
}
func (h invoiceGenerationBatchesHandlers) cancel(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	reason := strings.TrimSpace(c.PostForm("reason"))
	if reason == "" || len([]rune(reason)) > 500 {
		c.Redirect(303, fmt.Sprintf("/invoice-generation-batches/%d?status=cancel-invalid", id))
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(c.Request.Context(), `SELECT status FROM invoice_generation_batches WHERE id=? FOR UPDATE`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	} else if err != nil {
		c.AbortWithError(500, err)
		return
	}
	var verified int
	if err = tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM payment_allocations pa JOIN payments p ON p.id=pa.payment_id JOIN student_invoices i ON i.id=pa.student_invoice_id WHERE i.invoice_generation_batch_id=? AND p.status='VERIFIED'`, id).Scan(&verified); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if verified > 0 {
		c.Redirect(303, fmt.Sprintf("/invoice-generation-batches/%d?status=has-payments", id))
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE student_invoices SET status='CANCELLED',version=version+1,updated_at=UTC_TIMESTAMP(6) WHERE invoice_generation_batch_id=?`, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE invoice_generation_batches SET status='CANCELLED',cancellation_reason=?,cancelled_by_id=?,cancelled_at=UTC_TIMESTAMP(6) WHERE id=?`, reason, p.UserID, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,reason,performed_by_id,performed_at) VALUES('InvoiceGenerationBatch',?,'CANCELLED',?,?,UTC_TIMESTAMP(6))`, id, reason, p.UserID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, fmt.Sprintf("/invoice-generation-batches/%d?status=cancelled", id))
}

func (h invoiceGenerationBatchesHandlers) buildPreview(c *gin.Context, ids []int64) ([]invoicePreviewRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	marks := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, v := range ids {
		args[i] = v
	}
	q := `SELECT sbp.id,s.nim,s.name,bp.name,bps.id,bps.sequence_no,bps.description,bps.amount,(s.status='ACTIVE'),EXISTS(SELECT 1 FROM student_invoices i WHERE i.student_id=s.id AND i.billing_policy_schedule_id=bps.id) FROM student_billing_policies sbp JOIN students s ON s.id=sbp.student_id JOIN billing_policies bp ON bp.id=sbp.billing_policy_id JOIN billing_policy_schedules bps ON bps.billing_policy_id=bp.id WHERE sbp.status='ACTIVE' AND sbp.id IN (` + marks + `) ORDER BY s.nim,bps.sequence_no`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []invoicePreviewRow{}
	for rows.Next() {
		var x invoicePreviewRow
		var amount float64
		var active, exists bool
		var nim, name, description string
		if err = rows.Scan(&x.AssignmentID, &nim, &name, &x.Policy, &x.ScheduleID, &x.Sequence, &description, &amount, &active, &exists); err != nil {
			return nil, err
		}
		x.Student = nim + " - " + name
		x.Schedule = fmt.Sprintf("Termin %d - %s", x.Sequence, description)
		x.Amount = strconv.FormatFloat(amount, 'f', 2, 64)
		x.Ready = active && !exists
		if exists {
			x.Message = "Tagihan termin sudah ada."
		} else if !active {
			x.Message = "Mahasiswa tidak aktif."
		} else {
			x.Message = "Siap dibuat"
		}
		out = append(out, x)
	}
	return out, nil
}
func (h invoiceGenerationBatchesHandlers) nextSequence(c *gin.Context) int {
	prefix := fmt.Sprintf("INV-%d-", time.Now().Year())
	var n int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE invoice_number LIKE ?`, prefix+"%").Scan(&n)
	return n + 1
}
func (h invoiceGenerationBatchesHandlers) options(c *gin.Context, q string) []billingPolicyOption {
	rows, err := h.db.QueryContext(c.Request.Context(), q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []billingPolicyOption{}
	for rows.Next() {
		var x billingPolicyOption
		if rows.Scan(&x.ID, &x.Label) == nil {
			out = append(out, x)
		}
	}
	return out
}
func (h invoiceGenerationBatchesHandlers) strings(c *gin.Context, q string) []string {
	rows, err := h.db.QueryContext(c.Request.Context(), q)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var x string
		if rows.Scan(&x) == nil {
			out = append(out, x)
		}
	}
	return out
}
func (h invoiceGenerationBatchesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "invoice-generation-batches"}
}
func positiveIDs(raw []string) []int64 {
	seen := map[int64]bool{}
	out := []int64{}
	for _, v := range raw {
		if id := parsePositiveID(v); id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
func parseRupiahRaw(v string) (float64, error) { return strconv.ParseFloat(v, 64) }
func newInvoiceBatchNumber() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("INVGEN-%d-%s", time.Now().Year(), strings.ToUpper(hex.EncodeToString(b)))
}
func invoiceBatchStatusLabel(v string) string {
	if v == "CANCELLED" {
		return "Dibatalkan"
	}
	if v == "PROCESSING" {
		return "Diproses"
	}
	return "Selesai"
}
