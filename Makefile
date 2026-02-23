.PHONY: test lint generate-domain

# Runs all tests including integration
test:
	go test -v -race ./...

# Runs the linter
lint:
	golangci-lint run --timeout=5m ./...

# AI Hook: We leave this command for the AI to update the domain model 
# based on the contract file.
generate-domain:
	@echo "AI Agent: Read contracts/message.json and update internal/domain/message.go to match."
