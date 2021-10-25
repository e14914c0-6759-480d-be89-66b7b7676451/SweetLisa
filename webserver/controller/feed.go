package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"path"
	"strings"
)

func GetChatRSS(ctx *gin.Context) {
	chatIdentifier := ctx.Param("ChatIdentifier")
	chatIdentifier = strings.TrimSuffix(chatIdentifier, path.Ext(chatIdentifier))

	str, err := service.GetChatFeedRSS(nil, chatIdentifier)
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	ctx.Header("Content-Type", "application/rss+xml")
	ctx.Writer.WriteString(str)
}
