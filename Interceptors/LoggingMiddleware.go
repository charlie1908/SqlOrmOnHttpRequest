package Interceptors

import (
	"github.com/gin-gonic/gin"
	"log"
	"time"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userName = c.GetHeader("UserName")
		start := time.Now()
		log.Println("Yeni kişi ekleme isteği alındı:", c.Request.Method, c.Request.URL, userName)
		c.Next() // Bir sonraki middleware veya handler'a geç

		duration := time.Since(start)
		log.Println("Elapsed time:", duration)
	}
}
