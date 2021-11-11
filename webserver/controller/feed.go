package controller

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/common"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func GetChatFeed(ctx *gin.Context) {
	chatIdentifier, ext := SplitChatIdentifier(ctx)
	var (
		str string
		err error
	)
	// https://github.com/iovxw/rssbot/blob/6a2a08c51d88f29073ff3a929566d9924134a585/src/utlis.rs#L145-L153
	fromRssBot := strings.Contains(ctx.GetHeader("User-Agent"), "https://t.me/")
	switch strings.ToLower(ext) {
	case ".atom":
		str, err = service.GetChatFeed(nil, chatIdentifier, service.FeedFormatAtom, fromRssBot)
	case ".rss":
		str, err = service.GetChatFeed(nil, chatIdentifier, service.FeedFormatRSS, fromRssBot)
	case ".json":
		str, err = service.GetChatFeed(nil, chatIdentifier, service.FeedFormatJSON, fromRssBot)
	default:
		common.ResponseBadRequestError(ctx)
		return
	}
	if err != nil {
		common.ResponseError(ctx, err)
		return
	}
	ctx.Header("Content-Type", "application/rss+xml")
	ctx.Writer.WriteString(str)
}
