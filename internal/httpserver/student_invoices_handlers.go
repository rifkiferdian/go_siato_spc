package httpserver

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"siato-spc/internal/auth"
	"siato-spc/internal/config"
)

type studentInvoicesHandlers struct {
	cfg config.Config
	db  *sql.DB
}
type studentInvoiceItem struct {
	ID                                                                                                                                                    int64
	Number, Description, NIM, StudentName, ProgramCode, ChargeType, Period, DueDate, Total, Paid, Remaining, Status, StatusLabel, SourceType, SourceLabel string
	Sequence                                                                                                                                              int
}
type studentInvoiceMetrics struct{ Total, Unpaid, Partial, Overdue int }
type studentInvoiceForm struct {
	ID                                                                                                                                                                                     int64
	Number, StudentID, PeriodID, ChargeTypeID, AssignmentID, ScheduleID, Sequence, Description, Original, Discount, Penalty, Final, Paid, DueDate, Status, SourceType, Notes, ChangeReason string
}
type studentInvoicePrintView struct {
	Number, Description, NIM, StudentName, Program, Charge, Period, CreatedAt, DueDate string
	Original, Discount, Penalty, Final, Paid, Remaining, StatusLabel                   string
	QRCode                                                                             template.URL
}

func (h studentInvoicesHandlers) index(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	source := strings.ToUpper(strings.TrimSpace(c.Query("source_type")))
	if !validInvoiceStatus(status) {
		status = ""
	}
	if source != "AUTOMATIC" && source != "MANUAL" {
		source = ""
	}
	q := `SELECT i.id,i.invoice_number,i.description,s.nim,s.name,sp.code,ct.name,COALESCE(ap.code,'Tanpa periode'),DATE_FORMAT(i.due_date,'%d %b %Y'),i.final_amount,i.paid_amount,GREATEST(i.final_amount-i.paid_amount,0),i.status,i.source_type,COALESCE(i.sequence_no,0) FROM student_invoices i JOIN students s ON s.id=i.student_id JOIN study_programs sp ON sp.id=s.study_program_id JOIN charge_types ct ON ct.id=i.charge_type_id LEFT JOIN academic_periods ap ON ap.id=i.academic_period_id WHERE 1=1`
	args := []any{}
	if search != "" {
		q += ` AND (i.invoice_number LIKE ? OR s.nim LIKE ? OR s.name LIKE ? OR i.description LIKE ? OR ct.name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like, like, like)
	}
	if status != "" {
		q += ` AND i.status=?`
		args = append(args, status)
	}
	if source != "" {
		q += ` AND i.source_type=?`
		args = append(args, source)
	}
	q += ` ORDER BY i.created_at DESC,i.due_date`
	rows, err := h.db.QueryContext(c.Request.Context(), q, args...)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer rows.Close()
	items := []studentInvoiceItem{}
	for rows.Next() {
		var x studentInvoiceItem
		var total, paid, remaining float64
		if err = rows.Scan(&x.ID, &x.Number, &x.Description, &x.NIM, &x.StudentName, &x.ProgramCode, &x.ChargeType, &x.Period, &x.DueDate, &total, &paid, &remaining, &x.Status, &x.SourceType, &x.Sequence); err != nil {
			c.AbortWithError(500, err)
			return
		}
		x.Total = formatRupiah(total)
		x.Paid = formatRupiah(paid)
		x.Remaining = formatRupiah(remaining)
		x.StatusLabel = invoiceStatusLabel(x.Status)
		if x.SourceType == "AUTOMATIC" {
			x.SourceLabel = "Otomatis"
		} else {
			x.SourceLabel = "Manual"
		}
		items = append(items, x)
	}
	var metrics studentInvoiceMetrics
	if err = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*),COALESCE(SUM(status='UNPAID'),0),COALESCE(SUM(status='PARTIALLY_PAID'),0),COALESCE(SUM(status='OVERDUE'),0) FROM student_invoices`).Scan(&metrics.Total, &metrics.Unpaid, &metrics.Partial, &metrics.Overdue); err != nil {
		c.AbortWithError(500, err)
		return
	}
	data := h.baseData(p, "Tagihan Mahasiswa")
	data["Items"] = items
	data["Metrics"] = metrics
	data["Search"] = search
	data["SelectedStatus"] = status
	data["SelectedSource"] = source
	data["Status"] = c.Query("result")
	data["CanCreate"] = p.HasRole("Super Admin") || p.HasPermission("student-invoices.create")
	data["CanUpdate"] = p.HasRole("Super Admin") || p.HasPermission("student-invoices.update")
	data["CanDelete"] = p.HasRole("Super Admin") || p.HasPermission("student-invoices.delete")
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-invoices/index.html", data)
}
func (h studentInvoicesHandlers) createForm(c *gin.Context) {
	h.renderForm(c, 200, "Tambah Tagihan Mahasiswa", studentInvoiceForm{Number: h.nextNumber(c), StudentID: c.Query("student_id"), SourceType: "MANUAL", DueDate: time.Now().Format("2006-01-02"), Original: "0", Discount: "0", Penalty: "0", Final: "0", Paid: "0"}, "")
}
func (h studentInvoicesHandlers) create(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	f := invoiceFormFromRequest(c)
	f.Paid = "0"
	f.Final = invoiceFinal(f)
	f.Status = determineInvoiceStatus("", f.Final, f.Paid, f.DueDate)
	if msg := h.validate(c, f, 0); msg != "" {
		h.renderForm(c, 422, "Tambah Tagihan Mahasiswa", f, msg)
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(c.Request.Context(), `INSERT INTO student_invoices(invoice_number,student_id,academic_period_id,charge_type_id,student_billing_policy_id,billing_policy_schedule_id,sequence_no,description,original_amount,discount_amount,penalty_amount,final_amount,paid_amount,due_date,status,source_type,notes,created_at,updated_at,version) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,0,?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6),1)`, f.Number, f.StudentID, nullableID(f.PeriodID), f.ChargeTypeID, automaticValue(f.SourceType, f.AssignmentID), automaticValue(f.SourceType, f.ScheduleID), automaticValue(f.SourceType, f.Sequence), f.Description, f.Original, f.Discount, f.Penalty, f.Final, f.DueDate, f.Status, f.SourceType, nullableString(f.Notes))
	if err != nil {
		h.renderForm(c, 422, "Tambah Tagihan Mahasiswa", f, err.Error())
		return
	}
	id, _ := r.LastInsertId()
	newJSON, _ := json.Marshal(map[string]any{"invoice_number": f.Number, "student_id": f.StudentID, "final_amount": f.Final, "status": f.Status})
	if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,new_values_json,performed_by_id,performed_at) VALUES('StudentInvoice',?,'CREATED',?,?,UTC_TIMESTAMP(6))`, id, string(newJSON), p.UserID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, "/student-invoices?result=created")
}
func (h studentInvoicesHandlers) editForm(c *gin.Context) {
	f, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	h.renderForm(c, 200, "Ubah Tagihan Mahasiswa", f, "")
}
func (h studentInvoicesHandlers) update(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := parsePositiveID(c.Param("id"))
	old, err := h.find(c, id)
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	f := invoiceFormFromRequest(c)
	f.ID = id
	f.Number = old.Number
	f.StudentID = old.StudentID
	f.SourceType = old.SourceType
	f.AssignmentID = old.AssignmentID
	f.ScheduleID = old.ScheduleID
	f.Sequence = old.Sequence
	f.Paid = old.Paid
	var locked int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT (EXISTS(SELECT 1 FROM payment_allocations pa JOIN payments p ON p.id=pa.payment_id WHERE pa.student_invoice_id=? AND p.status='VERIFIED') OR EXISTS(SELECT 1 FROM student_invoice_adjustments WHERE student_invoice_id=?))`, id, id).Scan(&locked)
	paid, _ := strconv.ParseFloat(old.Paid, 64)
	if paid > 0 {
		f.Original = old.Original
	}
	if locked == 1 {
		f.Original = old.Original
		f.Discount = old.Discount
		f.Penalty = old.Penalty
		f.DueDate = old.DueDate
		f.PeriodID = old.PeriodID
		f.ChargeTypeID = old.ChargeTypeID
	}
	f.Final = invoiceFinal(f)
	f.Status = determineInvoiceStatus(old.Status, f.Final, f.Paid, f.DueDate)
	f.ChangeReason = strings.TrimSpace(c.PostForm("change_reason"))
	if f.ChangeReason == "" {
		h.renderForm(c, 422, "Ubah Tagihan Mahasiswa", f, "Alasan perubahan tagihan wajib diisi.")
		return
	}
	if msg := h.validate(c, f, id); msg != "" {
		h.renderForm(c, 422, "Ubah Tagihan Mahasiswa", f, msg)
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(c.Request.Context(), `UPDATE student_invoices SET academic_period_id=?,charge_type_id=?,description=?,original_amount=?,discount_amount=?,penalty_amount=?,final_amount=?,due_date=?,status=?,notes=?,updated_at=UTC_TIMESTAMP(6),version=version+1 WHERE id=?`, nullableID(f.PeriodID), f.ChargeTypeID, f.Description, f.Original, f.Discount, f.Penalty, f.Final, f.DueDate, f.Status, nullableString(f.Notes), id)
	if err != nil {
		h.renderForm(c, 422, "Ubah Tagihan Mahasiswa", f, err.Error())
		return
	}
	oldJSON, _ := json.Marshal(map[string]any{"original": old.Original, "discount": old.Discount, "penalty": old.Penalty, "final": old.Final, "due_date": old.DueDate, "status": old.Status})
	newJSON, _ := json.Marshal(map[string]any{"original": f.Original, "discount": f.Discount, "penalty": f.Penalty, "final": f.Final, "due_date": f.DueDate, "status": f.Status})
	if _, err = tx.ExecContext(c.Request.Context(), `INSERT INTO audit_logs(entity_type,entity_id,action,old_values_json,new_values_json,reason,performed_by_id,performed_at) VALUES('StudentInvoice',?,'UPDATED',?,?,?,?,UTC_TIMESTAMP(6))`, id, string(oldJSON), string(newJSON), f.ChangeReason, p.UserID); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, "/student-invoices?result=updated")
}
func (h studentInvoicesHandlers) delete(c *gin.Context) {
	p := auth.CurrentPrincipal(c)
	if !validSessionCSRF(p, c.PostForm("csrf_token")) {
		c.AbortWithStatus(403)
		return
	}
	id := c.Param("id")
	var refs int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM payment_allocations WHERE student_invoice_id=?)+(SELECT COUNT(*) FROM student_invoice_adjustments WHERE student_invoice_id=?)`, id, id).Scan(&refs); err != nil {
		c.AbortWithError(500, err)
		return
	}
	if refs > 0 {
		c.Redirect(303, "/student-invoices?result=in-use")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(c.Request.Context(), `DELETE FROM audit_logs WHERE entity_type='StudentInvoice' AND entity_id=?`, id); err == nil {
		_, err = tx.ExecContext(c.Request.Context(), `DELETE FROM student_invoices WHERE id=?`, id)
	}
	if err != nil {
		c.Redirect(303, "/student-invoices?result=delete-failed")
		return
	}
	if err = tx.Commit(); err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.Redirect(303, "/student-invoices?result=deleted")
}

func (h studentInvoicesHandlers) print(c *gin.Context) {
	f, err := h.find(c, c.Param("id"))
	if errors.Is(err, sql.ErrNoRows) {
		c.AbortWithStatus(404)
		return
	}
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	var view studentInvoicePrintView
	var createdAt time.Time
	var original, discount, penalty, final, paid float64
	err = h.db.QueryRowContext(c.Request.Context(), `SELECT i.invoice_number,i.description,s.nim,s.name,sp.name,ct.name,COALESCE(CONCAT(ap.academic_year,' / ',ap.semester),'Tanpa periode'),i.created_at,i.original_amount,i.discount_amount,i.penalty_amount,i.final_amount,i.paid_amount FROM student_invoices i JOIN students s ON s.id=i.student_id JOIN study_programs sp ON sp.id=s.study_program_id JOIN charge_types ct ON ct.id=i.charge_type_id LEFT JOIN academic_periods ap ON ap.id=i.academic_period_id WHERE i.id=?`, f.ID).Scan(&view.Number, &view.Description, &view.NIM, &view.StudentName, &view.Program, &view.Charge, &view.Period, &createdAt, &original, &discount, &penalty, &final, &paid)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	dueDate, _ := time.Parse("2006-01-02", f.DueDate)
	view.CreatedAt = formatFullIndonesianDate(createdAt.In(jakarta)) + " " + createdAt.In(jakarta).Format("15:04") + " WIB"
	view.DueDate = formatFullIndonesianDate(dueDate)
	view.Original = formatRupiah(original)
	view.Discount = formatRupiah(discount)
	view.Penalty = formatRupiah(penalty)
	view.Final = formatRupiah(final)
	view.Paid = formatRupiah(paid)
	view.Remaining = formatRupiah(max(final-paid, 0))
	view.StatusLabel = invoiceStatusLabel(f.Status)
	if png, qrErr := qrcode.Encode("SIATO-SPC|TAGIHAN|"+view.Number, qrcode.Medium, 180); qrErr == nil {
		view.QRCode = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
	}
	printedAt := time.Now().In(jakarta)
	c.Header("Cache-Control", "no-store")
	c.HTML(200, "student-invoices/print.html", gin.H{"Title": "Dokumen Tagihan", "AppName": h.cfg.Name, "Year": printedAt.Year(), "Invoice": view, "PrintedAt": formatFullIndonesianDate(printedAt) + " " + printedAt.Format("15:04") + " WIB"})
}

func formatFullIndonesianDate(value time.Time) string {
	months := [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%02d %s %d", value.Day(), months[value.Month()-1], value.Year())
}
func (h studentInvoicesHandlers) find(c *gin.Context, id any) (studentInvoiceForm, error) {
	var f studentInvoiceForm
	var period, assignment, schedule, sequence sql.NullInt64
	var due time.Time
	var original, discount, penalty, final, paid float64
	var notes sql.NullString
	err := h.db.QueryRowContext(c.Request.Context(), `SELECT id,invoice_number,student_id,academic_period_id,charge_type_id,student_billing_policy_id,billing_policy_schedule_id,sequence_no,description,original_amount,discount_amount,penalty_amount,final_amount,paid_amount,due_date,status,source_type,notes FROM student_invoices WHERE id=?`, id).Scan(&f.ID, &f.Number, &f.StudentID, &period, &f.ChargeTypeID, &assignment, &schedule, &sequence, &f.Description, &original, &discount, &penalty, &final, &paid, &due, &f.Status, &f.SourceType, &notes)
	if period.Valid {
		f.PeriodID = strconv.FormatInt(period.Int64, 10)
	}
	if assignment.Valid {
		f.AssignmentID = strconv.FormatInt(assignment.Int64, 10)
	}
	if schedule.Valid {
		f.ScheduleID = strconv.FormatInt(schedule.Int64, 10)
	}
	if sequence.Valid {
		f.Sequence = strconv.FormatInt(sequence.Int64, 10)
	}
	f.Original = strconv.FormatFloat(original, 'f', 2, 64)
	f.Discount = strconv.FormatFloat(discount, 'f', 2, 64)
	f.Penalty = strconv.FormatFloat(penalty, 'f', 2, 64)
	f.Final = strconv.FormatFloat(final, 'f', 2, 64)
	f.Paid = strconv.FormatFloat(paid, 'f', 2, 64)
	f.DueDate = due.Format("2006-01-02")
	f.Notes = notes.String
	return f, err
}
func invoiceFormFromRequest(c *gin.Context) studentInvoiceForm {
	return studentInvoiceForm{Number: strings.ToUpper(strings.TrimSpace(c.PostForm("invoice_number"))), StudentID: c.PostForm("student_id"), PeriodID: c.PostForm("academic_period_id"), ChargeTypeID: c.PostForm("charge_type_id"), AssignmentID: c.PostForm("student_billing_policy_id"), ScheduleID: c.PostForm("billing_policy_schedule_id"), Sequence: c.PostForm("sequence_no"), Description: strings.TrimSpace(c.PostForm("description")), Original: c.PostForm("original_amount"), Discount: c.PostForm("discount_amount"), Penalty: c.PostForm("penalty_amount"), Paid: c.PostForm("paid_amount"), DueDate: c.PostForm("due_date"), Status: c.PostForm("status"), SourceType: strings.ToUpper(strings.TrimSpace(c.PostForm("source_type"))), Notes: strings.TrimSpace(c.PostForm("notes"))}
}
func invoiceFinal(f studentInvoiceForm) string {
	o, _ := strconv.ParseFloat(f.Original, 64)
	d, _ := strconv.ParseFloat(f.Discount, 64)
	p, _ := strconv.ParseFloat(f.Penalty, 64)
	return strconv.FormatFloat(o-d+p, 'f', 2, 64)
}
func (h studentInvoicesHandlers) validate(c *gin.Context, f studentInvoiceForm, id int64) string {
	o, e1 := strconv.ParseFloat(f.Original, 64)
	d, e2 := strconv.ParseFloat(f.Discount, 64)
	p, e3 := strconv.ParseFloat(f.Penalty, 64)
	final, e4 := strconv.ParseFloat(f.Final, 64)
	paid, e5 := strconv.ParseFloat(f.Paid, 64)
	_, e6 := time.Parse("2006-01-02", f.DueDate)
	switch {
	case f.Number == "" || utf8.RuneCountInString(f.Number) > 50:
		return "Nomor tagihan wajib diisi dan maksimal 50 karakter."
	case parsePositiveID(f.StudentID) == 0:
		return "Mahasiswa wajib dipilih."
	case parsePositiveID(f.ChargeTypeID) == 0:
		return "Jenis biaya wajib dipilih."
	case f.PeriodID != "" && parsePositiveID(f.PeriodID) == 0:
		return "Periode akademik tidak valid."
	case f.Description == "" || utf8.RuneCountInString(f.Description) > 255:
		return "Keterangan wajib diisi dan maksimal 255 karakter."
	case e1 != nil || e2 != nil || e3 != nil || o < 0 || d < 0 || p < 0:
		return "Nominal, diskon, dan denda harus berupa angka nol atau lebih."
	case e4 != nil || final < 0:
		return "Diskon tidak boleh melebihi nominal awal ditambah denda."
	case e5 != nil || paid < 0 || paid > final:
		return "Nominal dibayar tidak valid atau melebihi total akhir."
	case e6 != nil:
		return "Tanggal jatuh tempo tidak valid."
	case f.SourceType != "MANUAL" && f.SourceType != "AUTOMATIC":
		return "Sumber tagihan tidak valid."
	case utf8.RuneCountInString(f.Notes) > 500:
		return "Catatan maksimal 500 karakter."
	}
	var count int
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE invoice_number=? AND id<>?`, f.Number, id).Scan(&count); err != nil || count > 0 {
		return "Nomor tagihan sudah digunakan."
	}
	if err := h.db.QueryRowContext(c.Request.Context(), `SELECT (SELECT COUNT(*) FROM students WHERE id=?)+(SELECT COUNT(*) FROM charge_types WHERE id=?)`, f.StudentID, f.ChargeTypeID).Scan(&count); err != nil || count != 2 {
		return "Mahasiswa atau jenis biaya tidak ditemukan."
	}
	if f.PeriodID != "" {
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM academic_periods WHERE id=?`, f.PeriodID).Scan(&count)
		if count == 0 {
			return "Periode akademik tidak ditemukan."
		}
	}
	if f.SourceType == "AUTOMATIC" {
		if parsePositiveID(f.AssignmentID) == 0 || parsePositiveID(f.ScheduleID) == 0 {
			return "Tagihan otomatis wajib memiliki penetapan dan jadwal kebijakan."
		}
		sequence := parsePositiveID(f.Sequence)
		var studentID, policyID int64
		if err := h.db.QueryRowContext(c.Request.Context(), `SELECT student_id,billing_policy_id FROM student_billing_policies WHERE id=?`, f.AssignmentID).Scan(&studentID, &policyID); err != nil || strconv.FormatInt(studentID, 10) != f.StudentID {
			return "Penetapan kebijakan tidak sesuai dengan mahasiswa."
		}
		var schedulePolicy int64
		var scheduleSequence int
		if err := h.db.QueryRowContext(c.Request.Context(), `SELECT billing_policy_id,sequence_no FROM billing_policy_schedules WHERE id=?`, f.ScheduleID).Scan(&schedulePolicy, &scheduleSequence); err != nil || schedulePolicy != policyID {
			return "Jadwal termin tidak sesuai dengan kebijakan."
		}
		if int64(scheduleSequence) != sequence {
			return "Urutan termin harus sama dengan jadwal yang dipilih."
		}
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE student_id=? AND billing_policy_schedule_id=? AND id<>?`, f.StudentID, f.ScheduleID, id).Scan(&count)
		if count > 0 {
			return "Termin tersebut sudah menjadi tagihan mahasiswa ini."
		}
	}
	return ""
}
func (h studentInvoicesHandlers) nextNumber(c *gin.Context) string {
	prefix := fmt.Sprintf("INV-%d-", time.Now().Year())
	var n int
	_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE invoice_number LIKE ?`, prefix+"%").Scan(&n)
	for {
		n++
		candidate := fmt.Sprintf("%s%04d", prefix, n)
		var exists int
		_ = h.db.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM student_invoices WHERE invoice_number=?`, candidate).Scan(&exists)
		if exists == 0 {
			return candidate
		}
	}
}
func (h studentInvoicesHandlers) renderForm(c *gin.Context, status int, title string, f studentInvoiceForm, message string) {
	data := h.baseData(auth.CurrentPrincipal(c), title)
	data["Form"] = f
	data["Students"] = h.options(c, `SELECT id,CONCAT(nim,' - ',name) FROM students ORDER BY name`)
	data["Periods"] = h.options(c, `SELECT id,CONCAT(code,' - ',academic_year) FROM academic_periods ORDER BY start_date DESC`)
	data["ChargeTypes"] = h.options(c, `SELECT id,CONCAT(code,' - ',name) FROM charge_types ORDER BY name`)
	data["Assignments"] = h.options(c, `SELECT sbp.id,CONCAT(s.nim,' - ',s.name,' | ',bp.code) FROM student_billing_policies sbp JOIN students s ON s.id=sbp.student_id JOIN billing_policies bp ON bp.id=sbp.billing_policy_id WHERE sbp.status='ACTIVE' ORDER BY s.name`)
	data["Schedules"] = h.options(c, `SELECT bps.id,CONCAT(bp.code,' | Termin ',bps.sequence_no,' - ',bps.description) FROM billing_policy_schedules bps JOIN billing_policies bp ON bp.id=bps.billing_policy_id ORDER BY bps.due_date`)
	data["IsEdit"] = f.ID > 0
	data["Error"] = message
	c.Header("Cache-Control", "no-store")
	c.HTML(status, "student-invoices/form.html", data)
}
func (h studentInvoicesHandlers) options(c *gin.Context, q string) []billingPolicyOption {
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
func (h studentInvoicesHandlers) baseData(p *auth.Principal, title string) gin.H {
	role := strings.Join(p.Roles, ", ")
	if role == "" {
		role = "Pengguna"
	}
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	return gin.H{"Title": title, "AppName": h.cfg.Name, "Year": time.Now().Year(), "User": p, "RoleLabel": role, "Initial": firstInitial(p.UserName), "CurrentDate": formatIndonesianDate(time.Now().In(jakarta)), "NavGroups": buildAdminNavigation(p), "ActiveMenu": "student-invoices"}
}
func determineInvoiceStatus(current, finalRaw, paidRaw, dueRaw string) string {
	if current == "CANCELLED" {
		return current
	}
	final, _ := strconv.ParseFloat(finalRaw, 64)
	paid, _ := strconv.ParseFloat(paidRaw, 64)
	due, _ := time.Parse("2006-01-02", dueRaw)
	if final == 0 {
		return "WAIVED"
	}
	if paid >= final {
		return "PAID"
	}
	if paid > 0 {
		return "PARTIALLY_PAID"
	}
	if due.Before(time.Now().Truncate(24 * time.Hour)) {
		return "OVERDUE"
	}
	return "UNPAID"
}
func validInvoiceStatus(v string) bool {
	return v == "UNPAID" || v == "PARTIALLY_PAID" || v == "PAID" || v == "OVERDUE" || v == "CANCELLED" || v == "WAIVED"
}
func automaticValue(source, v string) any {
	if source != "AUTOMATIC" || v == "" {
		return nil
	}
	return v
}
