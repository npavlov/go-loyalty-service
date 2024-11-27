# Define Go command, which can be overridden
GO ?= go


# Default target: formats code, runs the linter, and builds both agent and server binaries
.PHONY: all
all: fmt lint build

# ----------- Build Commands -----------
# Build the server binary from Go source files in cmd/server directory
.PHONY: build
build:
	$(GO) build -gcflags="all=-N -l" -o bin/server ${CURDIR}/cmd/gophermart/*.go

# ----------- Test Commands -----------
# Run all tests and generate a coverage profile (coverage.out)
.PHONY: test
test:
	$(GO) test ./... -race -coverprofile=coverage.out -covermode=atomic

# View the test coverage report in HTML format
.PHONY: check-coverage
check-coverage:
	$(GO) tool cover -html=coverage.out

# ----------- Clean Command -----------
# Clean the bin directory by removing all generated binaries
.PHONY: clean
clean:
	rm -rf bin/

# ----------- Run Commands -----------
# Run the server directly from Go source files in cmd/server directory
.PHONY: run
run:
	$(GO) run ${CURDIR}/cmd/gophermart/*.go

# ----------- Lint and Format Commands -----------
# Run the linter (golangci-lint) on all Go files in the project
.PHONY: lint
lint:
	golangci-lint run ./...

# Run the linter and automatically fix issues
.PHONY: lint-fix
lint-fix:
	golangci-lint run ./... --fix

# Format all Go files in the project using the built-in Go formatting tool
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Format all Go files in the project using gofumpt for strict formatting rules
.PHONY: gofumpt
gofumpt:
	gofumpt -l -w .

# ----------- Dependency Management -----------
# Update all Go module dependencies
.PHONY: deps
deps:
	$(GO) get -u ./...

# ----------- Database Migration Commands -----------
# Create a new migration using Atlas
.PHONY: atlas-migration
atlas-migration:
	atlas migrate diff $(MIGRATION_NAME) --env dev

# Apply migrations using Goose
.PHONY: goose-up
goose-up:
	goose -dir migrations postgres "$(DATABASE_DSN)" up

# Rollback migrations using Goose
.PHONY: goose-down
goose-down:
	goose -dir migrations postgres "$(DATABASE_DSN)" down
