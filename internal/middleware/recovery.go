package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func Recovery(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				logger.Error().
					Interface("panic", r).
					Str("stack", string(stack)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Str("ip", c.ClientIP()).
					Msg("panic recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "Internal server error",
					"code":    "INTERNAL_ERROR",
				})
			}
		}()
		c.Next()
	}
}
