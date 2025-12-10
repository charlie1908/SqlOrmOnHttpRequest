package Model

import (
	"github.com/gin-gonic/gin"
	"time"
)

type ServiceResponse[T any] struct {
	List                []T
	Entity              T
	Count               int
	Token               string
	RefreshToken        string
	CreatedTokenTime    time.Time
	ValidationErrorList []string
	Error               error
	Message             string
}

func NewServiceResponse[T any](c *gin.Context) ServiceResponse[T] {
	resp := ServiceResponse[T]{CreatedTokenTime: time.Now()}
	if token, _ := c.Get("Authorization"); token != "" && token != nil {
		resp.Token = token.(string)
	}
	if refresh, _ := c.Get("RefreshToken"); refresh != "" && refresh != nil {
		resp.RefreshToken = refresh.(string)
	}
	return resp
}
