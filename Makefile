build:
	CGO_ENABLED=0 go build -o bin/acorn -ldflags "-s -w" .

generate:
	go generate

image:
	docker build .

validate:
	golangci-lint --timeout 5m run

test:
	go test ./...

goreleaser:
	goreleaser build --snapshot --single-target --rm-dist

setup-ci-env:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.46.2

serve-docs:
	docker run -it --rm --workdir=/docs -p 3000:3000 -v $${PWD}/docs:/docs node:18-buster yarn start --host=0.0.0.0
