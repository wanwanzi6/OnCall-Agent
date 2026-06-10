package domain

import "time"

type Alert struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Service     string            `json:"service"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Description string            `json:"description"`
	Region      string            `json:"region,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	StartsAt    time.Time         `json:"starts_at"`

	// Deprecated legacy fields kept for the stage 1-2 mock agent package.
	AlertName   string `json:"alert_name,omitempty"`
	Instance    string `json:"instance,omitempty"`
	TriggeredAt string `json:"triggered_at,omitempty"`
}

type SOP struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}

type Evidence struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Source    string            `json:"source"`
	Query     string            `json:"query,omitempty"`
	Summary   string            `json:"summary"`
	Samples   []string          `json:"samples,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`

	// Deprecated legacy field kept for the stage 1-2 mock agent package.
	Logs []string `json:"logs,omitempty"`
}

type WorkflowStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`

	// Deprecated legacy fields kept for the stage 1-2 mock agent package.
	Tool    string      `json:"tool,omitempty"`
	Output  interface{} `json:"output,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

type AIOpsAnalyzeResult struct {
	TraceID      string         `json:"trace_id"`
	Report       string         `json:"report"`
	Alerts       []Alert        `json:"alerts"`
	Steps        []WorkflowStep `json:"steps"`
	Evidence     []Evidence     `json:"evidence"`
	Citations    []Citation     `json:"citations"`
	Mode         string         `json:"mode,omitempty"`
	FallbackUsed bool           `json:"fallback_used,omitempty"`
}

type AnalyzeReport struct {
	Alert          Alert          `json:"alert"`
	SOP            SOP            `json:"sop"`
	Evidence       Evidence       `json:"evidence"`
	PossibleCause  string         `json:"possible_cause"`
	Confidence     string         `json:"confidence"`
	Recommendation []string       `json:"recommendation"`
	RiskNotice     string         `json:"risk_notice"`
	Conclusion     string         `json:"conclusion"`
	Workflow       []WorkflowStep `json:"workflow"`
	Mock           bool           `json:"mock"`
	TraceID        string         `json:"trace_id,omitempty"`
}
