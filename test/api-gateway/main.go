package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/hashicorp/consul/api"
)

type Service struct {
	Name  string
	Index uint32
}

var (
	services = map[string]*Service{
		"service-a": &Service{Name: "service-a"},
		"service-b": &Service{Name: "service-b"},
	}
)

func getServiceAddress(serviceName string) (string, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return "", err
	}

	svc, ok := services[serviceName]
	if !ok {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	serviceInstances, _, err := client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", err
	}

	if len(serviceInstances) == 0 {
		return "", fmt.Errorf("no healthy instances of service %s found", serviceName)
	}

	instance := serviceInstances[atomic.AddUint32(&svc.Index, 1)%uint32(len(serviceInstances))].Service
	return fmt.Sprintf("%s:%d", instance.Address, instance.Port), nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Path[1:]
	address, err := getServiceAddress(serviceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://%s", address))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(body)
}

func main() {
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
