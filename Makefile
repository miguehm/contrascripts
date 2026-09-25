BIN     := bin
DIST    := dist
WASMENV := GOOS=js GOARCH=wasm
WASMFLAGS := -trimpath -ldflags="-s -w"
GOVERSION := $(shell go env GOVERSION)
# wasm_exec.js is not a build artefact: it is the mandatory Go<->JS runtime shim,
# and it must be the one shipped with the toolchain that compiled the .wasm.
# A mismatched copy makes go.run() fail at init, so it is never committed and
# always copied straight out of GOROOT.
WASM_EXEC := $(shell go env GOROOT)/lib/wasm/wasm_exec.js

.PHONY: all build test vet fmt check wasm dist manifest package sizes smoke clean

all: check dist

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

# dist is what a web project actually consumes: the modules plus the glue they
# cannot run without. `make wasm` alone yields two .wasm files that no browser
# can instantiate.
dist: wasm
	cp $(WASM_EXEC) $(DIST)/wasm_exec.js
	cp web/fountain.mjs $(DIST)/fountain.mjs
	@$(MAKE) --no-print-directory manifest
	@printf 'dist/ is ready: %s modules, glue from %s\n' "$$(ls $(DIST)/*.wasm | wc -l)" "$(GOVERSION)"

# manifest records which Go produced the modules. wasm_exec.js is pinned to that
# same toolchain, so a mismatch between the two is detectable without rebuilding.
manifest:
	@{ \
		printf '{\n'; \
		printf '  "goVersion": "%s",\n' "$(GOVERSION)"; \
		printf '  "wasmExec": "wasm_exec.js",\n'; \
		printf '  "modules": {\n'; \
		n=0; \
		for f in $(DIST)/*.wasm; do \
			n=$$((n + 1)); \
			[ $$n -gt 1 ] && printf ',\n'; \
			printf '    "%s": { "bytes": %s, "sha256": "%s" }' \
				"$$(basename $$f)" \
				"$$(wc -c < $$f)" \
				"$$(sha256sum $$f | cut -d' ' -f1)"; \
		done; \
		printf '\n  }\n}\n'; \
	} > $(DIST)/manifest.json

# smoke runs the modules in a real browser. It is the only check that catches
# the failure modes the Go compiler cannot see: a wasm_exec.js that does not
# match the toolchain, and the two modules fighting over globalThis.
#
# Two suites, because they answer different questions. The first drives the
# package on its own; the second runs a Vite app against it in both dev and
# build, where the wrapper is compiled into a bundle and has to be told where
# the .wasm files live.
smoke: dist
	cd web && npm test
	cd web/vite && npx playwright test

# package stages dist/ into another project's public folder, which is how a
# frontend consumes the bundle without Go installed over there: its build calls
#   make -C ../fountain package DIR=public/fountain
# The shim stays a plain file in that folder on purpose. Serving it from
# node_modules instead would need a copy plugin per bundler.
package: dist
	@test -n "$(DIR)" || { echo "usage: make package DIR=<dest>"; exit 1; }
	@mkdir -p $(DIR)
	cp $(DIST)/* $(DIR)/

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
