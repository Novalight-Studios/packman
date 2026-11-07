build:
	go build -o bin/packman .

run:
	go run . $(ARGS)

.PHONY: build run
