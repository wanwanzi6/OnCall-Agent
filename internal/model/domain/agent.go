package domain

import "time"

type AgentPlan struct {
	Goal      string          `json:"goal"`
	Status    string          `json:"status"`
	Steps     []AgentPlanStep `json:"steps"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AgentPlanStep struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Tool      string            `json:"tool,omitempty"`
	Args      map[string]string `json:"args,omitempty"`
	Rationale string            `json:"rationale,omitempty"`
	Status    string            `json:"status"`
	DependsOn []string          `json:"depends_on,omitempty"`
	Error     string            `json:"error,omitempty"`
}

type AgentIteration struct {
	Index        int       `json:"index"`
	Phase        string    `json:"phase"`
	StepID       string    `json:"step_id,omitempty"`
	Tool         string    `json:"tool,omitempty"`
	Observation  string    `json:"observation"`
	ReplanReason string    `json:"replan_reason,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
}
