package knowledge

import (
	"github.com/gin-gonic/gin"

	"oncall-agent/internal/model/response"
	"oncall-agent/internal/service"
)

func Register(router *gin.RouterGroup, knowledgeService *service.KnowledgeService) {
	router.POST("/knowledge/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err == nil {
			response.OK(c, knowledgeService.Upload(file.Filename, file.Size))
			return
		}

		var req struct {
			FileName string `json:"file_name"`
			Size     int64  `json:"size"`
		}
		if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
			response.BadRequest(c, "file multipart field or JSON file_name is required")
			return
		}

		response.OK(c, knowledgeService.Upload(req.FileName, req.Size))
	})
}
