package Interceptors

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	"httpRequestName/Model"
	shared "httpRequestName/Shared"
	"io"
	"log"
	"net/http"
	"time"
)

func InsertAuditLogMiddleware(operationName string, tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Header'dan UserName al
		username := c.GetHeader("UserName")
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing UserName header"})
			c.Abort()
			return
		}

		// Body'yi oku
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			c.Abort()
			return
		}

		// Body tekrar kullanılabilir hale getir
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Audit log verisi hazırla
		auditData := Model.AuditLog{
			UserName:      username,
			TableName:     tableName,
			OperationName: operationName,
			JsonModel:     string(bodyBytes),
			DateTime:      time.Now(),
		}

		// Logu ElasticSearch'e gönder
		if err := Core.InsertLog(auditData, shared.Config.ELASTICAUDITINDEX); err != nil {
			log.Println("Audit Log Error:", err)
			// Log hatası kullanıcıya gösterilmez, işlem devam eder
		}

		// İşlem devam etsin
		c.Next()
	}
}
