package pushgateway

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {

	authapi := r.Group("/metrics/job")
	authapi.GET("/*any", PushMetricsGetHash)
	authapi.PUT("/*any", PushMetricsRedirect)
	authapi.POST("/*any", PushMetricsRedirect)

	tapi := r.Group("/test")
	tapi.GET("/v1", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, I'm pgw gateway+ (｡A｡)")
	})
}
