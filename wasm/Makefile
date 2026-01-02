.PHONY: build clean setup serve

# Build the WASM binary
build:
	GOOS=js GOARCH=wasm go build -o example/headlessterm.wasm .

# Copy wasm_exec.js from Go installation
setup:
	@if [ -f "$$(go env GOROOT)/lib/wasm/wasm_exec.js" ]; then \
		cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" ./example/; \
	else \
		cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" ./example/; \
	fi

# Build and setup
all: setup build

# Clean build artifacts
clean:
	rm -f example/headlessterm.wasm
	rm -f example/wasm_exec.js

# Serve the example directory (requires Python 3)
serve: all
	@echo "Starting server at http://localhost:8080"
	cd example && python3 -m http.server 8080
