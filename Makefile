BINARY := claude-statusline
INSTALL_DIR := $(HOME)/.claude

.PHONY: build install test bench verify clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BINARY) .

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	chmod +x $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

test:
	go test -v -race -timeout=30s

bench:
	go test -bench=. -benchmem -count=3 -timeout=120s

verify:
	@TAG=$$(gh release view --json tagName -q .tagName) && \
	OS=$$(go env GOOS) && ARCH=$$(go env GOARCH) && \
	EXT=tar.gz && [ "$$OS" = "windows" ] && EXT=zip; \
	ARCHIVE="claude-statusline_$${TAG#v}_$${OS}_$${ARCH}.$${EXT}" && \
	TMP=$$(mktemp -d) && trap 'rm -rf $$TMP' EXIT && \
	echo "Verifying $$TAG for $${OS}/$${ARCH}..." && \
	gh release download "$$TAG" --pattern "$$ARCHIVE" --dir "$$TMP" && \
	tar xzf "$$TMP/$$ARCHIVE" -C "$$TMP" && \
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$$TMP/local-build" . && \
	RELEASE=$$(shasum -a 256 "$$TMP/claude-statusline" | awk '{print $$1}') && \
	LOCAL=$$(shasum -a 256 "$$TMP/local-build" | awk '{print $$1}') && \
	echo "Release: $$RELEASE" && \
	echo "Local:   $$LOCAL" && \
	if [ "$$RELEASE" = "$$LOCAL" ]; then \
		echo "\033[32m✓ MATCH — release binary is byte-identical to local build\033[0m"; \
	else \
		echo "\033[31m✗ MISMATCH — likely different Go version (CI uses 1.22)\033[0m"; \
		exit 1; \
	fi

clean:
	rm -f $(BINARY)
