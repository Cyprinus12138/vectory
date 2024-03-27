package http_handler

import (
	"github.com/Cyprinus12138/otelgin"
	"github.com/Cyprinus12138/vectory/internal/utils/monitor"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/gin-gonic/gin"
)

func BuildRouter() *gin.Engine {
	router := gin.Default()
	router.Use(otelgin.Middleware(pkg.ClusterName))

	router.GET("/metrics", monitor.HandleFunc())

	return router
}
