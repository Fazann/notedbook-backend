# Run db migration
.PHONY: migrate
migrate:
	@echo "Start migration"
	go run cmd/migration/main.go


# Run seeding data
.PHONY: seed
seed:
	@echo "Start seeding data"
	go run cmd/seed/main.go


# Run project with hot-reload
.PHONY: run
run:
	@echo "Start run application"
	air


# Build docker image
DOCKER_IMAGE=notebook-api:latest
.PHONY: build
build:
	@echo "Start build docker image"
	docker build --platform linux/amd64 -t $(DOCKER_IMAGE) .


# Run util package tests only
.PHONY: test-util
test-util:
	@echo "Run util package tests only"
	@if [ ! -f .env ] && [ -f .env.example ]; then cp .env.example .env; fi
	@go test ./tests/util -v


.PHONY: locale
locale:
	@echo "Run generate translate enum"
	go run cmd/locale/main.go

# Generate Swagger docs (docs/) from the handler annotations
SWAG_VERSION=v1.16.6
.PHONY: swagger
swagger:
	@echo "Generate swagger docs"
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) fmt -d ./cmd/app,./internal
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init -g cmd/app/main.go -o docs --outputTypes go,json,yaml
