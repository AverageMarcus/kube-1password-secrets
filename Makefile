.DEFAULT_GOAL := default

IMAGE ?= rg.fr-par.scw.cloud/averagemarcus/kube-1password-secrets:latest

.PHONY: test # Run all tests, linting and format checks
test: lint check-format run-tests

.PHONY: lint # Perform lint checks against code
lint:
	@go vet && golint -set_exit_status ./...

.PHONY: check-format # Checks code formatting and returns a non-zero exit code if formatting errors found
check-format:
	@gofmt -e -l .

.PHONY: format # Performs automatic format fixes on all code
format:
	@gofmt -s -w .

.PHONY: run-tests # Runs all tests
run-tests:
	@go test

.PHONY: fetch-deps # Fetch all project dependencies
fetch-deps:
	@go mod tidy

.PHONY: build # Build the project
build: lint check-format fetch-deps
	@go build -o kube-1password-secrets main.go

.PHONY: docker-build # Build the docker image
docker-build:
	@docker build -t $(IMAGE) .

.PHONY: docker-publish # Push the docker image to the remote registry
docker-publish:
	@docker push $(IMAGE)

.PHONY: run # Run the application
run:
	@go run main.go

.PHONY: ci # Perform CI specific tasks to perform on a pull request
ci:
	@echo "⚠️ 'ci' unimplemented"

.PHONY: release # Release the latest version of the application
release:
	@cd manifests && kustomize edit set image kube-1password-secrets=$(IMAGE) && kustomize build | kubectl apply -f -

.PHONY: help # Show this list of commands
help:
	@echo "kube-1password-secrets"
	@echo "Usage: make [target]"
	@echo ""
	@echo "target	description" | expand -t20
	@echo "-----------------------------------"
	@grep '^.PHONY: .* #' Makefile | sed 's/\.PHONY: \(.*\) # \(.*\)/\1	\2/' | expand -t20

default: test build
