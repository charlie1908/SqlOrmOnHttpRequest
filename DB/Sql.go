package DB

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gin-gonic/gin"
	"httpRequestName/Core"
	shared "httpRequestName/Shared"
	"log"
)

/*
SQL DB SCRIPT

CREATE TYPE dbo.UserTableType AS TABLE
(

	Name NVARCHAR(100),
	Age INT,
	Gender NVARCHAR(20),
	UserName NVARCHAR(100),
	Password NVARCHAR(100)

)

CREATE PROCEDURE dbo.InsertUsersBulk

	@Users dbo.UserTableType READONLY

AS
BEGIN

	INSERT INTO GOUSERS (Name, Age, Gender, UserName, Password)
	SELECT Name, Age, Gender, UserName, Password FROM @Users

END
*/

func SqlOpen(c ...*gin.Context) (*sql.DB, *context.Context) {
	var _userName = "Unknown"
	if len(c) > 0 {
		userNameHeader := GetUserNameFromContext(c[0])
		if userNameHeader != "" {
			decryptUserName, err := Core.Decrypt(userNameHeader, shared.Config.SECRETKEY)
			if err == nil {
				_userName = decryptUserName
			} else {
				_userName = userNameHeader
			}
		}
	}
	sqlConnection, _ := Core.Decrypt(shared.Config.SQLURL, shared.Config.SECRETKEY)
	db, err := sql.Open("sqlserver", sqlConnection)
	if err != nil {
		log.Fatal(err)
		//Core.LogElasticError(_userName, Core.ModulesEnum.System, Core.SystemEnumToString[Core.SystemActionEnum.SqlOpen], fmt.Sprint(err), 500)
		Core.LogElasticError(_userName, Core.ModulesEnum.System, Core.GetEnumName(Core.SystemActionEnum, Core.SystemActionEnum.SqlOpen), fmt.Sprint(err), 500)
	}
	err = db.Ping()
	if err != nil {
		fmt.Println("Error:", err)
		//Core.LogElasticError(_userName, Core.ModulesEnum.System, Core.SystemEnumToString[Core.SystemActionEnum.SqlPing], fmt.Sprint(err), 500)
		Core.LogElasticError(_userName, Core.ModulesEnum.System, Core.GetEnumName(Core.SystemActionEnum, Core.SystemActionEnum.SqlPing), fmt.Sprint(err), 500)
	}
	ctx, _ := context.WithTimeout(context.Background(), shared.Config.SQLTIMEOUT)
	return db, &ctx
}

func HandleError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func GetUserNameFromContext(c *gin.Context) string {
	userName := c.GetHeader("UserName")
	if userName != "" {
		return userName
	}

	// Eğer header boşsa, Set ile eklenmiş değeri dene
	if val, exists := c.Get("UserName"); exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return "Unknown"
}
