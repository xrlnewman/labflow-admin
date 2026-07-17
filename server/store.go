package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound              = errors.New("resource not found")
	ErrInvalidTransition     = errors.New("invalid appointment status transition")
	ErrMissingIdempotencyKey = errors.New("Idempotency-Key is required")
	ErrInvalidInput          = errors.New("invalid input")
	ErrIdempotencyBusy       = errors.New("request with the same Idempotency-Key is in progress")
)

type CareStore interface {
	Dashboard(context.Context) (Dashboard, error)
	ListDepartments(context.Context) ([]Department, error)
	ListDoctors(context.Context) ([]Doctor, error)
	ListPatients(context.Context, int, int) ([]Patient, int, error)
	ListAppointments(context.Context, int, int, string) ([]Appointment, int, error)
	GetAppointment(context.Context, string) (Appointment, error)
	CreateAppointment(context.Context, Appointment) (Appointment, error)
	UpdateAppointmentStatus(context.Context, string, string, string) (Appointment, AppointmentEvent, error)
	ListAppointmentEvents(context.Context, string) ([]AppointmentEvent, error)
	ListFollowups(context.Context, int, int, string) ([]Followup, int, error)
	CreateFollowup(context.Context, Followup) (Followup, error)
	CompleteFollowup(context.Context, string) (Followup, error)
	ListSamples(context.Context, int, int, string, string) ([]Sample, int, error)
	GetSample(context.Context, string) (Sample, error)
	CreateSample(context.Context, Sample, []SampleTest) (Sample, error)
	TransitionSample(context.Context, string, string, string) (Sample, SampleEvent, error)
	SaveSampleReport(context.Context, string, SampleReport) (Sample, SampleEvent, error)
	ListSampleEvents(context.Context, string) ([]SampleEvent, error)
}

// MemoryStore is deterministic and dependency-free for unit tests and demos.
type MemoryStore struct {
	mu            sync.RWMutex
	seq           atomic.Uint64
	appointments  map[string]Appointment
	events        map[string][]AppointmentEvent
	followups     map[string]Followup
	samples       map[string]Sample
	sampleTests   map[string][]SampleTest
	sampleReports map[string]SampleReport
	sampleEvents  map[string][]SampleEvent
	departments   []Department
	doctors       []Doctor
	patients      []Patient
}

func NewMemoryStore() *MemoryStore {
	s := &MemoryStore{
		appointments: map[string]Appointment{}, events: map[string][]AppointmentEvent{}, followups: map[string]Followup{},
		samples: map[string]Sample{}, sampleTests: map[string][]SampleTest{}, sampleReports: map[string]SampleReport{}, sampleEvents: map[string][]SampleEvent{},
		departments: []Department{{ID: "line-biochem", Name: "生化检验"}, {ID: "line-micro", Name: "微生物检验"}, {ID: "line-immuno", Name: "免疫检验"}, {ID: "line-molecular", Name: "分子诊断"}},
		doctors:     []Doctor{{ID: "tech-01", Name: "林实验员", Department: "生化检验", Status: "检测中", TodayCount: 18}, {ID: "tech-02", Name: "沈实验员", Department: "微生物检验", Status: "检测中", TodayCount: 16}, {ID: "tech-03", Name: "赵实验员", Department: "免疫检验", Status: "检测中", TodayCount: 12}, {ID: "tech-04", Name: "周实验员", Department: "分子诊断", Status: "休息中", TodayCount: 10}, {ID: "tech-05", Name: "陈实验员", Department: "生化检验", Status: "检测中", TodayCount: 14}, {ID: "tech-06", Name: "王实验员", Department: "微生物检验", Status: "检测中", TodayCount: 16}},
	}
	for i := 1; i <= 30; i++ {
		s.patients = append(s.patients, Patient{ID: fmt.Sprintf("LB-%03d", i), Name: fmt.Sprintf("样本批次 A%03d", i), Phone: fmt.Sprintf("1380000%04d", i), LastVisit: "2026-07-15"})
	}
	statuses := []string{AppointmentCompleted, AppointmentServing, AppointmentWaiting, AppointmentChecked, AppointmentPending}
	for i := 1; i <= 20; i++ {
		status := statuses[(i-1)%len(statuses)]
		id := fmt.Sprintf("LB-0716-%03d", 80+i)
		s.appointments[id] = Appointment{ID: id, PatientID: fmt.Sprintf("PT-%03d", i), Patient: s.patients[i-1].Name, Department: s.departments[(i-1)%len(s.departments)].Name, Doctor: s.doctors[(i-1)%len(s.doctors)].Name, ScheduledAt: fmt.Sprintf("2026-07-16T%02d:00:00+08:00", 8+(i%10)), Status: status, CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
		if status != AppointmentPending {
			s.events[id] = append(s.events[id], AppointmentEvent{ID: id + "-EV-1", AppointmentID: id, FromStatus: AppointmentPending, ToStatus: status, Actor: "seed", CreatedAt: nowUTC()})
		}
	}
	for i := 1; i <= 12; i++ {
		id := fmt.Sprintf("FW-0716-%03d", i)
		s.followups[id] = Followup{ID: id, PatientID: fmt.Sprintf("LB-%03d", i), Patient: s.patients[i-1].Name, Summary: "质控曲线复核与结果归档", DueAt: "2026-07-17", Status: FollowupPending, CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	}
	for i := 1; i <= 12; i++ {
		id := fmt.Sprintf("SM-0717-%03d", i)
		flow := []string{SampleStatusSubmitted, SampleStatusReceived, SampleStatusTesting, SampleStatusReviewing, SampleStatusReported, SampleStatusArchived}
		statusIndex := (i - 1) % len(flow)
		s.samples[id] = Sample{ID: id, SubjectAlias: fmt.Sprintf("受检者-%03d", i), SampleType: []string{"血液", "尿液", "咽拭子"}[(i-1)%3], CollectedAt: fmt.Sprintf("2026-07-17T%02d:00:00Z", 8+(i%8)), Status: flow[statusIndex], CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
		s.sampleTests[id] = []SampleTest{{ID: id + "-T1", SampleID: id, Name: "基础检验", Status: "待检验", CreatedAt: nowUTC()}, {ID: id + "-T2", SampleID: id, Name: "专项检验", Status: "待检验", CreatedAt: nowUTC()}}
		for step := 1; step <= statusIndex; step++ {
			actions := []string{"接收样本", "开始检验", "提交报告", "复核报告", "归档报告"}
			s.sampleEvents[id] = append(s.sampleEvents[id], SampleEvent{ID: fmt.Sprintf("%s-E%d", id, step), SampleID: id, Action: actions[step-1], FromStatus: flow[step-1], ToStatus: flow[step], Actor: "seed", CreatedAt: fmt.Sprintf("2026-07-17T0%d:00:0%dZ", step, step)})
		}
	}
	s.seq.Store(1000)
	return s
}

func (s *MemoryStore) next(prefix string) string { return fmt.Sprintf("%s-%d", prefix, s.seq.Add(1)) }

func (s *MemoryStore) Dashboard(_ context.Context) (Dashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d := Dashboard{AverageWaitMinutes: 12}
	for _, a := range s.appointments {
		d.TodayAppointments++
		switch a.Status {
		case AppointmentCompleted:
			d.Completed++
		case AppointmentChecked, AppointmentWaiting, AppointmentServing:
			d.CheckedIn++
		}
	}
	for _, f := range s.followups {
		if f.Status == FollowupPending {
			d.PendingFollowups++
		}
	}
	return d, nil
}
func (s *MemoryStore) ListDepartments(_ context.Context) ([]Department, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Department(nil), s.departments...), nil
}
func (s *MemoryStore) ListDoctors(_ context.Context) ([]Doctor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Doctor(nil), s.doctors...), nil
}
func (s *MemoryStore) ListPatients(_ context.Context, page, pageSize int) ([]Patient, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return paginate(s.patients, page, pageSize)
}
func (s *MemoryStore) ListAppointments(_ context.Context, page, pageSize int, status string) ([]Appointment, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Appointment, 0, len(s.appointments))
	for _, a := range s.appointments {
		if status == "" || a.Status == status {
			all = append(all, a)
		}
	}
	return paginate(all, page, pageSize)
}
func (s *MemoryStore) GetAppointment(_ context.Context, id string) (Appointment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.appointments[id]
	if !ok {
		return Appointment{}, ErrNotFound
	}
	return a, nil
}
func (s *MemoryStore) CreateAppointment(_ context.Context, a Appointment) (Appointment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.ID == "" {
		a.ID = s.next("AP")
	}
	if a.Status == "" {
		a.Status = AppointmentPending
	}
	if a.CreatedAt == "" {
		a.CreatedAt = nowUTC()
	}
	a.UpdatedAt = a.CreatedAt
	s.appointments[a.ID] = a
	return a, nil
}
func (s *MemoryStore) UpdateAppointmentStatus(_ context.Context, id, status, actor string) (Appointment, AppointmentEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.appointments[id]
	if !ok {
		return Appointment{}, AppointmentEvent{}, ErrNotFound
	}
	if !appointmentTransitions[a.Status][status] {
		return Appointment{}, AppointmentEvent{}, ErrInvalidTransition
	}
	old := a.Status
	a.Status = status
	a.UpdatedAt = nowUTC()
	s.appointments[id] = a
	event := AppointmentEvent{ID: s.next("EV"), AppointmentID: id, FromStatus: old, ToStatus: status, Actor: actor, CreatedAt: nowUTC()}
	s.events[id] = append(s.events[id], event)
	return a, event, nil
}
func (s *MemoryStore) ListAppointmentEvents(_ context.Context, id string) ([]AppointmentEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.appointments[id]; !ok {
		return nil, ErrNotFound
	}
	return append([]AppointmentEvent(nil), s.events[id]...), nil
}
func (s *MemoryStore) ListFollowups(_ context.Context, page, pageSize int, status string) ([]Followup, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := make([]Followup, 0, len(s.followups))
	for _, f := range s.followups {
		if status == "" || f.Status == status {
			all = append(all, f)
		}
	}
	return paginate(all, page, pageSize)
}
func (s *MemoryStore) CreateFollowup(_ context.Context, f Followup) (Followup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f.ID == "" {
		f.ID = s.next("FW")
	}
	if f.Status == "" {
		f.Status = FollowupPending
	}
	if f.CreatedAt == "" {
		f.CreatedAt = nowUTC()
	}
	f.UpdatedAt = f.CreatedAt
	s.followups[f.ID] = f
	return f, nil
}
func (s *MemoryStore) CompleteFollowup(_ context.Context, id string) (Followup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.followups[id]
	if !ok {
		return Followup{}, ErrNotFound
	}
	if f.Status != FollowupPending {
		return Followup{}, ErrInvalidTransition
	}
	f.Status = FollowupCompleted
	f.UpdatedAt = nowUTC()
	s.followups[id] = f
	return f, nil
}

func (s *MemoryStore) ListSamples(_ context.Context, page, pageSize int, status, keyword string) ([]Sample, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	all := make([]Sample, 0, len(s.samples))
	for _, sample := range s.samples {
		if status != "" && sample.Status != status {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(sample.ID), keyword) && !strings.Contains(strings.ToLower(sample.SubjectAlias), keyword) && !strings.Contains(strings.ToLower(sample.SampleType), keyword) {
			continue
		}
		all = append(all, sampleSummary(sample))
	}
	sort.Slice(all, func(i, j int) bool { return all[i].UpdatedAt > all[j].UpdatedAt })
	return paginate(all, page, pageSize)
}

func (s *MemoryStore) GetSample(_ context.Context, id string) (Sample, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sample, ok := s.samples[id]
	if !ok {
		return Sample{}, ErrNotFound
	}
	sample.Tests = append([]SampleTest(nil), s.sampleTests[id]...)
	if report, ok := s.sampleReports[id]; ok {
		reportCopy := report
		sample.Report = &reportCopy
	}
	sample.Events = append([]SampleEvent(nil), s.sampleEvents[id]...)
	return sample, nil
}

func (s *MemoryStore) CreateSample(_ context.Context, sample Sample, tests []SampleTest) (Sample, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sample.ID == "" {
		sample.ID = s.next("SM")
	}
	if sample.Status == "" {
		sample.Status = SampleStatusSubmitted
	}
	if sample.CreatedAt == "" {
		sample.CreatedAt = nowUTC()
	}
	sample.UpdatedAt = sample.CreatedAt
	sample.Tests = nil
	sample.Report = nil
	sample.Events = nil
	s.samples[sample.ID] = sample
	for i := range tests {
		if tests[i].ID == "" {
			tests[i].ID = fmt.Sprintf("%s-T%d", sample.ID, i+1)
		}
		tests[i].SampleID = sample.ID
		if tests[i].Status == "" {
			tests[i].Status = "待检验"
		}
		if tests[i].CreatedAt == "" {
			tests[i].CreatedAt = sample.CreatedAt
		}
	}
	s.sampleTests[sample.ID] = append([]SampleTest(nil), tests...)
	return sample, nil
}

func (s *MemoryStore) TransitionSample(_ context.Context, id, action, actor string) (Sample, SampleEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sample, ok := s.samples[id]
	if !ok {
		return Sample{}, SampleEvent{}, ErrNotFound
	}
	next, ok := sampleActionTarget[action]
	if !ok || !sampleTransitions[sample.Status][next] {
		return Sample{}, SampleEvent{}, ErrInvalidTransition
	}
	if strings.TrimSpace(actor) == "" {
		actor = "运营人员"
	}
	old := sample.Status
	sample.Status = next
	sample.UpdatedAt = nowUTC()
	s.samples[id] = sample
	if report, ok := s.sampleReports[id]; ok && (next == SampleStatusReported || next == SampleStatusArchived) {
		report.Status = next
		report.UpdatedAt = sample.UpdatedAt
		s.sampleReports[id] = report
	}
	event := SampleEvent{ID: s.next("SE"), SampleID: id, Action: action, FromStatus: old, ToStatus: next, Actor: actor, CreatedAt: nowUTC()}
	s.sampleEvents[id] = append(s.sampleEvents[id], event)
	return sample, event, nil
}

func (s *MemoryStore) SaveSampleReport(_ context.Context, id string, report SampleReport) (Sample, SampleEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sample, ok := s.samples[id]
	if !ok {
		return Sample{}, SampleEvent{}, ErrNotFound
	}
	if !sampleTransitions[sample.Status][SampleStatusReviewing] {
		return Sample{}, SampleEvent{}, ErrInvalidTransition
	}
	now := nowUTC()
	if report.ID == "" {
		report.ID = s.next("RP")
	}
	report.SampleID = id
	report.Status = SampleStatusReviewing
	if report.CreatedAt == "" {
		report.CreatedAt = now
	}
	report.UpdatedAt = now
	s.sampleReports[id] = report
	old := sample.Status
	sample.Status = SampleStatusReviewing
	sample.UpdatedAt = now
	s.samples[id] = sample
	event := SampleEvent{ID: s.next("SE"), SampleID: id, Action: "提交报告", FromStatus: old, ToStatus: SampleStatusReviewing, Actor: "检验员", CreatedAt: now}
	s.sampleEvents[id] = append(s.sampleEvents[id], event)
	return sample, event, nil
}

func (s *MemoryStore) ListSampleEvents(_ context.Context, id string) ([]SampleEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.samples[id]; !ok {
		return nil, ErrNotFound
	}
	return append([]SampleEvent(nil), s.sampleEvents[id]...), nil
}

func sampleSummary(sample Sample) Sample {
	sample.Tests = nil
	sample.Report = nil
	sample.Events = nil
	return sample
}

func paginate[T any](all []T, page, pageSize int) ([]T, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	total := len(all)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

// SQLStore persists the same workflow in MySQL 8.4. Schema and seed live in deploy/mysql/init.sql.
type SQLStore struct{ db *sql.DB }

func NewSQLStore(ctx context.Context, dsn string) (*SQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLStore{db: db}, nil
}
func (s *SQLStore) Dashboard(ctx context.Context) (Dashboard, error) {
	var d Dashboard
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(status='已完成'),0), COALESCE(SUM(status IN ('已收样','检测排队','检测中')),0) FROM appointments`).Scan(&d.TodayAppointments, &d.Completed, &d.CheckedIn)
	if err != nil {
		return d, err
	}
	d.AverageWaitMinutes = 12
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM followups WHERE status='待完成'`).Scan(&d.PendingFollowups); err != nil {
		return d, err
	}
	return d, nil
}
func (s *SQLStore) ListDepartments(ctx context.Context) ([]Department, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name FROM departments ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Department{}
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *SQLStore) ListDoctors(ctx context.Context) ([]Doctor, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,department,status,today_count FROM doctors ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Doctor{}
	for rows.Next() {
		var d Doctor
		if err := rows.Scan(&d.ID, &d.Name, &d.Department, &d.Status, &d.TodayCount); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (s *SQLStore) ListPatients(ctx context.Context, page, pageSize int) ([]Patient, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM patients`).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,phone,last_visit FROM patients ORDER BY created_at DESC LIMIT ? OFFSET ?`, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Patient{}
	for rows.Next() {
		var p Patient
		if err := rows.Scan(&p.ID, &p.Name, &p.Phone, &p.LastVisit); err != nil {
			return nil, 0, err
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}
func (s *SQLStore) ListAppointments(ctx context.Context, page, pageSize int, status string) ([]Appointment, int, error) {
	var total int
	args := []any{}
	count := "SELECT COUNT(*) FROM appointments"
	q := "SELECT id,patient_id,patient_name,department,doctor,scheduled_at,status,created_at,updated_at FROM appointments"
	if status != "" {
		count += " WHERE status=?"
		q += " WHERE status=?"
		args = append(args, status)
	}
	if err := s.db.QueryRowContext(ctx, count, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	q += " ORDER BY scheduled_at ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Appointment{}
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(&a.ID, &a.PatientID, &a.Patient, &a.Department, &a.Doctor, &a.ScheduledAt, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
func (s *SQLStore) GetAppointment(ctx context.Context, id string) (Appointment, error) {
	var a Appointment
	err := s.db.QueryRowContext(ctx, `SELECT id,patient_id,patient_name,department,doctor,scheduled_at,status,created_at,updated_at FROM appointments WHERE id=?`, id).Scan(&a.ID, &a.PatientID, &a.Patient, &a.Department, &a.Doctor, &a.ScheduledAt, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, ErrNotFound
	}
	return a, err
}
func (s *SQLStore) CreateAppointment(ctx context.Context, a Appointment) (Appointment, error) {
	if a.ID == "" {
		a.ID = fmt.Sprintf("AP-%d", time.Now().UnixNano())
	}
	if a.Status == "" {
		a.Status = AppointmentPending
	}
	if a.CreatedAt == "" {
		a.CreatedAt = nowUTC()
	}
	a.UpdatedAt = a.CreatedAt
	_, err := s.db.ExecContext(ctx, `INSERT INTO appointments (id,patient_id,patient_name,department,doctor,scheduled_at,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`, a.ID, a.PatientID, a.Patient, a.Department, a.Doctor, a.ScheduledAt, a.Status, a.CreatedAt, a.UpdatedAt)
	return a, err
}
func (s *SQLStore) UpdateAppointmentStatus(ctx context.Context, id, status, actor string) (Appointment, AppointmentEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Appointment{}, AppointmentEvent{}, err
	}
	defer tx.Rollback()
	var a Appointment
	if err = tx.QueryRowContext(ctx, `SELECT id,patient_id,patient_name,department,doctor,scheduled_at,status,created_at,updated_at FROM appointments WHERE id=? FOR UPDATE`, id).Scan(&a.ID, &a.PatientID, &a.Patient, &a.Department, &a.Doctor, &a.ScheduledAt, &a.Status, &a.CreatedAt, &a.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return Appointment{}, AppointmentEvent{}, ErrNotFound
	} else if err != nil {
		return Appointment{}, AppointmentEvent{}, err
	}
	if !appointmentTransitions[a.Status][status] {
		return Appointment{}, AppointmentEvent{}, ErrInvalidTransition
	}
	old := a.Status
	a.Status = status
	a.UpdatedAt = nowUTC()
	if _, err = tx.ExecContext(ctx, `UPDATE appointments SET status=?,updated_at=? WHERE id=?`, status, a.UpdatedAt, id); err != nil {
		return Appointment{}, AppointmentEvent{}, err
	}
	event := AppointmentEvent{ID: fmt.Sprintf("EV-%d", time.Now().UnixNano()), AppointmentID: id, FromStatus: old, ToStatus: status, Actor: actor, CreatedAt: nowUTC()}
	if _, err = tx.ExecContext(ctx, `INSERT INTO appointment_events (id,appointment_id,from_status,to_status,actor,created_at) VALUES (?,?,?,?,?,?)`, event.ID, id, event.FromStatus, event.ToStatus, event.Actor, event.CreatedAt); err != nil {
		return Appointment{}, AppointmentEvent{}, err
	}
	if err = tx.Commit(); err != nil {
		return Appointment{}, AppointmentEvent{}, err
	}
	return a, event, nil
}
func (s *SQLStore) ListAppointmentEvents(ctx context.Context, id string) ([]AppointmentEvent, error) {
	if _, err := s.GetAppointment(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,appointment_id,from_status,to_status,actor,created_at FROM appointment_events WHERE appointment_id=? ORDER BY created_at ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AppointmentEvent{}
	for rows.Next() {
		var e AppointmentEvent
		if err := rows.Scan(&e.ID, &e.AppointmentID, &e.FromStatus, &e.ToStatus, &e.Actor, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *SQLStore) ListFollowups(ctx context.Context, page, pageSize int, status string) ([]Followup, int, error) {
	var total int
	args := []any{}
	count := "SELECT COUNT(*) FROM followups"
	q := "SELECT id,patient_id,patient_name,summary,due_at,status,created_at,updated_at FROM followups"
	if status != "" {
		count += " WHERE status=?"
		q += " WHERE status=?"
		args = append(args, status)
	}
	if err := s.db.QueryRowContext(ctx, count, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	q += " ORDER BY due_at ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Followup{}
	for rows.Next() {
		var f Followup
		if err := rows.Scan(&f.ID, &f.PatientID, &f.Patient, &f.Summary, &f.DueAt, &f.Status, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, f)
	}
	return out, total, rows.Err()
}
func (s *SQLStore) CreateFollowup(ctx context.Context, f Followup) (Followup, error) {
	if f.ID == "" {
		f.ID = fmt.Sprintf("FW-%d", time.Now().UnixNano())
	}
	if f.Status == "" {
		f.Status = FollowupPending
	}
	if f.CreatedAt == "" {
		f.CreatedAt = nowUTC()
	}
	f.UpdatedAt = f.CreatedAt
	_, err := s.db.ExecContext(ctx, `INSERT INTO followups (id,patient_id,patient_name,summary,due_at,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`, f.ID, f.PatientID, f.Patient, f.Summary, f.DueAt, f.Status, f.CreatedAt, f.UpdatedAt)
	return f, err
}
func (s *SQLStore) CompleteFollowup(ctx context.Context, id string) (Followup, error) {
	var f Followup
	err := s.db.QueryRowContext(ctx, `SELECT id,patient_id,patient_name,summary,due_at,status,created_at,updated_at FROM followups WHERE id=?`, id).Scan(&f.ID, &f.PatientID, &f.Patient, &f.Summary, &f.DueAt, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Followup{}, ErrNotFound
	}
	if err != nil {
		return Followup{}, err
	}
	if f.Status != FollowupPending {
		return Followup{}, ErrInvalidTransition
	}
	f.Status = FollowupCompleted
	f.UpdatedAt = nowUTC()
	_, err = s.db.ExecContext(ctx, `UPDATE followups SET status=?,updated_at=? WHERE id=?`, f.Status, f.UpdatedAt, id)
	return f, err
}

func (s *SQLStore) ListSamples(ctx context.Context, page, pageSize int, status, keyword string) ([]Sample, int, error) {
	conditions := []string{}
	args := []any{}
	if strings.TrimSpace(status) != "" {
		conditions = append(conditions, "status=?")
		args = append(args, strings.TrimSpace(status))
	}
	if strings.TrimSpace(keyword) != "" {
		conditions = append(conditions, "(id LIKE ? OR subject_alias LIKE ? OR sample_type LIKE ?)")
		value := "%" + strings.TrimSpace(keyword) + "%"
		args = append(args, value, value, value)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM samples"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, "SELECT id,subject_alias,sample_type,collected_at,status,created_at,updated_at FROM samples"+where+" ORDER BY updated_at DESC,id DESC LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Sample{}
	for rows.Next() {
		var sample Sample
		if err := rows.Scan(&sample.ID, &sample.SubjectAlias, &sample.SampleType, &sample.CollectedAt, &sample.Status, &sample.CreatedAt, &sample.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, sample)
	}
	return out, total, rows.Err()
}

func (s *SQLStore) GetSample(ctx context.Context, id string) (Sample, error) {
	var sample Sample
	err := s.db.QueryRowContext(ctx, `SELECT id,subject_alias,sample_type,collected_at,status,created_at,updated_at FROM samples WHERE id=?`, id).Scan(&sample.ID, &sample.SubjectAlias, &sample.SampleType, &sample.CollectedAt, &sample.Status, &sample.CreatedAt, &sample.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Sample{}, ErrNotFound
	}
	if err != nil {
		return Sample{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,sample_id,name,status,created_at FROM sample_tests WHERE sample_id=? ORDER BY id`, id)
	if err != nil {
		return Sample{}, err
	}
	for rows.Next() {
		var test SampleTest
		if err := rows.Scan(&test.ID, &test.SampleID, &test.Name, &test.Status, &test.CreatedAt); err != nil {
			rows.Close()
			return Sample{}, err
		}
		sample.Tests = append(sample.Tests, test)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Sample{}, err
	}
	rows.Close()
	var report SampleReport
	err = s.db.QueryRowContext(ctx, `SELECT id,sample_id,result,remark,status,created_at,updated_at FROM sample_reports WHERE sample_id=? ORDER BY created_at DESC LIMIT 1`, id).Scan(&report.ID, &report.SampleID, &report.Result, &report.Remark, &report.Status, &report.CreatedAt, &report.UpdatedAt)
	if err == nil {
		sample.Report = &report
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Sample{}, err
	}
	events, err := s.ListSampleEvents(ctx, id)
	if err != nil {
		return Sample{}, err
	}
	sample.Events = events
	return sample, nil
}

func (s *SQLStore) CreateSample(ctx context.Context, sample Sample, tests []SampleTest) (Sample, error) {
	if sample.ID == "" {
		sample.ID = fmt.Sprintf("SM-%d", time.Now().UnixNano())
	}
	if sample.Status == "" {
		sample.Status = SampleStatusSubmitted
	}
	if sample.CreatedAt == "" {
		sample.CreatedAt = nowUTC()
	}
	sample.UpdatedAt = sample.CreatedAt
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Sample{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO samples (id,subject_alias,sample_type,collected_at,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`, sample.ID, sample.SubjectAlias, sample.SampleType, sample.CollectedAt, sample.Status, sample.CreatedAt, sample.UpdatedAt); err != nil {
		return Sample{}, err
	}
	for i, test := range tests {
		if test.ID == "" {
			test.ID = fmt.Sprintf("%s-T%d", sample.ID, i+1)
		}
		if test.Status == "" {
			test.Status = "待检验"
		}
		if test.CreatedAt == "" {
			test.CreatedAt = sample.CreatedAt
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO sample_tests (id,sample_id,name,status,created_at) VALUES (?,?,?,?,?)`, test.ID, sample.ID, test.Name, test.Status, test.CreatedAt); err != nil {
			return Sample{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Sample{}, err
	}
	return sample, nil
}

func (s *SQLStore) TransitionSample(ctx context.Context, id, action, actor string) (Sample, SampleEvent, error) {
	next, ok := sampleActionTarget[action]
	if !ok {
		return Sample{}, SampleEvent{}, ErrInvalidTransition
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Sample{}, SampleEvent{}, err
	}
	defer tx.Rollback()
	var sample Sample
	if err = tx.QueryRowContext(ctx, `SELECT id,subject_alias,sample_type,collected_at,status,created_at,updated_at FROM samples WHERE id=? FOR UPDATE`, id).Scan(&sample.ID, &sample.SubjectAlias, &sample.SampleType, &sample.CollectedAt, &sample.Status, &sample.CreatedAt, &sample.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return Sample{}, SampleEvent{}, ErrNotFound
	} else if err != nil {
		return Sample{}, SampleEvent{}, err
	}
	if !sampleTransitions[sample.Status][next] {
		return Sample{}, SampleEvent{}, ErrInvalidTransition
	}
	if strings.TrimSpace(actor) == "" {
		actor = "运营人员"
	}
	now := nowUTC()
	old := sample.Status
	sample.Status = next
	sample.UpdatedAt = now
	if _, err := tx.ExecContext(ctx, `UPDATE samples SET status=?,updated_at=? WHERE id=?`, sample.Status, sample.UpdatedAt, id); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	if next == SampleStatusReported || next == SampleStatusArchived {
		if _, err := tx.ExecContext(ctx, `UPDATE sample_reports SET status=?,updated_at=? WHERE sample_id=?`, next, sample.UpdatedAt, id); err != nil {
			return Sample{}, SampleEvent{}, err
		}
	}
	event := SampleEvent{ID: fmt.Sprintf("SE-%d", time.Now().UnixNano()), SampleID: id, Action: action, FromStatus: old, ToStatus: next, Actor: actor, CreatedAt: now}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sample_events (id,sample_id,action,from_status,to_status,actor,created_at) VALUES (?,?,?,?,?,?,?)`, event.ID, id, event.Action, event.FromStatus, event.ToStatus, event.Actor, event.CreatedAt); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	return sample, event, nil
}

func (s *SQLStore) SaveSampleReport(ctx context.Context, id string, report SampleReport) (Sample, SampleEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Sample{}, SampleEvent{}, err
	}
	defer tx.Rollback()
	var sample Sample
	if err = tx.QueryRowContext(ctx, `SELECT id,subject_alias,sample_type,collected_at,status,created_at,updated_at FROM samples WHERE id=? FOR UPDATE`, id).Scan(&sample.ID, &sample.SubjectAlias, &sample.SampleType, &sample.CollectedAt, &sample.Status, &sample.CreatedAt, &sample.UpdatedAt); errors.Is(err, sql.ErrNoRows) {
		return Sample{}, SampleEvent{}, ErrNotFound
	} else if err != nil {
		return Sample{}, SampleEvent{}, err
	}
	if !sampleTransitions[sample.Status][SampleStatusReviewing] {
		return Sample{}, SampleEvent{}, ErrInvalidTransition
	}
	now := nowUTC()
	if report.ID == "" {
		report.ID = fmt.Sprintf("RP-%d", time.Now().UnixNano())
	}
	report.SampleID = id
	report.Status = SampleStatusReviewing
	if report.CreatedAt == "" {
		report.CreatedAt = now
	}
	report.UpdatedAt = now
	if _, err := tx.ExecContext(ctx, `INSERT INTO sample_reports (id,sample_id,result,remark,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`, report.ID, id, report.Result, report.Remark, report.Status, report.CreatedAt, report.UpdatedAt); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	old := sample.Status
	sample.Status = SampleStatusReviewing
	sample.UpdatedAt = now
	if _, err := tx.ExecContext(ctx, `UPDATE samples SET status=?,updated_at=? WHERE id=?`, sample.Status, sample.UpdatedAt, id); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	event := SampleEvent{ID: fmt.Sprintf("SE-%d", time.Now().UnixNano()), SampleID: id, Action: "提交报告", FromStatus: old, ToStatus: SampleStatusReviewing, Actor: "检验员", CreatedAt: now}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sample_events (id,sample_id,action,from_status,to_status,actor,created_at) VALUES (?,?,?,?,?,?,?)`, event.ID, id, event.Action, event.FromStatus, event.ToStatus, event.Actor, event.CreatedAt); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return Sample{}, SampleEvent{}, err
	}
	return sample, event, nil
}

func (s *SQLStore) ListSampleEvents(ctx context.Context, id string) ([]SampleEvent, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM samples WHERE id=?`, id).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, ErrNotFound
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,sample_id,action,from_status,to_status,actor,created_at FROM sample_events WHERE sample_id=? ORDER BY created_at ASC,id ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SampleEvent{}
	for rows.Next() {
		var event SampleEvent
		if err := rows.Scan(&event.ID, &event.SampleID, &event.Action, &event.FromStatus, &event.ToStatus, &event.Actor, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

// NewStoreFromEnv selects MySQL when MYSQL_DSN is configured and otherwise uses the seedable memory store.
func NewStoreFromEnv(ctx context.Context) (CareStore, func() error, error) {
	dsn := strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	if dsn == "" {
		return NewMemoryStore(), func() error { return nil }, nil
	}
	store, err := NewSQLStore(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}
	return store, store.db.Close, nil
}

// idempotencyStore is intentionally tiny: Redis is the production implementation, memory keeps tests hermetic.
type idempotencyStore interface {
	Get(context.Context, string) (string, bool, error)
	Set(context.Context, string, string, time.Duration) error
	Lock(context.Context, string, time.Duration) (func(), error)
}
type NoopIdempotency struct{}

var noopIdempotencyValues sync.Map

func (n NoopIdempotency) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := noopIdempotencyValues.Load(key)
	if !ok {
		return "", false, nil
	}
	return v.(string), true, nil
}
func (n NoopIdempotency) Set(_ context.Context, key, value string, _ time.Duration) error {
	noopIdempotencyValues.Store(key, value)
	return nil
}
func (n NoopIdempotency) Lock(_ context.Context, _ string, _ time.Duration) (func(), error) {
	return func() {}, nil
}

// memoryIdempotency is used by tests so duplicate writes return the original resource.
type memoryIdempotency struct {
	mu     sync.Mutex
	values map[string]string
}

func newMemoryIdempotency() *memoryIdempotency {
	return &memoryIdempotency{values: map[string]string{}}
}
func (m *memoryIdempotency) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[key]
	return v, ok, nil
}
func (m *memoryIdempotency) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.values[key] = value
	return nil
}
func (m *memoryIdempotency) Lock(_ context.Context, _ string, _ time.Duration) (func(), error) {
	return func() {}, nil
}

func parseInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
