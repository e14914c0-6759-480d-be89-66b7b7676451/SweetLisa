package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/gin-gonic/gin"
	"path"
)

func GetChat(ctx *gin.Context) {
	chatIdentifier := ctx.Param("ChatIdentifier")
	switch path.Ext(chatIdentifier) {
	case ".rss":
		GetChatRSS(ctx)
	default:
		common.ResponseBadRequestError(ctx)
	}
}
