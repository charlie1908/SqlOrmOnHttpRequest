package Model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Person struct {
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Gender   string `json:"gender"`
	UserName string `json:"username" validate:"encrypt"`
	Password string `json:"password" validate:"hash"`
	Id       int    `json:"id"`
}

func (p *Person) TableName() string {
	return "GoUsers"
}

type LoginModel struct {
	Username string `validate:"encrypt"`
	Password string
}

type LoginLog struct {
	PostDate time.Time
	Username string
}

var GenderType = genderType()

func genderType() *gender {
	return &gender{
		Male:   "Male",
		Female: "Female",
	}
}

type gender struct {
	Male   string
	Female string
}

func Filters(p []Person, f func(Person) bool) []Person {
	var result []Person
	for _, p := range p {
		if f(p) {
			result = append(result, p)
		}
	}
	return result
}

/*func GenerateToken(n int) string {
	//Generate 8 Character Token
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:n]
}*/

func GenerateToken(n int) string {
	if n <= 0 {
		return ""
	}

	byteLen := (n + 1) / 2 // hex gives 2 chars per byte
	bytes := make([]byte, byteLen)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)[:n]
}

type AuditLog struct {
	UserName      string
	JsonModel     string
	OperationName string
	TableName     string
	DateTime      time.Time
}

type ErrorLog struct {
	UserName   string
	Message    string
	ModuleName string
	ActionName string
	DateTime   time.Time
	ErrorCode  int
}

type RawValue struct {
	Expr string
}
