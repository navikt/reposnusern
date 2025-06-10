ARTIFACT_NAME := reposnusern

build:
	@go build -o bin/${ARTIFACT_NAME}/${ARTIFACT_NAME} cmd/${ARTIFACT_NAME}/main.go 

run:
	@go run cmd/${ARTIFACT_NAME}/main.go 

go-test:
	@go test -v $(shell go list ./... | grep -v /test/)
	