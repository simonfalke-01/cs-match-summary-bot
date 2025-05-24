package webhooks

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func StartServer(host, port string) error {
	r := gin.Default()
	r.POST("/demoReady", demoReadyHandler)
	return r.Run(fmt.Sprintf("%s:%s", host, port))
}

func demoReadyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "received"})
}