

test:
	go test -cover ./...
 
 vendor:
	go mod tidy
	go mod vendor
