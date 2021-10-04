package router

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/gin-gonic/gin"
)

func Run() error {
	engine := gin.New()
	engine.Use(gin.Recovery())
	//api := engine.Group("api")
	//{
	//
	//}
	return engine.Run(config.GetConfig().Address)
}
