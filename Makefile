.PHONY: build run test clean docker-build k8s-deploy run-node1 run-node2 run-node3 run-cluster clean-data status stop-cluster

# Build the application
build:
	go build -o bin/raftkv ./cmd/raftkv

# Run the application
run: build
	./bin/raftkv --config configs/node1.yaml

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf data/

# Clean data directories
clean-data:
	rm -rf data/node1 data/node2 data/node3

# Build Docker image
docker-build:
	docker build -t raftkv:latest .

# Deploy to Kubernetes
k8s-deploy:
	kubectl apply -f k8s/deployment.yaml

# Start minikube cluster
k8s-start:
	minikube start

# Stop minikube cluster
k8s-stop:
	minikube stop

# Get minikube IP
k8s-ip:
	minikube ip

# Port forward the service
k8s-port-forward:
	kubectl port-forward service/raftkv 8080:8080 8001:8001

# Run individual nodes
run-node1:
	go build -o bin/raftkv ./cmd/raftkv
	./bin/raftkv --config configs/node1.yaml

run-node2:
	go build -o bin/raftkv ./cmd/raftkv
	./bin/raftkv --config configs/node2.yaml

run-node3:
	go build -o bin/raftkv ./cmd/raftkv
	./bin/raftkv --config configs/node3.yaml

# Run entire cluster
run-cluster: build
	./bin/raftkv --config configs/node1.yaml & echo $$! > data/node1.pid
	sleep 2
	./bin/raftkv --config configs/node2.yaml & echo $$! > data/node2.pid
	sleep 2
	./bin/raftkv --config configs/node3.yaml & echo $$! > data/node3.pid

# Check cluster status
status:
	@echo "Node 1 status:"
	@curl -s http://localhost:8080/v1/status || echo "Node 1 is not running"
	@echo "\nNode 2 status:"
	@curl -s http://localhost:8081/v1/status || echo "Node 2 is not running"
	@echo "\nNode 3 status:"
	@curl -s http://localhost:8082/v1/status || echo "Node 3 is not running"

# Stop all nodes
stop-cluster:
	@if [ -f data/node1.pid ]; then kill $$(cat data/node1.pid) 2>/dev/null || true; rm data/node1.pid; fi
	@if [ -f data/node2.pid ]; then kill $$(cat data/node2.pid) 2>/dev/null || true; rm data/node2.pid; fi
	@if [ -f data/node3.pid ]; then kill $$(cat data/node3.pid) 2>/dev/null || true; rm data/node3.pid; fi
	@echo "All nodes stopped." 