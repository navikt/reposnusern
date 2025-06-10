ARTIFACT_NAME := reposnusern

build:
	@go build -o bin/${ARTIFACT_NAME}/${ARTIFACT_NAME} cmd/${ARTIFACT_NAME}/main.go 

run:
	@go run cmd/${ARTIFACT_NAME}/main.go 

go-test:
	@go test -v $(shell go list ./... | grep -v /test/)

COVER_OUT = cover.out
COVER_FILTERED = cover.filtered.out

# Filer og mapper du vil ekskludere
EXCLUDE_FILES = \
    cmd/$(ARTIFACT_NAME)/main.go \
    internal/storage/ \
    internal/models/

EXCLUDE_GREP := $(foreach f,$(EXCLUDE_FILES),| grep -v $(f))

go-test-with-cover:
	@go test -coverprofile=$(COVER_OUT) -v $(shell go list ./... | grep -v /test/)
	@cat $(COVER_OUT) $(EXCLUDE_GREP) > $(COVER_FILTERED)
	@go tool cover -html=$(COVER_FILTERED) -o cover.html
	@open cover.html || xdg-open cover.html || echo "Åpne cover.html manuelt"

generate-mocks:
	@mockery --all --with-expecter --keeptree
