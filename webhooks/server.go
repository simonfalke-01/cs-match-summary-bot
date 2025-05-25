package webhooks

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// HandlerFunctions holds all the handler functions that can be injected from main package
type HandlerFunctions struct {
	DemoReady   gin.HandlerFunc
	DemoParsed  gin.HandlerFunc
	MatchQuery  gin.HandlerFunc
	UserQuery   gin.HandlerFunc
	GuildQuery  gin.HandlerFunc
}

func StartServer(host, port string, handlers *HandlerFunctions) error {
	r := gin.Default()
	
	// Default handler for demoReady if none provided
	demoReadyHandler := func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "received"})
	}
	if handlers != nil && handlers.DemoReady != nil {
		demoReadyHandler = handlers.DemoReady
	}
	
	// Default handler for demoParsed if none provided
	demoParsedHandler := func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "received"})
	}
	if handlers != nil && handlers.DemoParsed != nil {
		demoParsedHandler = handlers.DemoParsed
	}
	
	// Webhook endpoints
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/demoReady", demoReadyHandler)
		webhooks.POST("/demoParsed", demoParsedHandler)
	}
	
	// API endpoints for querying data
	if handlers != nil {
		api := r.Group("/api/v1")
		{
			if handlers.MatchQuery != nil {
				api.GET("/match/:shareCode", handlers.MatchQuery)
			}
			if handlers.UserQuery != nil {
				api.GET("/user/:steamID", handlers.UserQuery)
			}
			if handlers.GuildQuery != nil {
				api.GET("/guild/:guildID", handlers.GuildQuery)
			}
		}
	}
	
	return r.Run(fmt.Sprintf("%s:%s", host, port))
}