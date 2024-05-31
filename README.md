# distributed-system-golang

install golang
install docker
install kubectl
install minikube
install helm

## Command

minikube start
helm repo add hashicorp <https://helm.releases.hashicorp.com>
helm install consul hashicorp/consul --set global.name=consul

docker build -t api-gateway:latest -f api-gateway/Dockerfile api-gateway/
docker build -t service-a:latest -f services/service-a/Dockerfile services/service-a/
docker build -t service-b:latest -f services/service-b/Dockerfile services/service-b/

minikube image load api-gateway:latest
minikube image load service-a:latest
minikube image load service-b:latest

kubectl apply -f nginx-ingress/kubernetes/nginx-ingress-deployment.yaml
kubectl apply -f api-gateway/kubernetes/api-gateway-deploymeent.yaml
kubectl apply -f services/service-a/kubernetes/service-a-deploymennt.yaml
kubectl apply -f services/service-b/kubernetes/service-b-deployment.yaml

kubectl get pods
kubectl get deployments
kubectl get services
kubectl get events
kubectl logs <pod-name>
kubectl get svc
kubectl get ingress
kubectl describe deployment api-gateway

          minikube service api-gateway --url  
kubectl scale deployment service-a --replicas=6
