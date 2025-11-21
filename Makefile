APP_NAME=reviewer-service
BIN_DIR=bin
K6_SCRIPT=tests/load-testing/pr-test.js

.PHONY: build run clean docker-build docker-run k6-run

build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/app

run:
	go run ./cmd/app

clean:
	rm -rf $(BIN_DIR)
	docker-compose down

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker-compose up -d --build

# --- Запуск K6 нагрузочного теста ---
k6-run:
	BASE_URL=http://localhost:8080 k6 run $(K6_SCRIPT)
