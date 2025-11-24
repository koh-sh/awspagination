.PHONY: test fmt cov tidy lint lint-fix build modernize modernize-fix ci tool-install test-vendor

COVFILE = coverage.out
COVHTML = cover.html
GITHUB_REPOSITORY = koh-sh/awspagination

# Generate vendor directories for test packages
test-vendor:
	cd testdata/src/test && go mod tidy && go mod vendor
	cd testdata/src/testskip && go mod tidy && go mod vendor

test: test-vendor
	go test ./... -json | go tool tparse -all

fmt:
	go tool gofumpt -l -w .

cov: test-vendor
	go test -cover ./... -coverprofile=$(COVFILE)
	go tool cover -html=$(COVFILE) -o $(COVHTML)
	CI=1 GITHUB_REPOSITORY=$(GITHUB_REPOSITORY) octocov
	rm $(COVFILE)

tidy:
	go mod tidy -v

lint:
	go tool golangci-lint run

lint-fix:
	go tool golangci-lint run --fix

build:
	go build ./cmd/awspagination

ci: fmt modernize-fix lint-fix build cov

# Go Modernize
modernize:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -test ./...

modernize-fix:
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...

tool-install:
	go get -tool mvdan.cc/gofumpt
	go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	go get -tool github.com/mfridman/tparse
	brew install k1LoW/tap/octocov
