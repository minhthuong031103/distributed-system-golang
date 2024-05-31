package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

type Gateway struct {
	consulClient *api.Client
	serviceIndex map[string]*uint32
}

func NewGateway() *Gateway {
	consulConfig := api.DefaultConfig()

	// consulConfig.Address = os.Getenv("CONSUL_HTTP_ADDR")
	consulConfig.Address = "http://consul-server.default.svc.cluster.local:8500"
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &Gateway{
		consulClient: consulClient,
		serviceIndex: make(map[string]*uint32),
	}
}

func (g *Gateway) getNextServiceInstance(serviceName string) (string, error) {
	services, _, err := g.consulClient.Health().Service(serviceName, "", true, nil)
	fmt.Println(services)
	if err != nil {
		return "", err
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	if _, exists := g.serviceIndex[serviceName]; !exists {
		g.serviceIndex[serviceName] = new(uint32)
	}

	idx := atomic.AddUint32(g.serviceIndex[serviceName], 1)
	service := services[(idx-1)%uint32(len(services))]

	return fmt.Sprintf("http://%s:%d", service.Service.Address, service.Service.Port), nil
}

func (g *Gateway) proxyHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetURL, err := g.getNextServiceInstance(serviceName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		c.Request.URL.Path = c.Param("path")
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func main() {
	gateway := NewGateway()
	router := gin.Default()

	router.GET("/service-a/*path", gateway.proxyHandler("service-a"))
	router.GET("/service-b/*path", gateway.proxyHandler("service-b"))

	router.Run(":8080")
}
