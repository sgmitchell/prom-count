lint:
	go fmt
	go vet

build:
	go build -o prom-count .

test: build
	go test ./...
