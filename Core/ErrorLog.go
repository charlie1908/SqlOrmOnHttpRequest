package Core

import (
	"httpRequestName/Model"
	shared "httpRequestName/Shared"
	"log"
	"time"
)

// func LogElasticError(userName string, moduleName int, actionName int, message string, errorCode int) bool {
func LogElasticError(userName string, moduleName int, actionName string, message string, errorCode int) bool {
	// Audit log verisi hazırla
	//var _username, _ = Decrypt(userName, shared.Config.SECRETKEY)
	errorData := Model.ErrorLog{
		UserName: userName,
		//ModuleName: ModuleEnumToString[moduleName],
		ModuleName: GetEnumName(ModulesEnum, moduleName),
		//ActionName: LoginEnumToString[actionName],
		ActionName: actionName,
		Message:    message,
		DateTime:   time.Now(),
		ErrorCode:  errorCode,
	}

	// Logu ElasticSearch'e gönder
	if err := InsertLog(errorData, shared.Config.ELASTICERRORINDEX); err != nil {
		log.Println("Login 401 Unauthorized Log Error:", err)
		return false
		// Gecersiz Login denemesi Elastic'e kaydedilir.
	}
	return true
}
