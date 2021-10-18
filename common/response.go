package common

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

type Code string

type Resp struct {
	Status int
	Body   gin.H
}

const (
	SUCCESS = "SUCCESS"
	FAIL    = "FAIL"
)

var BadRequestErr = fmt.Errorf("bad request")

func Response(ctx *gin.Context, code Code, data interface{}) (status int, body gin.H) {
	if code == FAIL {
		switch data.(type) {
		case string:
			data = data.(string)
			body = gin.H{
				"Code":    code,
				"Message": data,
				"Data":    nil,
			}
		default:
			body = gin.H{
				"Code":    code,
				"Message": nil,
				"Data":    data,
			}
		}
		ctx.JSON(status, body)
		return status, body
	}
	body = gin.H{
		"Code":    code,
		"Message": nil,
		"Data":    data,
	}
	ctx.JSON(status, body)
	return status, body
}

func ResponseError(ctx *gin.Context, err error) {
	Response(ctx, FAIL, err.Error())
}

func ResponseBadRequestError(ctx *gin.Context) {
	Response(ctx, FAIL, BadRequestErr.Error())
}
func ResponseSuccess(ctx *gin.Context, data interface{}) {
	Response(ctx, SUCCESS, data)
}
