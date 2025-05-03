all: help

lint:
	@echo "Running linter..."
	golangci-lint run ./

test:
	@echo "Running tests..."
	go test ./ -gcflags=all=-l --race -v -short -coverprofile=coverage.out

test-coverage:
	@echo "Generating test coverage report..."
	go test -coverprofile=coverage.out ./
	@echo "Test coverage profile generated: coverage.out"
	@echo "Use 'make view-coverage' to view the HTML report."

view-coverage: test-coverage
	@echo "Opening test coverage report in browser..."
	go tool cover -html=coverage.out

deps:
	@echo "Installing dependencies..."
	go mod tidy

clean:
	@echo "Cleaning up..."
	rm -rf coverage.out

help:
	@echo "--------------------------------------------------------------"
	@echo "Available targets:"
	@echo "  deps            - Install dependencies (run 'go mod tidy')"
	@echo "  lint            - Run linter (requires golangci-lint)"
	@echo "  test            - Run tests with race detection and coverage"
	@echo "  test-coverage   - Generate test coverage report"
	@echo "  view-coverage   - Open test coverage report in browser"
	@echo "  clean           - Remove generated files (coverage.out)"
	@echo "  help            - Show this help message"
	@echo "--------------------------------------------------------------"
	@echo "Credits: Muhammad Saleh Solahudin <https://github.com/ZihxS>"
	@echo "--------------------------------------------------------------"
