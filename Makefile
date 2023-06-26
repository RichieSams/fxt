all: test


test:
	go test -cover ./...

release:
	goreleaser release --clean

release_test:
	goreleaser release --snapshot --clean --skip-publish

vendor:
	go mod tidy
	go mod vendor
