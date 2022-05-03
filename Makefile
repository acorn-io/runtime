build:
	go build -o bin/acorn .

test:
	go test -v -test.v ./...
