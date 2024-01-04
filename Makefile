.PHONY: bench check tests clean

tests: check

check:
	@go test ./... -benchmem -race

clean:
	@rm *.test */*.test

bench:
	@(cd bench; go test -bench=. -run XXX)

lint:
	@golangci-lint run
