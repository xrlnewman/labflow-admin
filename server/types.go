package main

import "time"

// Appointment is a sample batch in the laboratory operational workflow.
type Appointment struct {
	ID          string `json:"id"`
	PatientID   string `json:"patientId,omitempty"`
	Patient     string `json:"patient"`
	Department  string `json:"department"`
	Doctor      string `json:"doctor"`
	ScheduledAt string `json:"scheduledAt"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// AppointmentEvent records every state transition for audit and queue replay.
type AppointmentEvent struct {
	ID            string `json:"id"`
	AppointmentID string `json:"appointmentId"`
	FromStatus    string `json:"fromStatus"`
	ToStatus      string `json:"toStatus"`
	Actor         string `json:"actor"`
	CreatedAt     string `json:"createdAt"`
}

// Department is a laboratory testing line.
type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Doctor is an operational laboratory technician profile.
type Doctor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Department string `json:"department"`
	Status     string `json:"status"`
	TodayCount int    `json:"todayCount"`
}

// Patient contains synthetic sample identifiers used by the demo workflow.
type Patient struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	LastVisit string `json:"lastVisit"`
}

// Followup is a laboratory quality-control task.
type Followup struct {
	ID        string `json:"id"`
	PatientID string `json:"patientId,omitempty"`
	Patient   string `json:"patient"`
	Summary   string `json:"summary"`
	DueAt     string `json:"dueAt"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateAppointmentInput is accepted by POST /appointments.
type CreateAppointmentInput struct {
	PatientID   string `json:"patientId"`
	Patient     string `json:"patient"`
	Department  string `json:"department"`
	Doctor      string `json:"doctor"`
	ScheduledAt string `json:"scheduledAt"`
}

// UpdateAppointmentStatusInput is accepted by POST /appointments/:id/status.
type UpdateAppointmentStatusInput struct {
	Status string `json:"status" binding:"required"`
	Actor  string `json:"actor"`
}

// CreateFollowupInput is accepted by POST /followups.
type CreateFollowupInput struct {
	PatientID string `json:"patientId"`
	Patient   string `json:"patient"`
	Summary   string `json:"summary"`
	DueAt     string `json:"dueAt"`
}

// Dashboard contains operational KPIs used by admin and mobile clients.
type Dashboard struct {
	TodayAppointments  int `json:"todayAppointments"`
	AverageWaitMinutes int `json:"averageWaitMinutes"`
	Completed          int `json:"completed"`
	CheckedIn          int `json:"checkedIn"`
	PendingFollowups   int `json:"pendingFollowups"`
}

// Sample is the de-identified sample registration used by the report workflow.
type Sample struct {
	ID           string        `json:"id"`
	SubjectAlias string        `json:"subjectAlias"`
	SampleType   string        `json:"sampleType"`
	CollectedAt  string        `json:"collectedAt"`
	Status       string        `json:"status"`
	CreatedAt    string        `json:"createdAt"`
	UpdatedAt    string        `json:"updatedAt"`
	Tests        []SampleTest  `json:"tests,omitempty"`
	Report       *SampleReport `json:"report,omitempty"`
	Events       []SampleEvent `json:"events,omitempty"`
}

// SampleTest is one requested laboratory test for a sample.
type SampleTest struct {
	ID        string `json:"id"`
	SampleID  string `json:"sampleId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// SampleReport stores a fictional result before it is released and archived.
type SampleReport struct {
	ID        string `json:"id"`
	SampleID  string `json:"sampleId"`
	Result    string `json:"result"`
	Remark    string `json:"remark"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SampleEvent records an auditable sample state change.
type SampleEvent struct {
	ID         string `json:"id"`
	SampleID   string `json:"sampleId"`
	Action     string `json:"action"`
	FromStatus string `json:"fromStatus"`
	ToStatus   string `json:"toStatus"`
	Actor      string `json:"actor"`
	CreatedAt  string `json:"createdAt"`
}

// CreateSampleInput is accepted by POST /samples.
type CreateSampleInput struct {
	SubjectAlias string   `json:"subjectAlias"`
	SampleType   string   `json:"sampleType"`
	CollectedAt  string   `json:"collectedAt"`
	Tests        []string `json:"tests"`
}

// SampleActorInput is accepted by sample state transition endpoints.
type SampleActorInput struct {
	Actor string `json:"actor"`
}

// CreateReportInput is accepted by POST /samples/:id/report.
type CreateReportInput struct {
	Result string `json:"result"`
	Remark string `json:"remark"`
}

const (
	SampleStatusSubmitted = "待送检"
	SampleStatusReceived  = "已接收"
	SampleStatusTesting   = "检验中"
	SampleStatusReviewing = "待复核"
	SampleStatusReported  = "已出报告"
	SampleStatusArchived  = "已归档"
)

var sampleTransitions = map[string]map[string]bool{
	SampleStatusSubmitted: {SampleStatusReceived: true},
	SampleStatusReceived:  {SampleStatusTesting: true},
	SampleStatusTesting:   {SampleStatusReviewing: true},
	SampleStatusReviewing: {SampleStatusReported: true},
	SampleStatusReported:  {SampleStatusArchived: true},
	SampleStatusArchived:  {},
}

var sampleActionTarget = map[string]string{
	"接收样本": SampleStatusReceived,
	"开始检验": SampleStatusTesting,
	"复核报告": SampleStatusReported,
	"归档报告": SampleStatusArchived,
}

const (
	AppointmentPending   = "待收样"
	AppointmentChecked   = "已收样"
	AppointmentWaiting   = "检测排队"
	AppointmentServing   = "检测中"
	AppointmentCompleted = "已完成"
	AppointmentCancelled = "已作废"
	FollowupPending      = "待完成"
	FollowupCompleted    = "已完成"
)

var appointmentTransitions = map[string]map[string]bool{
	AppointmentPending:   {AppointmentChecked: true, AppointmentCancelled: true},
	AppointmentChecked:   {AppointmentWaiting: true, AppointmentCancelled: true},
	AppointmentWaiting:   {AppointmentServing: true, AppointmentCancelled: true},
	AppointmentServing:   {AppointmentCompleted: true},
	AppointmentCompleted: {},
	AppointmentCancelled: {},
}

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339Nano) }
