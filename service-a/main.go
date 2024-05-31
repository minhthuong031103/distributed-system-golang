package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

const (
	ttl = time.Second * 8
)

var (
	address = os.Getenv("POD_IP")
)

type Service struct {
	consulClient *api.Client
	requestCount uint32
	serverName   string
}

func NewService(serverName string) *Service {
	consulAddress := os.Getenv("CONSUL_HTTP_ADDR")
	if consulAddress == "" {
		consulAddress = "http://consul-server.default.svc.cluster.local:8500"
	}
	client, err := api.NewClient(&api.Config{
		Address: consulAddress,
	})
	if err != nil {
		log.Fatal(err)
	}
	return &Service{
		consulClient: client,
		serverName:   serverName,
	}
}

func (s *Service) updateHealthCheck() {
	checkId := "service-a-check" + address
	ticker := time.NewTicker(time.Second * 5)
	for {
		err := s.consulClient.Agent().UpdateTTL(checkId, "online", api.HealthPassing)
		if err != nil {
			log.Fatal(err)
		}
		<-ticker.C
	}
}

func (s *Service) registerServiceWithConsul() {
	checkId := "service-a-check" + address

	check := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: ttl.String(),
		TLSSkipVerify:                  true,
		TTL:                            ttl.String(),
		CheckID:                        checkId,
	}

	// Get the IP address and port from environment variables
	port, err := strconv.Atoi(os.Getenv("SERVICE_PORT"))
	fmt.Println("address: ", address)
	if address == "" || err != nil {
		log.Fatal("POD_IP or SERVICE_PORT environment variable is not set")
	}

	//generate the uuid for the service
	register := &api.AgentServiceRegistration{
		ID:      "service-a-" + address,
		Name:    "service-a",
		Tags:    []string{"service-a"},
		Address: address,
		Port:    port,
		Check:   check,
	}
	query := map[string]interface{}{
		"type":        "service",
		"service":     "service-a",
		"passingonly": true,
	}
	plan, err := watch.Parse(query)
	if err != nil {
		log.Fatal(err)
	}

	plan.HybridHandler = func(index watch.BlockingParamVal, result interface{}) {
		switch msg := result.(type) {
		case []*api.ServiceEntry:
			for _, entry := range msg {
				fmt.Println("new member joined ", entry.Service)
			}
		}
	}

	go func() {
		plan.RunWithConfig("", &api.Config{})
	}()

	if err := s.consulClient.Agent().ServiceRegister(register); err != nil {
		log.Fatal(err)
	}
}

func (s *Service) Start() {
	address := os.Getenv("POD_IP")
	port := os.Getenv("SERVICE_PORT")
	router := gin.Default()
	router.GET("/service-a", func(c *gin.Context) {
		atomic.AddUint32(&s.requestCount, 1)
		c.JSON(http.StatusOK, gin.H{
			"message":       "Hello from service-a" + address + ":" + port,
			"server_name":   s.serverName,
			"request_count": atomic.LoadUint32(&s.requestCount),
		})
	})

	s.registerServiceWithConsul()
	go s.updateHealthCheck()

	if err := router.Run(":8082"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	s := NewService("service-a-8082")
	s.Start()
}
