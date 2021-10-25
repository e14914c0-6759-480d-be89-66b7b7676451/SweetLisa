package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/gin-gonic/gin"
	"path"
	"strings"
)

func SplitChatIdentifier(ctx *gin.Context) (id string, ext string) {
	chatIdentifier := ctx.Param("ChatIdentifier")
	ext = path.Ext(chatIdentifier)
	return strings.TrimSuffix(chatIdentifier, ext), ext
}

func GetChat(ctx *gin.Context) {
	_, ext := SplitChatIdentifier(ctx)
	switch strings.ToLower(ext) {
	case ".rss", ".atom", ".json":
		GetChatFeed(ctx)
	default:
		common.ResponseBadRequestError(ctx)
	}
}
