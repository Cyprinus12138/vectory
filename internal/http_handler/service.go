package http_handler

import (
	"github.com/Cyprinus12138/vectory/internal/utils/monitor"
	"github.com/gin-gonic/gin"
)

func BuildRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/metrics", monitor.HandleFunc())

	return router
}
