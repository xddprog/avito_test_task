APP_NAME=reviewer-service
BIN_DIR=bin
K6_SCRIPT=tests/load-testing/pr-test.js

E2E_TEST_DIR=tests/e2e-testing

.PHONY: build run clean docker-build docker-run k6-run e2e-test lint install-lint

build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/app

run:
	go mod download
	go mod tidy
	go run ./cmd/app

clean:
	rm -rf $(BIN_DIR)
	docker-compose down

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker-compose up -d --build

k6-run:
	BASE_URL=http://localhost:8080 k6 run $(K6_SCRIPT)

e2e-test:
	go test -v -count=1 ./$(E2E_TEST_DIR)

lint:
	@golangci-lint run ./...

install-lint:
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint-fix:
	@golangci-lint run ./... --fix