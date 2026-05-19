// File: internal/delivery/http/feedback_handler.go
package http

import (
	"net/http"
	"strconv"

	"github.com/azharf99/algo-feedback/internal/domain"
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/azharf99/algo-feedback/pkg/i18n"
	"github.com/gin-gonic/gin"
)

type FeedbackHandler struct {
	usecase domain.FeedbackUsecase
}

func (h *FeedbackHandler) getLang(c *gin.Context) string {
	return ctxutil.GetLanguage(c.Request.Context())
}

func NewFeedbackHandler(r *gin.RouterGroup, us domain.FeedbackUsecase) {
	handler := &FeedbackHandler{usecase: us}

	feedbackRoutes := r.Group("/feedbacks")
	{
		// Endpoints canggih!
		feedbackRoutes.POST("/seeder", handler.RunSeeder)
		feedbackRoutes.POST("/generate-pdf", handler.GeneratePDF)
		feedbackRoutes.POST("/generate-graduation-pdf", handler.GenerateGraduationPDF)
		feedbackRoutes.POST("/generate-all-pdf", handler.GenerateAllPendingPDF)
		feedbackRoutes.POST("/send-wa", handler.SendWhatsApp)

		feedbackRoutes.GET("", handler.GetAll)
		feedbackRoutes.GET("/summary", handler.GetWeeklySummary)
		feedbackRoutes.GET("/:id", handler.GetByID)
		feedbackRoutes.GET("/:id/download", handler.DownloadPDF)
		feedbackRoutes.POST("", handler.Create)
		feedbackRoutes.PUT("/:id", handler.Update)
		feedbackRoutes.DELETE("/:id", handler.Delete)
		feedbackRoutes.DELETE("/bulk", handler.BulkDelete)

		// Graduation Feedbacks
		feedbackRoutes.GET("/graduation", handler.GetGraduationFeedbacks)
		feedbackRoutes.GET("/graduation/:id/download", handler.DownloadGraduationPDF)
		feedbackRoutes.PUT("/graduation/:id", handler.UpdateGraduationFeedback)
		feedbackRoutes.DELETE("/graduation/:id", handler.DeleteGraduationFeedback)
	}
}

func (h *FeedbackHandler) BulkDelete(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	lang := h.getLang(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	if err := h.usecase.BulkDelete(c.Request.Context(), req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

func (h *FeedbackHandler) GetWeeklySummary(c *gin.Context) {
	result, err := h.usecase.GetWeeklySummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// RunSeeder: POST /feedbacks/seeder?group_id=1&all=true
func (h *FeedbackHandler) RunSeeder(c *gin.Context) {
	allStr := c.Query("all")
	all := allStr == "true"
	lang := h.getLang(c)

	var groupIDPtr *uint
	if gIDStr := c.Query("group_id"); gIDStr != "" {
		if id, err := strconv.Atoi(gIDStr); err == nil {
			parsedID := uint(id)
			groupIDPtr = &parsedID
		}
	}

	result, err := h.usecase.GenerateFeedback(c.Request.Context(), groupIDPtr, all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_seeder_success"),
		"data":    result,
	})
}

// GeneratePDF: POST /feedbacks/generate-pdf
func (h *FeedbackHandler) GeneratePDF(c *gin.Context) {
	lang := h.getLang(c)
	// Parsing Request Body JSON
	var req struct {
		StudentID *uint   `json:"student_id"`
		Course    *string `json:"course"`
		Number    *uint   `json:"number"`
		All       bool    `json:"all"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	result, err := h.usecase.GeneratePDFAsync(c.Request.Context(), req.StudentID, req.Course, req.Number, req.All)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_pdf_task_failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_pdf_background"),
		"tasks":   result,
	})
}

// GenerateAllPendingPDF: POST /feedbacks/generate-all-pdf
func (h *FeedbackHandler) GenerateAllPendingPDF(c *gin.Context) {
	lang := h.getLang(c)
	result, err := h.usecase.GeneratePendingPDFs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "error_pdf_task_failed")})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_pdf_background"),
		"tasks":   result,
	})
}

// SendWhatsApp: POST /feedbacks/send-wa?student_id=1xxxxxxx
func (h *FeedbackHandler) SendWhatsApp(c *gin.Context) {
	lang := h.getLang(c)
	var studentIDPtr *uint
	if fIDStr := c.Query("student_id"); fIDStr != "" {
		if id, err := strconv.Atoi(fIDStr); err == nil {
			parsedID := uint(id)
			studentIDPtr = &parsedID
		}
	}

	result, err := h.usecase.SendFeedbackPDF(c.Request.Context(), studentIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_wa_scheduled"),
		"data":    result,
	})
}

// GetAll: GET /feedbacks
// Mendukung pagination opsional via query params: ?page=1&limit=10
func (h *FeedbackHandler) GetAll(c *gin.Context) {
	lang := h.getLang(c)
	if c.Query("page") != "" || c.Query("limit") != "" {
		var params domain.PaginationParams
		if err := c.ShouldBindQuery(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
			return
		}
		result, stats, err := h.usecase.GetPaginated(c.Request.Context(), params)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"data":  result.Data,
			"meta":  result,
			"stats": stats,
		})
		return
	}

	result, stats, err := h.usecase.GetPaginated(c.Request.Context(), domain.PaginationParams{Page: 1, Limit: 100})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  result.Data,
		"meta":  result,
		"stats": stats,
	})
}

// GetByID: GET /feedbacks/:id
func (h *FeedbackHandler) GetByID(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	feedback, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_feedback_not_found")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": feedback})
}

// DownloadPDF: GET /feedbacks/:id/download
func (h *FeedbackHandler) DownloadPDF(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	feedback, err := h.usecase.GetByID(c.Request.Context(), uint(id))
	if err != nil || feedback.URLPDF == nil || *feedback.URLPDF == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}

	// Set headers to prevent caching
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// File served from local storage
	c.File(*feedback.URLPDF)
}

// Create: POST /feedbacks
func (h *FeedbackHandler) Create(c *gin.Context) {
	lang := h.getLang(c)
	var req domain.Feedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Create(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": i18n.T(lang, "msg_save_failed")})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": i18n.T(lang, "msg_save_success"), "data": req})
}

// Update: PUT /feedbacks/:id
func (h *FeedbackHandler) Update(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var req domain.Feedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

// Delete: DELETE /feedbacks/:id
func (h *FeedbackHandler) Delete(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	if err := h.usecase.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}

// GenerateGraduationPDF: POST /feedbacks/generate-graduation-pdf
func (h *FeedbackHandler) GenerateGraduationPDF(c *gin.Context) {
	lang := h.getLang(c)
	var req struct {
		StudentID *uint   `json:"student_id" binding:"required"`
		Course    *string `json:"course" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}

	result, err := h.usecase.GenerateGraduationPDFAsync(c.Request.Context(), req.StudentID, req.Course)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": i18n.T(lang, "msg_pdf_background"),
		"tasks":   result,
	})
}

// GetGraduationFeedbacks: GET /feedbacks/graduation
func (h *FeedbackHandler) GetGraduationFeedbacks(c *gin.Context) {
	lang := h.getLang(c)
	var params domain.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_data")})
		return
	}
	result, err := h.usecase.GetPaginatedGraduationFeedbacks(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": result.Data,
		"meta": result,
	})
}

// DownloadGraduationPDF: GET /feedbacks/graduation/:id/download
func (h *FeedbackHandler) DownloadGraduationPDF(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	gf, err := h.usecase.GetGraduationFeedbackByID(c.Request.Context(), uint(id))
	if err != nil || gf.URLPDF == nil || *gf.URLPDF == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": i18n.T(lang, "error_file_not_found")})
		return
	}

	c.File(*gf.URLPDF)
}

// UpdateGraduationFeedback: PUT /feedbacks/graduation/:id
func (h *FeedbackHandler) UpdateGraduationFeedback(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	var req domain.GraduationFeedback
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.UpdateGraduationFeedback(c.Request.Context(), uint(id), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_update_success")})
}

// DeleteGraduationFeedback: DELETE /feedbacks/graduation/:id
func (h *FeedbackHandler) DeleteGraduationFeedback(c *gin.Context) {
	lang := h.getLang(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": i18n.T(lang, "error_invalid_id")})
		return
	}

	if err := h.usecase.DeleteGraduationFeedback(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": i18n.T(lang, "msg_delete_success")})
}
