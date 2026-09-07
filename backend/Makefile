.PHONY: build run worker test test-unit test-integration test-race docker-up docker-down seed lint verify benchmark
build:
	go build -o bin/api ./api
	go build -o bin/worker ./worker
	go build -o bin/seed ./cmd/seed
run:
	go run ./api

worker:
	go run ./worker
test:
	go test ./...
test-unit: test
test-integration:
	go test -tags=integration -count=1 ./...
test-race:
	go test -race ./...
docker-up:
	docker compose up --build -d
docker-down:
	docker compose down
seed:
	go run ./cmd/seed
lint:
	go vet ./...
verify: test test-race lint
benchmark:
	go test ./internal/pricing -run '^$$' -bench . -benchmem
