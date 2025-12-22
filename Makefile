.PHONY: phony
phony-goal: ; @echo $@

install: fetch-dependencies
	go install github.com/incu6us/goimports-reviser/v3@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/vladopajic/go-test-coverage/v2@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

fetch-dependencies:
	go mod download

imports:
	go mod tidy
	goimports-reviser -rm-unused -set-alias -format -recursive .

format:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run --fix ./...

test:
	go test -covermode atomic -coverprofile .reports/testcoverage.out ./...

coverage: test
	go tool cover -html=.reports/testcoverage.out -o .reports/testcoverage.html
	go-test-coverage --config=.testcoverage.yml

check:
	govulncheck ./...

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