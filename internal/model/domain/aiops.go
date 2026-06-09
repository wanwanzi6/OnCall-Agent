package domain

type Alert struct {
	AlertName   string `json:"alert_name"`
	Service     string `json:"service"`
	Severity    string `json:"severity"`
	Region      string `json:"region"`
	Instance    string `json:"instance"`
	TriggeredAt string `json:"triggered_at"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type SOP struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}

type Evidence struct {
	Query   string   `json:"query"`
	Summary string   `json:"summary"`
	Logs    []string `json:"logs"`
}

type WorkflowStep struct {
	Name    string      `json:"name"`
	Tool    string      `json:"tool,omitempty"`
	Status  string      `json:"status"`
	Summary string      `json:"summary"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
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
