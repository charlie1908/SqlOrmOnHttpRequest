package Core

import "reflect"

var ModulesEnum = newModule()
var UserActionEnum = newUserActionEnum()
var LoginActionEnum = newLoginActionEnum()
var SystemActionEnum = newSystemActionEnum()
var AuthActionEnum = newAuthActionEnum()

/*var ModuleEnumToString = map[int]string{
	ModulesEnum.Login:  "Login",
	ModulesEnum.Users:  "Users",
	ModulesEnum.System: "System",
}

var LoginEnumToString = map[int]string{
	LoginActionEnum.Login: "Login",
}

var UserEnumToString = map[int]string{
	UserActionEnum.GetUserById:    "GetUserById",
	UserActionEnum.BulkInsertUser: "BulkInsertUser",
	UserActionEnum.InsertUser:     "InsertUser",
	UserActionEnum.GetAllUsers:    "GetAllUsers",
}

var SystemEnumToString = map[int]string{
	SystemActionEnum.UnKnown:      "UnKnown",
	SystemActionEnum.InvalidToken: "CheckToken",
	SystemActionEnum.SqlPing:      "SqlPing",
	SystemActionEnum.SqlOpen:      "SqlOpen",
}*/

func newModule() *moduleEnum {
	return &moduleEnum{
		Users:  1,
		Login:  2,
		System: 3,
	}
}

func newLoginActionEnum() *loginActionEnum {
	return &loginActionEnum{
		Login: 1,
	}
}

func newSystemActionEnum() *systemActionEnum {
	return &systemActionEnum{
		UnKnown:      1,
		SqlOpen:      2,
		SqlPing:      4,
		InvalidToken: 8,
	}
}

func newUserActionEnum() *userActionEnum {
	return &userActionEnum{
		InsertUser:     1,
		GetUserById:    2,
		GetAllUsers:    4,
		BulkInsertUser: 8,
		UpdateUser:     16,
	}
}

func newAuthActionEnum() *authActionEnum {
	return &authActionEnum{
		Login: 1,
	}
}

type moduleEnum struct {
	Login  int
	Users  int
	System int
}

type userActionEnum struct {
	InsertUser     int
	BulkInsertUser int
	GetUserById    int
	GetAllUsers    int
	UpdateUser     int
}

type loginActionEnum struct {
	Login int
}

type systemActionEnum struct {
	UnKnown      int
	SqlOpen      int
	SqlPing      int
	InvalidToken int
}
type authActionEnum struct {
	Login int
}

func GetEnumName(enum interface{}, value int) string {
	v := reflect.ValueOf(enum).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		if int(v.Field(i).Int()) == value {
			return t.Field(i).Name
		}
	}
	return "Unknown"
}
