.PHONY: dev test build frontend-build backend-test demo docker-up docker-down docker-config milvus-up milvus-down

dev:
	go run ./cmd/server

test: backend-test frontend-build

build: frontend-build
	go build ./cmd/server

frontend-build:
	cd web/frontend && npm run build

backend-test:
	go test ./...

demo:
	go run ./cmd/server

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-config:
	docker compose config

milvus-up:
	docker compose --profile milvus up -d

milvus-down:
	docker compose --profile milvus down
