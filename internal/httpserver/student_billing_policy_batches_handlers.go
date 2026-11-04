package httpserver

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studentBillingPolicyBatchesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type billingBatchItem struct {
	ID                                                             int64
	Number, PolicyCode, PolicyName, CreatedAt, Status, StatusLabel string
	TotalCandidates, TotalCreated, TotalSkipped, TotalFailed       int
}
type billingBatchCandidate struct {
	ID                                    int64
	NIM, Name, Program, Category, Message string
	CohortYear                            int
	Ready                                 bool
}
type billingBatchDetailItem struct{ NIM, Name, Amount, Status, StatusLabel, Message string }
type billingBatchForm struct{ PolicyID, ProgramID, CohortYear, CategoryID, CohortClass string }
type billingBatchDetail struct {
	ID                                                                                 int64
	Number, PolicyCode, PolicyName, Status, StatusLabel, CancellationReason, CreatedAt string
	TotalCandidates, TotalCreated, TotalSkipped, TotalFailed                           int
}

func (h studentBillingPolicyBatchesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT b.id,b.batch_number,bp.code,bp.name,DATE_FORMAT(CONVERT_TZ(b.created_at,'+00:00','+07:00'),'%d %b %Y %H:%i'),b.status,b.total_candidates,b.total_created,b.total_skipped,b.total_failed FROM student_billing_policy_batches b JOIN billing_policies bp ON bp.id=b.billing_policy_id ORDER BY b.created_at DESC LIMIT 100`)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []billingBatchItem{}
	metrics := struct{ Total, Created, Skipped int }{}
	for rows.Next() {
		var x billingBatchItem
		if err = rows.Scan(&x.ID, &x.Number, &x.PolicyCode, &x.PolicyName, &x.CreatedAt, &x.Status, &x.TotalCandidates, &x.TotalCreated, &x.TotalSkipped, &x.TotalFailed); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.StatusLabel = batchStatusLabel(x.Status)
		items = append(items, x)
		metrics.Total++
		metrics.Created += x.TotalCreated
		metrics.Skipped += x.TotalSkipped
	}
	data := h.baseData(p, "Penetapan Massal")
	data["Items"] = items
	data["Metrics"] = metrics
	data["Status"] = c.Query("status")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-billing-policy-batches.create")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-billing-policy-batches/index.html", data)
}
func (h studentBillingPolicyBatchesHandlers) createForm(c *gin.Context) {
	f := billingBatchForm{PolicyID: c.Query("billing_policy_id"), ProgramID: c.Query("study_program_id"), CohortYear: c.Query("cohort_year"), CategoryID: c.Query("student_category_id"), CohortClass: strings.TrimSpace(c.Query("cohort_year_class"))}
	h.renderCreate(c, 200, f, "")
}
func (h studentBillingPolicyBatchesHandlers) generate(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := billingBatchForm{PolicyID: c.PostForm("billing_policy_id"), ProgramID: c.PostForm("study_program_id"), CohortYear: c.PostForm("cohort_year"), CategoryID: c.PostForm("student_category_id"), CohortClass: strings.TrimSpace(c.PostForm("cohort_year_class"))}
	candidates, policyAmount, err := h.candidates(c, f)
	if err != nil {
		h.renderCreate(c, 422, f, err.Error())
		return
	}
	selected := map[int64]bool{}
	for _, v := range c.PostFormArray("selected_student_ids") {
		if id := parsePositiveID(v); id > 0 {
			selected[id] = true
		}
	}
	ready := []billingBatchCandidate{}
	for _, candidate := range candidates {
		if candidate.Ready && selected[candidate.ID] {
			ready = append(ready, candidate)
		}
	}
	if parsePositiveID(f.PolicyID) == 0 || len(ready) == 0 {
		h.renderCreate(c, 422, f, "Pilih minimal satu mahasiswa yang siap ditetapkan.")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	number := newBatchNumber()
	result, err := tx.ExecContext(c.Request.Context(), `INSERT INTO student_billing_policy_batches(batch_number,billing_policy_id,study_program_id,cohort_year,student_category_id,cohort_year_class,status,total_candidates,total_created,total_skipped,total_failed,initiated_by_id,created_at) VALUES(?,?,?,?,?,?,'PROCESSING',?,0,0,0,?,UTC_TIMESTAMP(6))`, number, f.PolicyID, nullableID(f.ProgramID), nullableInt(f.CohortYear), nullableID(f.CategoryID), nullableString(f.CohortClass), len(ready), p.UserID)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	batchID, _ := result.LastInsertId()
	for _, student := range ready {
		assignment, err := tx.ExecContext(c.Request.Context(), `INSERT INTO student_billing_policies(student_id,billing_policy_id,effective_amount,assignment_type,status,assigned_at,created_at,updated_at) VALUES(?,?,?,'AUTOMATIC','ACTIVE',CURDATE(),UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))`, student.ID, f.PolicyID, policyAmount)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
		assignmentID, _ := assignment.LastInsertId()
		if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO student_billing_policy_batch_items(student_billing_policy_batch_id,student_id,student_billing_policy_id,effective_amount,status,message,created_at) VALUES(?,?,?,?, 'CREATED','Penetapan berhasil dibuat.',UTC_TIMESTAMP(6))`, batchID, student.ID, assignmentID, policyAmount); err != nil {
			c.AbortWithError(500, err)
			return
		}
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE student_billing_policy_batches SET status='COMPLETED',total_created=? WHERE id=?`, len(ready), batchID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, fmt.Sprintf("/student-billing-policy-batches/%d?status=created", batchID))
}
func (h studentBillingPolicyBatchesHandlers) details(c *gin.Context) {
	id := parsePositiveID(c.Param("id"))
	var d billingBatchDetail
	var reason sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT b.id,b.batch_number,bp.code,bp.name,b.status,b.total_candidates,b.total_created,b.total_skipped,b.total_failed,b.cancellation_reason,DATE_FORMAT(CONVERT_TZ(b.created_at,'+00:00','+07:00'),'%d %b %Y %H:%i') FROM student_billing_policy_batches b JOIN billing_policies bp ON bp.id=b.billing_policy_id WHERE b.id=?`, id).Scan(&d.ID, &d.Number, &d.PolicyCode, &d.PolicyName, &d.Status, &d.TotalCandidates, &d.TotalCreated, &d.TotalSkipped, &d.TotalFailed, &reason, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	d.StatusLabel = batchStatusLabel(d.Status)
	d.CancellationReason = reason.String
	rows, err := h.db.QueryContext(c.Request.Context(), `SELECT s.nim,s.name,i.effective_amount,i.status,COALESCE(i.message,'') FROM student_billing_policy_batch_items i JOIN students s ON s.id=i.student_id WHERE i.student_billing_policy_batch_id=? ORDER BY s.nim`, id)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []billingBatchDetailItem{}
	for rows.Next() {
		var x billingBatchDetailItem
		var amount float64
		if err = rows.Scan(&x.NIM, &x.Name, &amount, &x.Status, &x.Message); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Amount = formatRupiah(amount)
		if x.Status == "CREATED" {
			x.StatusLabel = "Dibuat"
		} else {
			x.StatusLabel = "Dilewati"
		}
		items = append(items, x)
	}
	p := auth.CurrentPrincipal(c)
	data := h.baseData(p, "Detail Penetapan Massal")
	data["Batch"] = d
	data["Items"] = items
	data["Status"] = c.Query("status")
	data["CanCancel"] = (p.HasRole("Super Admin") || p.HasPermission("student-billing-policy-batches.cancel")) && d.Status != "CANCELLED"
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-billing-policy-batches/details.html", data)
}
func (h studentBillingPolicyBatchesHandlers) cancel(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	reason := strings.TrimSpace(c.PostForm("reason"))
	if reason == "" || len([]rune(reason)) > 500 {
		c.Redirect(303, fmt.Sprintf("/student-billing-policy-batches/%d?status=cancel-invalid", id))
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	var status string
	if err = tx.QueryRowContext(c.Request.Context(), `SELECT status FROM student_billing_policy_batches WHERE id=? FOR UPDATE`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	} else if err != nil {
		c.AbortWithError(500, err)
		return
	}
	if status == "CANCELLED" {
		c.Redirect(303, fmt.Sprintf("/student-billing-policy-batches/%d?status=already-cancelled", id))
		return
	}
	var invoices int
	if err = tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices si JOIN student_billing_policy_batch_items bi ON bi.student_billing_policy_id=si.student_billing_policy_id WHERE bi.student_billing_policy_batch_id=?`, id).Scan(&invoices); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if invoices > 0 {
		c.Redirect(303, fmt.Sprintf("/student-billing-policy-batches/%d?status=has-invoices", id))
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE student_billing_policies sbp JOIN student_billing_policy_batch_items bi ON bi.student_billing_policy_id=sbp.id SET sbp.status='CANCELLED',sbp.updated_at=UTC_TIMESTAMP(6) WHERE bi.student_billing_policy_batch_id=?`, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE student_billing_policy_batches SET status='CANCELLED',cancellation_reason=?,cancelled_by_id=?,cancelled_at=UTC_TIMESTAMP(6) WHERE id=?`, reason, p.UserID, id); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,reason,performed_by_id,performed_at) VALUES('StudentBillingPolicyBatch',?,'CANCELLED',?,?,UTC_TIMESTAMP(6))`, id, reason, p.UserID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, fmt.Sprintf("/student-billing-policy-batches/%d?status=cancelled", id))
}

func (h studentBillingPolicyBatchesHandlers) candidates(c *gin.Context, f billingBatchForm) ([]billingBatchCandidate, float64, error) {
	if parsePositiveID(f.PolicyID) == 0 {
		return nil, 0, nil
	}
	var amount float64
	var active bool
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT total_amount,is_active FROM billing_policies WHERE id=?`, f.PolicyID).Scan(&amount, &active); err != nil {
		return nil, 0, fmt.Errorf("kebijakan pembayaran tidak ditemukan")
	}
	if !active {
		return nil, 0, fmt.Errorf("kebijakan pembayaran tidak aktif")
	}
	q := `SELECT s.id,s.nim,s.name,sp.code,s.cohort_year,COALESCE(sc.name,'-'),EXISTS(SELECT 1 FROM student_billing_policies x WHERE x.student_id=s.id AND x.billing_policy_id=? AND x.status<>'CANCELLED') FROM students s JOIN study_programs sp ON sp.id=s.study_program_id LEFT JOIN student_categories sc ON sc.id=s.student_category_id WHERE s.status='ACTIVE'`
	args := []any{f.PolicyID}
	if parsePositiveID(f.ProgramID) > 0 {
		q += ` AND s.study_program_id=?`
		args = append(args, f.ProgramID)
	}
	if parsePositiveID(f.CohortYear) > 0 {
		q += ` AND s.cohort_year=?`
		args = append(args, f.CohortYear)
	}
	if parsePositiveID(f.CategoryID) > 0 {
		q += ` AND s.student_category_id=?`
		args = append(args, f.CategoryID)
	}
	if f.CohortClass != "" {
		q += ` AND s.cohort_year_class=?`
		args = append(args, f.CohortClass)
	}
	q += ` ORDER BY s.nim`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []billingBatchCandidate{}
	for rows.Next() {
		var x billingBatchCandidate
		var exists bool
		if err = rows.Scan(&x.ID, &x.NIM, &x.Name, &x.Program, &x.CohortYear, &x.Category, &exists); err != nil {
			return nil, 0, err
		}
		x.Ready = !exists
		if exists {
			x.Message = "Penetapan yang sama sudah ada."
		} else {
			x.Message = "Siap ditetapkan"
		}
		out = append(out, x)
	}
	return out, amount, nil
}
func (h studentBillingPolicyBatchesHandlers) renderCreate(c *gin.Context, status int, f billingBatchForm, message string) {
	candidates, _, err := h.candidates(c, f)
	if err != nil && message == "" {
		message = err.Error()
	}
	data := h.baseData(auth.CurrentPrincipal(c), "Buat Penetapan Massal")
	data["Form"] = f
	data["Policies"] = h.loadOptions(c, `SELECT id,CONCAT(code,' - ',name) FROM billing_policies WHERE is_active=1 ORDER BY name`)
	data["Programs"] = h.loadOptions(c, `SELECT id,name FROM study_programs WHERE is_active=1 ORDER BY name`)
	data["Categories"] = h.loadOptions(c, `SELECT id,name FROM student_categories WHERE is_active=1 ORDER BY name`)
	data["Years"] = h.loadStrings(c, `SELECT DISTINCT cohort_year FROM students ORDER BY cohort_year DESC`)
	data["Candidates"] = candidates
	ready := 0
	for _, x := range candidates {
		if x.Ready {
			ready++
		}
	}
	data["ReadyCount"] = ready
	data["SkippedCount"] = len(candidates) - ready
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "student-billing-policy-batches/create.html", data)
}
func (h studentBillingPolicyBatchesHandlers) loadOptions(c *gin.Context, q string) []billingPolicyOption {
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
func (h studentBillingPolicyBatchesHandlers) loadStrings(c *gin.Context, q string) []string {
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
func (h studentBillingPolicyBatchesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "student-billing-policy-batches"}
}
func newBatchNumber() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("SBP-%d-%s", time.Now().Year(), strings.ToUpper(hex.EncodeToString(b)))
}
func batchStatusLabel(v string) string {
	if v == "CANCELLED" {
		return "Dibatalkan"
	}
	if v == "PROCESSING" {
		return "Diproses"
	}
	return "Selesai"
}
