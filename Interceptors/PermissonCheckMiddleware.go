package Interceptors

import (
	"github.com/gin-gonic/gin"
	"httpRequestName/Service"
	"net/http"
)

func PermissionCheck(controllerID int, actionNumber int, actionName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var userName = c.GetHeader("UserName")
		//var result, err = Service.GetUserPermissions(userName, controllerID, actionNumber, actionName, c)
		var result, err = Service.GetUserPermissionsByRole(userName, controllerID, actionNumber, actionName, c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if !result {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not Authorized!"})
			c.Abort()
			return
		}
		c.Next()
	}
}
