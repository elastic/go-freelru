.PHONY: bench check tests clean

tests: check

check:
	@go test ./... -benchmem -race

clean:
	@rm *.test */*.test

bench:
	@(cd bench; go test -bench=. -run XXX)

GOLANGCI_LINT_VERSION = "v2.1.6"
lint:
	docker run --rm -t -v $$(pwd):/app -w /app \
golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run
