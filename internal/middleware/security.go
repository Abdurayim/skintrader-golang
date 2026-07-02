package middleware

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "0")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		c.Header("Cross-Origin-Resource-Policy", "cross-origin")
		c.Next()
	}
}

func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(413, gin.H{
				"success": false,
				"message": "Request body too large",
				"code":    "PAYLOAD_TOO_LARGE",
			})
			return
		}
		c.Request.Body = &limitedReader{c.Request.Body, maxBytes}
		c.Next()
	}
}

type limitedReader struct {
	reader    interface{ Read([]byte) (int, error) }
	remaining int64
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	if lr.remaining <= 0 {
		return 0, &payloadTooLargeError{}
	}
	if int64(len(p)) > lr.remaining {
		p = p[:lr.remaining]
	}
	n, err := lr.reader.Read(p)
	lr.remaining -= int64(n)
	return n, err
}

func (lr *limitedReader) Close() error {
	if closer, ok := lr.reader.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

type payloadTooLargeError struct{}

func (e *payloadTooLargeError) Error() string {
	return "request body too large"
}
