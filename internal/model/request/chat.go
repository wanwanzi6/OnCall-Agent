package request

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
	UserID  string `json:"user_id,omitempty"`
}
