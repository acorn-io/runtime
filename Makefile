build:
	CGO_ENABLED=0 go build -o bin/acorn -ldflags "-s -w" .

generate:
	go generate

image:
	docker build .

validate:
	golangci-lint --timeout 5m run

test:
	go test -v -test.v ./...

goreleaser:
	goreleaser build --snapshot --single-target --rm-dist

setup-ci-env:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.46.2
