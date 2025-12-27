.PHONY: phony
phony-goal: ; @echo $@

verify-tools:
	@go tool golangci-lint --version >/dev/null 2>&1 || { echo >&2 "golangci-lint is not installed. Run 'make install'"; exit 1; }
	@go tool goimports-reviser -version >/dev/null 2>&1 || { echo >&2 "goimports-reviser is not installed. Run 'make install'"; exit 1; }
	@go tool go-test-coverage --version >/dev/null 2>&1 || { echo >&2 "go-test-coverage is not installed. Run 'make install'"; exit 1; }
	@go tool mockgen --version >/dev/null 2>&1 || { echo >&2 "go-test-coverage is not installed. Run 'make install'"; exit 1; }
	@go tool govulncheck --version >/dev/null 2>&1 || { echo >&2 "govulncheck is not installed. Run 'make install'"; exit 1; }
	@echo "All tools are installed."

install:
	cd tools && go get -tool github.com/incu6us/goimports-reviser/v3@latest
	cd tools && go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	cd tools && go get -tool go.uber.org/mock/mockgen@latest
	cd tools && go get -tool github.com/vladopajic/go-test-coverage/v2@latest
	cd tools && go get -tool golang.org/x/vuln/cmd/govulncheck@latest

imports: verify-tools
	go mod tidy
	go tool goimports-reviser -rm-unused -set-alias -format -recursive .

format:
	go fmt ./...

vet:
	go vet ./...

lint: verify-tools
	go tool golangci-lint run --fix ./...

test:
	go test -covermode atomic -coverprofile .reports/testcoverage.out ./...

coverage: verify-tools test
	go tool cover -html=.reports/testcoverage.out -o .reports/testcoverage.html
	go tool go-test-coverage --config=.testcoverage.yml

check: verify-tools
	go tool govulncheck ./...

validate: imports format vet lint coverage check

report:
	# make vet > .reports/vet.out 2>&1
	@$(MAKE) vet > .reports/vet.out 2>&1 || true

	# make lint > .reports/lint.out 2>&1
	@$(MAKE) lint > .reports/lint.out 2>&1 || true

	# make coverage > .reports/coverage.out 2>&1
	@$(MAKE) coverage > .reports/coverage.out 2>&1 || true

	# make check > .reports/check.out 2>&1
	@$(MAKE) check > .reports/check.out 2>&1 || true


##
update-dependencies:
	go get -u ./... && go get -t -u ./... && go mod tidy