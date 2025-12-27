.PHONY: phony
phony-goal: ; @echo $@

install:
	cd tools && go get -tool github.com/incu6us/goimports-reviser/v3@latest
	cd tools && go get -tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	cd tools && go get -tool go.uber.org/mock/mockgen@latest
	cd tools && go get -tool github.com/vladopajic/go-test-coverage/v2@latest
	cd tools && go get -tool golang.org/x/vuln/cmd/govulncheck@latest

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