build:
	CGO_ENABLED=0 go build -o bin/acorn -ldflags "-s -w" .

download-latest-ui:
	curl -sL https://cdn.acrn.io/ui/latest.tar.gz | tar xzf - -C pkg/ui/server/static/dashboard --strip-components=1

download-ui:
	curl -sL https://cdn.acrn.io/ui/latest.tar.gz | tar xzf - -C pkg/ui/server/static/dashboard --strip-components=1

generate:
	go generate

image:
	docker build .

validate:
	golangci-lint run

validate-ci:
	go generate
	go mod tidy
	go run tools/gendocs/main.go
	if [ -n "$$(git status --porcelain)" ]; then \
		git status --porcelain; \
		echo "Encountered dirty repo!"; \
		exit 1 \
	;fi

test:
	go test ./...

goreleaser:
	goreleaser build --snapshot --single-target --rm-dist

setup-ci-env:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.49.0

# This will initialize the node_modules needed to run the docs dev server. Run this before running serve-docs
init-docs:
	docker run --rm --workdir=/docs -v $${PWD}/docs:/docs node:18-buster yarn install

# Ensure docs build without errors. Makes sure generated docs are in-sync with CLI.
validate-docs:
	docker run --rm --workdir=/docs -v $${PWD}/docs:/docs node:18-buster yarn build
	go run tools/gendocs/main.go
	if [ -n "$$(git status --porcelain --untracked-files=no)" ]; then \
		git status --porcelain --untracked-files=no; \
		echo "Encountered dirty repo!"; \
		exit 1 \
	;fi

# Launch development server for the docs site
serve-docs:
	acorn run -i ./docs

gen-docs:
	go run tools/gendocs/main.go
