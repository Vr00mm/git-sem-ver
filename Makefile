.PHONY: build test lint lint-fix fmt clean

BINARY := git-sem-ver
CMD     := ./cmd/git-sem-ver

build:
	go build -o $(BINARY) $(CMD)

test:
	go test -race -shuffle=on ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

fmt:
	golangci-lint fmt ./...

clean:
	rm -f $(BINARY)
