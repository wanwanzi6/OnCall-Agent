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

	router.POST("/knowledge/search", func(c *gin.Context) {
		var req struct {
			Query string `json:"query" binding:"required"`
			TopK  int    `json:"top_k"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query is required")
			return
		}
		results, err := knowledgeService.Search(c.Request.Context(), req.Query, req.TopK)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"results": results})
	})

	router.GET("/knowledge/documents", func(c *gin.Context) {
		docs, err := knowledgeService.ListDocuments(c.Request.Context())
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.OK(c, gin.H{"documents": docs})
	})

	router.DELETE("/knowledge/documents/:id", func(c *gin.Context) {
		if err := knowledgeService.DeleteDocument(c.Request.Context(), c.Param("id")); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.OK(c, gin.H{"deleted": true})
	})
}
