default: install

build-deps:
	GO111MODULE=off go get -u golang.org/x/lint/golint
	GO111MODULE=off go get -u oss.indeed.com/go/go-groups

fmt:
	go-groups -w .
	gofmt -s -w .

deps:
	go mod download
	go mod verify

test:
	go vet ./...
	golint -set_exit_status ./...
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

run:
	go run ./cmd/actuary/main.go
