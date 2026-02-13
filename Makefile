BINS=./cmd/ingest-api ./cmd/fusefs ./cmd/scanner-agent

build:
	go build $(BINS)

test:
	go test ./...

lint:
	go vet ./...

up:
	docker compose up -d

down:
	docker compose down

migrate:
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate \
		-path=/migrations -database "postgres://app:app@localhost:5432/virtualfs?sslmode=disable" up

seed:
	psql "postgres://app:app@localhost:5432/virtualfs?sslmode=disable" -c "select now();"
