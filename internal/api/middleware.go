package api

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"oncall-agent/internal/infra/trace"
)

func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(trace.HeaderName)
		if traceID == "" {
			traceID = trace.NewID()
		}
		c.Set("trace_id", traceID)
		c.Request = c.Request.WithContext(trace.WithTraceID(c.Request.Context(), traceID))
		c.Writer.Header().Set(trace.HeaderName, traceID)
		c.Next()
	}
}

func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.InfoContext(c.Request.Context(), "http request",
			"trace_id", trace.FromContext(c.Request.Context()),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}
