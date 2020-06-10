package pushgateway

import (
	"github.com/gin-gonic/gin"

	"net/http"
	"dynamic-sharding/pkg/sd"
	"log"
)

func PushMetricsGetHash(c *gin.Context) {

	path := c.Request.URL.Path

	node, err := sd.PgwNodeRing.GetNode(path)
	if err != nil {
		c.String(http.StatusInternalServerError, "get_node_from_hashring_error")
	}

	nextUrl := "http://" + node + path
	log.Printf("[PushMetrics][request_path:%s][redirect_url:%s]", path, nextUrl)
	c.String(http.StatusOK, "nextUrl:"+nextUrl)

}

func PushMetricsRedirect(c *gin.Context) {

	path := c.Request.URL.Path

	node, err := sd.PgwNodeRing.GetNode(path)
	if err != nil {
		c.String(http.StatusInternalServerError, "get_node_from_hashring_error")
	}

	nextUrl := "http://" + node + path
	log.Printf("[PushMetrics][request_path:%s][redirect_url:%s]", path, nextUrl)
	//c.Redirect(http.StatusMovedPermanently, nextUrl)
	//c.Redirect(http.StatusTemporaryRedirect, nextUrl)
	c.Redirect(http.StatusPermanentRedirect, nextUrl)
	c.Abort()

}
