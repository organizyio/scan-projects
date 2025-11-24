.PHONY: build install clean

build:
	./build.sh

install:
	# installs a native binary as "scan-projects" (Linux/macOS only)
	GOOS=$(shell uname | tr '[:upper:]' '[:lower:]') GOARCH=amd64 go build -o scan-projects ./cmd/scan-projects
	@echo "Run ./scan-projects --root . --output projects.json"

clean:
	rm -rf dist scan-projects
