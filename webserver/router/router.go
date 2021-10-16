package router

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/webserver/controller"
	"github.com/gin-gonic/gin"
)

func Run() error {
	engine := gin.New()
	engine.Use(gin.Recovery())
	api := engine.Group("/api/:ChatIdentifier")
	{
		api.GET("ticket", controller.GetTicket)
		api.GET("verification", controller.GetVerification)
	}
	server:= api.Group("server")
	{
		server.POST("register", controller.PostRegister)
	}
	return engine.Run(config.GetConfig().Address)
}
