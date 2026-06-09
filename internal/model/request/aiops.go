package request

type AnalyzeRequest struct {
	AlertName string `json:"alert_name,omitempty"`
	Service   string `json:"service,omitempty"`
}
