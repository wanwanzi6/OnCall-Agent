package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK         = 0
	CodeBadRequest = 400
	CodeInternal   = 500
)

type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
		TraceID: traceID(c),
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Code:    CodeBadRequest,
		Message: message,
		TraceID: traceID(c),
	})
}

func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, APIResponse{
		Code:    CodeInternal,
		Message: message,
		TraceID: traceID(c),
	})
}

func traceID(c *gin.Context) string {
	if value, ok := c.Get("trace_id"); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}
