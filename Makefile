.PHONY: build clean

build:
	go build -ldflags "-s -w -X main.version=dev" -o dk ./cmd/dk

clean:
	rm -f dk
