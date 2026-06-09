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
			result, uploadErr := knowledgeService.SaveUpload(c.Request.Context(), file)
			if uploadErr != nil {
				response.BadRequest(c, uploadErr.Error())
				return
			}
			response.OK(c, result)
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

		result, uploadErr := knowledgeService.UploadMetadata(c.Request.Context(), req.FileName, req.Size)
		if uploadErr != nil {
			response.BadRequest(c, uploadErr.Error())
			return
		}
		response.OK(c, result)
	})
}
