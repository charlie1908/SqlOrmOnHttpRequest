package Interceptors

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	"httpRequestName/Model"
	shared "httpRequestName/Shared"
	"io"
	"log"
	"net/http"
	"time"
)

func InsertLoginLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Body'yi oku
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			c.Abort()
			return
		}
		// Body tekrar kullanılabilir hale getir
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// JSON'u çöz
		var login Model.LoginModel
		if err := json.Unmarshal(bodyBytes, &login); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login JSON", "detail": err.Error()})
			c.Abort()
			return
		}

		// Log verisini hazırla (UserID olmadan)
		logData := Model.LoginLog{
			Username: login.Username,
			PostDate: time.Now(),
		}

		// İşleme devam et (handler'lar çalışsın)
		c.Next()

		if c.Writer.Status() == http.StatusOK { //LOGIN ISLEMI BASARILI ISE
			// Request tamamlandıktan sonra logu kaydet
			if err := Core.InsertLog(logData, shared.Config.ELASTICLOGININDEX); err != nil {
				log.Println("Login Log Error: Failed to insert log:", err)
			}
		}
	}
}
