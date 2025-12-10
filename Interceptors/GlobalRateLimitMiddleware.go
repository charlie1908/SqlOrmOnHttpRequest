package Interceptors

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	"httpRequestName/Core"
	"net"
	"net/http"
	"strings"
)

func GlobalRateLimitMiddleware(limiter *limiter.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Swagger gibi endpointleri dışarıda bırak
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") || c.Request.URL.Path == "/favicon.ico" {
			c.Next()
			return
		}

		//User Bazli Rate Limit
		//Token Yok ise RateLimit uygulanmiyacak mi ?
		/*var ip = c.Request.RemoteAddr
		  fmt.Println(ip)
		  token := c.GetHeader("Authorization")
		  var userName = c.GetHeader("UserName")
		  key := ""
		  if token == "" || !ValidateToken(token, userName, c) {
		  	//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid Token for Rate Limiter Error"})
		  	//return
		  	key = "global"
		  }*/
		//key := "global" // tüm istekler için ortak key
		key := getClientIP(c) // User IP bazli RateLimit key

		limitCtx, err := limiter.Get(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Rate limiter error"})
			//c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": errorMessage})
			return
		}

		// Header'lara rate limit bilgisi ekle (opsiyonel)
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limitCtx.Limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", limitCtx.Remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", limitCtx.Reset))

		if limitCtx.Reached {
			errorMessage := Core.Translate("ratelimit_exceed", nil)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				//"error": "Rate limit exceeded. Try again later.",
				"error": errorMessage,
			})
			return
		}

		c.Next()
	}
}

func getClientIP(c *gin.Context) string {
	// X-Forwarded-For başlığını al
	remoteAddr := strings.TrimSpace(c.Request.Header.Get("X-Forwarded-For"))

	// Eğer boşsa, RemoteAddr kullan
	if remoteAddr == "" {
		remoteAddr = strings.TrimSpace(c.Request.RemoteAddr)
	}

	// Eğer yine boşsa "global" döndür
	if remoteAddr == "" {
		remoteAddr = "global"
	}

	// IP:port varsa, sadece IP'yi al
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return ip
}
