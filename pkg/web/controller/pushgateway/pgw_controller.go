package pushgateway

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"dynamic-sharding/pkg/sd"
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
	c.Redirect(http.StatusTemporaryRedirect, nextUrl)
	//c.Redirect(http.StatusPermanentRedirect, nextUrl)
	c.Abort()

}
