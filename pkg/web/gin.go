package web

import (
	"time"
	"net/http"

	"github.com/gin-gonic/gin"

	"dynamic-sharding/pkg/web/controller/pushgateway"
)

func StartGin(port string, r *gin.Engine) error {

	pushgateway.Routes(r)
	s := &http.Server{
		Addr:           port,
		Handler:        r,
		ReadTimeout:    time.Duration(5) * time.Second,
		WriteTimeout:   time.Duration(5) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err := s.ListenAndServe()
	return err

}
