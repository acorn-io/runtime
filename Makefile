build:
	CGO_ENABLED=0 go build -o bin/acorn -ldflags "-s -w" .

generate:
	go generate

mocks:
	go run github.com/golang/mock/mockgen --build_flags=--mod=mod -destination=./pkg/mocks/mock_client.go -package=mocks github.com/acorn-io/acorn/pkg/client Client,ProjectClientFactory

image:
	docker build .

setup-ci-image:
	docker build -t acorn:v-ci .
	docker save acorn:v-ci | docker exec -i $$(docker ps | grep k3s | awk '{print $$1}') ctr --address /run/k3s/containerd/containerd.sock images import -

validate:
	golangci-lint run

validate-ci: setup-ci-env
	go generate
	go mod tidy
	go run tools/gendocs/main.go
	if [ -n "$$(git status --porcelain)" ]; then \
		git status --porcelain; \
		echo "Encountered dirty repo!"; \
		git diff; \
		exit 1 \
	;fi

test:
	go test $(TEST_FLAGS) ./...

goreleaser:
	goreleaser build --snapshot --single-target --rm-dist

setup-ci-env:
	if ! command -v golangci-lint &> /dev/null; then \
  		echo "Could not find golangci-lint, installing."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.51.1; \
	fi


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
		git diff; \
		exit 1 \
	;fi

# Launch development server for the docs site
serve-docs:
	acorn run -i ./docs

gen-docs:
	go run tools/gendocs/main.go

#cut a new version for release with items in docs/docs
gen-docs-release:
	if [ -z ${version} ]; then \
  			echo "version not set (version=x.x)"; \
    		exit 1 \
    	;fi
	if [ -z ${prev-version} ]; then \
  			echo "prev-version not set (prev-version=x.x)"; \
    		exit 1 \
    	;fi
	make gen-docs
	docker run --rm --workdir=/docs -v $${PWD}/docs:/docs node:18-buster yarn docusaurus docs:version ${version}
	awk '/versions/&& ++c == 1 {print;print "\t\t\t\"${prev-version}\": {label: \"${prev-version}\", banner: \"none\", path: \"${prev-version}\"},";next}1' ./docs/docusaurus.config.js > tmp.config.js && mv tmp.config.js ./docs/docusaurus.config.js

#depreceate a specific docs version (will still be included within docs dropdown)
deprecate-docs-version:
	if [ -z ${version} ]; then \
  			echo "version not set (version=x.x)"; \
    		exit 1 \
    	;fi
	echo "deprecating ${version} from documentation"
	grep -v '"${version}": {label: "${version}", banner: "none", path: "${version}"},' ./docs/docusaurus.config.js  > tmp.config.js && mv tmp.config.js ./docs/docusaurus.config.js

#completly remove doc version from docs site
remove-docs-version:
	if [ -z ${version} ]; then \
  			echo "version not set (version=x.x)"; \
    		exit 1 \
    	;fi
	echo "removing ${version} from documentation completely"
	-rm  "./docs/versioned_sidebars/version-${version}-sidebars.json"
	-rm  -r ./docs/versioned_docs/version-${version}
	jq 'del(.[] | select(. == "${version}"))' ./docs/versions.json > tmp.json && mv tmp.json ./docs/versions.json
	grep -v '"${version}": {label: "${version}", banner: "none", path: "${version}"},' ./docs/docusaurus.config.js  > tmp.config.js && mv tmp.config.js ./docs/docusaurus.config.js
