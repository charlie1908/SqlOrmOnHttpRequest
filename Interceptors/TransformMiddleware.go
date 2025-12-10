package Interceptors

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	"io"
	"net/http"
	"strings"
)

// TransformPersonMiddleware Middleware Kritik User Verilerini Transform Ediyor
func TransformMiddleware[T any]() gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []T

		// Body'yi oku
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Tekrar kullanılabilir hale getir

		// Önce []T olarak dene
		if err := json.Unmarshal(bodyBytes, &items); err != nil {
			// Olmaz ise tekil T dene
			var single T
			if err := json.Unmarshal(bodyBytes, &single); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid JSON: " + err.Error(),
				})
				c.Abort()
				return
			}
			items = []T{single}
		}

		// Doğrudan Core.ValidateStruct çağır
		errs := Core.ValidateStruct(&items)
		if len(errs) > 0 {
			var messages []string
			for _, e := range errs {
				messages = append(messages, e.Error())
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Validation failed",
				"detail": strings.Join(messages, "; "),
			})

			c.Abort()
			return
		}

		// Context'e koy
		if len(items) > 1 {
			c.Set("payload", items)
		} else if len(items) == 1 {
			c.Set("payload", items[0])
		} else {
			c.Abort()
			return
		}
		c.Next()
	}
}
