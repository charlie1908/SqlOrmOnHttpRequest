package Interceptors

import (
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	"httpRequestName/Model"
	shared "httpRequestName/Shared"
	"net/http"
	"strings"
	"time"
)

func AuthMiddleware(moduleName int, actionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		var userName = c.GetHeader("UserName")
		if token == "" || !ValidateToken(token, userName, c) {
			Core.LogElasticError(userName, moduleName, actionName, "Invalid Token Error", 401)
			errorMessage := Core.Translate("unauthorized", nil)
			//c.JSON(http.StatusUnauthorized, gin.H{"error": "Yetkisiz erişim"})
			c.JSON(http.StatusUnauthorized, gin.H{"error": errorMessage})
			c.Abort()
			return
		}
		c.Next()
	}
}

func ValidateToken(tok string, userName string, c *gin.Context) bool {
	redisClient := Core.GetRedisClient()
	var redisToken = new(string)
	var err = redisClient.GetKey(Core.GenerateRedisKey(userName, false), redisToken)
	if err != nil {
		return false
	}
	if *redisToken == tok { //Gelen Token dogru ise
		var lifeTime, errLifetime = redisClient.GetLifeTimeMinutes(Core.GenerateRedisKey(userName, false)) //Expire suresi alinir.
		if errLifetime != nil {
			return false
		}
		//Token Expire suresine 15 dakka kalmis ise degistirilir..
		if lifeTime <= 15 { //1. Kontrol
			usermutex := Core.GetUserMutex(userName) //go get github.com/patrickmn/go-cache
			usermutex.Lock()                         //Kullanici bazli Lock
			defer usermutex.Unlock()
			//2.Kere Kontrol
			var lifeTime, errLifetime = redisClient.GetLifeTimeMinutes(Core.GenerateRedisKey(userName, false)) //2.kere Expire suresi alinir.
			if errLifetime != nil {
				return false
			}
			if lifeTime <= 15 { //2.Kontrol
				var redisRefreshToken = new(string)
				var err2 = redisClient.GetKey(Core.GenerateRedisKey(userName, true), redisRefreshToken)
				if err2 != nil {
					return true // Hata olustugu icin Tokenlari yenilemeden gececek. Ama var olan Token dogru.
				}

				//RefreshToken Header'dan alinir.
				var refreshToken = c.GetHeader("RefreshToken")
				if strings.TrimSpace(refreshToken) != "" {
					var redisRefreshToken = new(string)
					var errRefreshToken = redisClient.GetKey(Core.GenerateRedisKey(userName, true), redisRefreshToken)
					if errRefreshToken == nil { //RefreshToken Redis'de var ise isleme devam edilir. Yok ise var olan Token ile devam edilir.
						if *redisRefreshToken == refreshToken { //Refresh Token da dogru ise Tokenlar yenilenir.
							//---------------SAVE CURRENT TOKEN & REFRESHTOKEN AS TOKEN_OLD, REFRESHTOKEN_OLD---------
							//Move Current Token to Token_OLD for 1 minute
							var currentRedisToken = new(string)
							var currentRedisRefreshToken = new(string)
							var errToken = redisClient.GetKey(Core.GenerateRedisKey(userName, false), currentRedisToken)
							if errToken != nil {
								return true // Hata olustugu icin Tokeni yenilemeden gececek. Ama var olan Token dogru.
							}
							var errSetoOldToken = redisClient.SetKey(Core.GenerateRedisKey(userName, false)+"-Old", *currentRedisToken, 1*time.Minute)
							if errSetoOldToken != nil {
								return true
							}
							//Move Current refreshToken to RefreshToken_OLD for 1 minute
							var errRefreshToken = redisClient.GetKey(Core.GenerateRedisKey(userName, true), currentRedisRefreshToken)
							if errRefreshToken != nil {
								return true
							}
							var errSetRefreshTokenOld = redisClient.SetKey(Core.GenerateRedisKey(userName, true)+"-Old", *currentRedisRefreshToken, 1*time.Minute)
							if errSetRefreshTokenOld != nil {
								return true
							}
							//------------------SAVED TOKEN_OLD---------------------------------------------
							//Create New Token
							newToken := Model.GenerateToken(36)
							err = redisClient.SetKey(Core.GenerateRedisKey(userName, false), newToken, shared.Config.TOKENEXPIRETIME)
							if err != nil {
								return true // Hata olustugu icin Tokeni yenilemeden gececek. Ama var olan Token dogru.
							}
							c.Header("Authorization", newToken)
							c.Set("Authorization", newToken) // Bu kisim NewServiceResponse'dan cekilir.
							//Create New RefreshToken
							newrefreshToken := Model.GenerateToken(36)
							err = redisClient.SetKey(Core.GenerateRedisKey(userName, true), newrefreshToken, shared.Config.REFRESHTOKENEXPIRETIME)
							if err != nil {
								return true // Hata olustugu icin RefreshToken'i yenilemeden gececek. Ama var olan Token dogru.
							}
							c.Header("RefreshToken", newrefreshToken)
							c.Set("RefreshToken", newrefreshToken) // Bu kisim NewServiceResponse'dan cekilir.
						}
					}
				}
			}
		}
		return true
	}
	//Check Old Token
	var redisToken_Old = new(string)
	var errOld = redisClient.GetKey(Core.GenerateRedisKey(userName, false)+"-Old", redisToken_Old)
	if errOld != nil {
		return false
	}
	if *redisToken_Old == tok { //Gelen Token-Old dogru ise
		return true
	}
	return false
}
