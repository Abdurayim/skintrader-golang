package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	LangEN = "en"
	LangRU = "ru"
	LangUZ = "uz"
)

var supportedLanguages = map[string]bool{
	LangEN: true,
	LangRU: true,
	LangUZ: true,
}

func Language() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := LangEN

		// Check Accept-Language header
		acceptLang := c.GetHeader("Accept-Language")
		if acceptLang != "" {
			// Parse primary language tag
			parts := strings.Split(acceptLang, ",")
			if len(parts) > 0 {
				primary := strings.TrimSpace(strings.Split(parts[0], ";")[0])
				code := strings.ToLower(strings.Split(primary, "-")[0])
				if supportedLanguages[code] {
					lang = code
				}
			}
		}

		// Check query param override
		if q := c.Query("lang"); q != "" {
			code := strings.ToLower(q)
			if supportedLanguages[code] {
				lang = code
			}
		}

		c.Set("language", lang)
		c.Next()
	}
}

func GetLanguage(c *gin.Context) string {
	if lang, ok := c.Get("language"); ok {
		return lang.(string)
	}
	return LangEN
}
