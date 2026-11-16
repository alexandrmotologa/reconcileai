.PHONY: all build test clean run lint bench

BINARY_NAME=bin/reconcileai.exe

all: build

build-frontend:
	cd web && npm install && npm run build

build: build-frontend
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/reconcileai

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./internal/auditor

generate-dossiers:
	go run ./cmd/reconcileai generate-dossier --dir ./examples/dossiers

run: build
	./$(BINARY_NAME) serve --port 7272

clean:
	rm -rf bin/ web/dist/
