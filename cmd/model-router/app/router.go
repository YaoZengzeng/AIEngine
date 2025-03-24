package app

import (
	"net/http"

	"AIEngine/pkg/filters/auth"
	"AIEngine/pkg/router"

	"github.com/gin-gonic/gin"
)

// Starts router
func startRouter(stop <-chan struct{}) {
	// Your application logic here
	engine := gin.Default()
	// TODO: add middle ware
	// engine.Use()

	engine.Use(auth.Authenticate)
	engine.Use(auth.Authorize)

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r := router.NewRouter()
	go r.Run(stop)

	// vllm use /v1 prefix, not sure other frameworks
	engine.Any("/v1", r.HandlerFunc())

	// TODO: customize the port
	go engine.Run(":8080")
	go engine.RunTLS(":8443", "cert.pem", "key.pem")
	<-stop
}
