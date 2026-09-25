BIN     := bin
DIST    := dist
WASMENV := GOOS=js GOARCH=wasm
WASMFLAGS := -trimpath -ldflags="-s -w"

.PHONY: all build test vet fmt check wasm sizes clean

all: check wasm

# build compiles the CLI into bin/, keeping the module root free of binaries.
build:
	mkdir -p $(BIN)
	go build -o $(BIN)/fountain ./cmd/fountain

# check is what CI should run. The wasm targets need an explicit GOOS: the
# //go:build wasm tag makes `go vet ./...` on linux skip cmd/wasm-* entirely,
# so without these the WebAssembly glue would never be type-checked.
check: fmt vet test

fmt:
	@out="$$(gofmt -l cmd internal)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...
	$(WASMENV) go vet ./...

test:
	go test ./...

# wasm builds the browser modules. fountain-parser carries no PDF renderer and
# no fonts, so it is meant to load on first paint; fountain-pdf bundles gofpdf
# and the embedded Courier Prime faces and is loaded on demand.
wasm:
	mkdir -p $(DIST)
	$(WASMENV) go build $(WASMFLAGS) -o $(DIST)/fountain-parser.wasm ./cmd/wasm-parser
	$(WASMENV) go build $(WASMFLAGS) -o $(DIST)/fountain-pdf.wasm ./cmd/wasm-pdf

# sizes reports raw, gzip and brotli footprints, which is the number that
# decides whether the split into two modules is still worth keeping.
sizes: wasm
	@printf '%-20s %10s %10s %10s\n' module raw gzip brotli
	@for f in $(DIST)/*.wasm; do \
		br=$$(command -v brotli >/dev/null 2>&1 && brotli -q 11 -c $$f | wc -c || echo "n/a"); \
		printf '%-20s %10s %10s %10s\n' \
			"$$(basename $$f .wasm)" \
			"$$(stat -c%s $$f)" \
			"$$(gzip -9 -c $$f | wc -c)" \
			"$$br"; \
	done

clean:
	rm -rf $(BIN) $(DIST) fountain
