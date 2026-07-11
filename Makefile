.PHONY: build run test vet clean init help

BINARY := dataset-builder
CMD := ./cmd/dataset-builder

build:
	go build -o $(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf data/ datasets/ repos/ config.yaml

init: build
	./$(BINARY) init

help:
	go run $(CMD) --help
