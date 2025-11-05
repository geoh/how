.PHONY: build install clean test

build:
	go build -o how ./cmd/how

install:
	go install ./cmd/how

clean:
	rm -f how

test:
	go test -v ./...
