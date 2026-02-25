BIN       := bin
BINARY    := mastermind
CMD       := ./cmd/mastermind
CONFIGS   := -level configs/default-level.yaml -palette configs/default-palette.yaml

.PHONY: help build run clean vet lint test cross

help: ## Show available targets
	@grep -E '^[a-z]+:.*##' $(MAKEFILE_LIST) | awk -F ':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: vet test ## Vet, test, and compile
	@mkdir -p $(BIN)
	go build -o $(BIN)/$(BINARY) $(CMD)

run: build ## Build and run with default configs
	./$(BIN)/$(BINARY) $(CONFIGS)

test: ## Run all tests
	go test ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Error: golangci-lint is not installed. See https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

cross: ## Cross-compile for macOS and Windows
	@mkdir -p $(BIN)
	GOOS=darwin  GOARCH=arm64 go build -o $(BIN)/$(BINARY)-macos       $(CMD)
	GOOS=darwin  GOARCH=amd64 go build -o $(BIN)/$(BINARY)-macos-intel $(CMD)
	GOOS=windows GOARCH=amd64 go build -o $(BIN)/$(BINARY).exe         $(CMD)

clean: ## Remove build artifacts
	rm -rf $(BIN)
	rm -f $(BINARY) $(BINARY)-* $(BINARY).exe
