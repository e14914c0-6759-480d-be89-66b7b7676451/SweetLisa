package router

import (
	"embed"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/config"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/SweetLisa/webserver/controller"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
)

// relativeFS implements fs.FS
type relativeFS struct {
	root        fs.FS
	relativeDir string
}

func (c relativeFS) Open(name string) (fs.File, error) {
	return c.root.Open(filepath.Join(c.relativeDir, name))
}

func Run(f embed.FS) error {
	engine := gin.New()
	templ := template.Must(template.New("").ParseFS(f, "static/*.tmpl"))
	engine.SetHTMLTemplate(templ)
	engine.Use(gin.Recovery())
	engine.GET("/chat/:ChatIdentifier", func(c *gin.Context) {
		chatIdentifier := c.Param("ChatIdentifier")
		if len(path.Ext(chatIdentifier)) > 1 {
			controller.GetChat(c)
		} else {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"ChatIdentifier": c.Param("ChatIdentifier"),
			})
		}
	})
	webFS := relativeFS{
		root:        f,
		relativeDir: "static",
	}
	fs.WalkDir(webFS, "/", func(path string, info fs.DirEntry, err error) error {
		if path == "/" {
			return nil
		}
		if info.IsDir() {
			engine.StaticFS("/"+info.Name(), http.FS(relativeFS{
				root:        webFS,
				relativeDir: path,
			}))
			return filepath.SkipDir
		}
		engine.GET("/"+info.Name(), func(ctx *gin.Context) {
			ctx.FileFromFS(path, http.FS(webFS))
		})
		return nil
	})
	api := engine.Group("api")

	chat := api.Group("chat/:ChatIdentifier")
	{
		chat.GET("ticket", controller.GetTicket)
		chat.GET("verification", controller.GetVerification)
	}

	ticket := api.Group("ticket/:Ticket")
	{
		ticket.GET("sub", controller.GetSubscription)
		ticket.POST("register", controller.PostRegister)
		ticket.POST("renew", controller.PostRenew)
	}
	return engine.Run(config.GetConfig().Address)
}
