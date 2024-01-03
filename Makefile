.PHONY: benchmarks check tests clean

tests: check

check:
	@go test ./... -benchmem -race

clean:
	@rm *.test */*.test

benchmarks:
	@(cd bench; go test -count 1 -bench . -run XXX)

lint:
	@golangci-lint run
