package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

const (
	ttl     = time.Second * 8
	checkId = "check_health"
)

type Service struct {
	consulClient *api.Client
}

func NewService() *Service {
	client, err := api.NewClient(&api.Config{})
	if err != nil {
		log.Fatal(err)
	}
	return &Service{
		consulClient: client,
	}
}

func (s *Service) updateHealthCheck() {
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
	check := &api.AgentServiceCheck{
		DeregisterCriticalServiceAfter: ttl.String(),
		TLSSkipVerify:                  true,
		TTL:                            ttl.String(),
		CheckID:                        checkId,
	}
	register := &api.AgentServiceRegistration{
		ID:      "service-b",
		Name:    "service-b",
		Tags:    []string{"service-b"},
		Address: "127.0.0.1",
		Port:    8083,
		Check:   check,
	}
	query := map[string]interface{}{
		"type":        "service",
		"service":     "service-b",
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
	router := gin.Default()
	router.GET("/service-b", func(c *gin.Context) {
		c.String(http.StatusOK, "service b")
	})

	s.registerServiceWithConsul()
	go s.updateHealthCheck()

	if err := router.Run(":8083"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	s := NewService()
	s.Start()
}
