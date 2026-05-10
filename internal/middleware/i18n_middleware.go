package middleware

import (
	"github.com/azharf99/algo-feedback/pkg/ctxutil"
	"github.com/gin-gonic/gin"
)

func I18nMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = "Indonesia" // Default
		}

		// Set in Gin context
		c.Set("language", lang)

		// Set in standard context for usecases
		ctx := c.Request.Context()
		ctx = ctxutil.WithLanguage(ctx, lang)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
