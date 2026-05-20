BINARY := claude-statusline
INSTALL_DIR := $(HOME)/.claude

.PHONY: build install clean

build:
	go build -trimpath -ldflags='-s -w' -o $(BINARY) .

install: build
	cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	chmod +x $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo 'Update statusLine.command in settings.json to: $(INSTALL_DIR)/$(BINARY)'

test:
	go test -v -timeout=30s

bench:
	go test -bench=. -benchmem -count=3 -timeout=120s

clean:
	rm -f $(BINARY)
