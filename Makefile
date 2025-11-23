container_runtime := $(shell which docker || which podman)

$(info using ${container_runtime})

help: ## показать справку по командам
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

tools: ## установить необходимые инструменты разработки
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Tools installed"

up: ## запустить сервисы через Docker Compose
	${container_runtime} compose up --build -d

down: ## остановить сервисы
	${container_runtime} compose down

clean: ## остановить сервисы и удалить volumes
	${container_runtime} compose down -v

test: ## запустить полный цикл тестирования 
	make clean
	make up
	@echo "wait cluster to start" && sleep 10
	make test-integration
	make clean
	@echo "test finished"

lint: ## запустить линтер
	cd reviewer && golangci-lint run ./...

test-integration: ## запустить интеграционные тесты (требует запущенную БД)
	@echo "Starting integration tests..."
	@echo "Make sure PostgreSQL is running and accessible"
	cd reviewer && go test -v ./internal/adapters/rest/... -run "Integration|E2E"

build: ## собрать приложение
	cd reviewer && go build -o pr-reviewer ./main.go

logs: ## показать логи всех сервисов
	${container_runtime} compose logs -f

postgres-logs: ## показать логи postgres
	${container_runtime} compose logs -f postgres

app-logs: ## показать логи приложения
	${container_runtime} compose logs -f app
