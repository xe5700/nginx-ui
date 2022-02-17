package router

import (
	"bufio"
	"encoding/base64"
	"github.com/0xJacky/Nginx-UI/api"
	"github.com/0xJacky/Nginx-UI/frontend"
	"github.com/0xJacky/Nginx-UI/model"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			tmp, _ := base64.StdEncoding.DecodeString(c.Query("token"))
			token = string(tmp)
			if token == "" {
				c.JSON(http.StatusForbidden, gin.H{
					"message": "auth fail",
				})
				c.Abort()
				return
			}
		}

		n := model.CheckToken(token)

		if n < 1 {
			c.JSON(http.StatusForbidden, gin.H{
				"message": "auth fail",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

type serverFileSystemType struct {
	http.FileSystem
}

func (f serverFileSystemType) Exists(prefix string, path string) bool {
	_, err := f.Open(filepath.Join(prefix, path))
	return err == nil
}

func mustFS(dir string) (serverFileSystem static.ServeFileSystem) {

	sub, err := fs.Sub(frontend.DistFS, filepath.Join("dist", dir))

	if err != nil {
		log.Println(err)
	}

	serverFileSystem = serverFileSystemType{
		http.FS(sub),
	}

	return
}

func InitRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())

	r.Use(gin.Recovery())

	r.Use(static.Serve("/", mustFS("")))

	r.NoRoute(func(c *gin.Context) {
		accept := c.Request.Header.Get("Accept")
		if strings.Contains(accept, "text/html") {
			file, _ := mustFS("").Open("index.html")
			stat, _ := file.Stat()
			c.DataFromReader(http.StatusOK, stat.Size(), "text/html",
				bufio.NewReader(file), nil)
		}
	})

	g := r.Group("/api")
	{
		g.GET("install", api.InstallLockCheck)
		g.POST("install", api.InstallNginxUI)

		g.POST("/login", api.Login)
		g.DELETE("/logout", api.Logout)

		g := g.Group("/", authRequired())
		{
			g.GET("/analytic", api.Analytic)

			g.GET("/users", api.GetUsers)
			g.GET("/user/:id", api.GetUser)
			g.POST("/user", api.AddUser)
			g.POST("/user/:id", api.EditUser)
			g.DELETE("/user/:id", api.DeleteUser)

			g.GET("domains", api.GetDomains)
			g.GET("domain/:name", api.GetDomain)
			g.POST("domain/:name", api.EditDomain)
			g.POST("domain/:name/enable", api.EnableDomain)
			g.POST("domain/:name/disable", api.DisableDomain)
			g.DELETE("domain/:name", api.DeleteDomain)

			g.GET("configs", api.GetConfigs)
			g.GET("config/:name", api.GetConfig)
			g.POST("config", api.AddConfig)
			g.POST("config/:name", api.EditConfig)

			g.GET("backups", api.GetFileBackupList)
			g.GET("backup/:id", api.GetFileBackup)

			g.GET("template/:name", api.GetTemplate)

			g.GET("cert/issue/:domain", api.IssueCert)
			g.GET("cert/:domain/info", api.CertInfo)

			// 添加域名到自动续期列表
			g.POST("cert/:domain", api.AddDomainToAutoCert)
			// 从自动续期列表中删除域名
			g.DELETE("cert/:domain", api.RemoveDomainFromAutoCert)
		}
	}

	return r
}
