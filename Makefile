BINARY_NAME=blockblox
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")

.PHONY: build clean build-macos-binaries package-macos-binaries generate-macos-checksums update-homebrew-formula release

build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist

build-macos-binaries:
	@echo "Building macOS binaries..."
	@mkdir -p dist
	@echo "Building darwin-amd64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-darwin-amd64
	@echo "Building darwin-arm64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-darwin-arm64
	@echo "Done"

package-macos-binaries: build-macos-binaries
	@echo "Packaging macOS binaries..."
	@cd dist && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd dist && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "Done"

generate-macos-checksums: package-macos-binaries
	@echo "Generating checksums..."
	@cd dist && shasum -a 256 $(BINARY_NAME)-$(VERSION)-darwin-*.tar.gz > checksums.txt
	@cat dist/checksums.txt

update-homebrew-formula: generate-macos-checksums
	@echo "Updating Homebrew formula..."
	@AMD64_SHA=$$(cd dist && shasum -a 256 $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz | cut -d' ' -f1); \
	ARM64_SHA=$$(cd dist && shasum -a 256 $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz | cut -d' ' -f1); \
	CLEAN_VERSION=$$(echo "$(VERSION)" | sed 's/^v//'); \
	sed -i '' "s/version \".*\"/version \"$$CLEAN_VERSION\"/" Formula/blockblox.rb; \
	sed -i '' "s|download/v.*/blockblox-v.*-darwin-amd64.tar.gz|download/$(VERSION)/blockblox-$(VERSION)-darwin-amd64.tar.gz|" Formula/blockblox.rb; \
	sed -i '' "s|download/v.*/blockblox-v.*-darwin-arm64.tar.gz|download/$(VERSION)/blockblox-$(VERSION)-darwin-arm64.tar.gz|" Formula/blockblox.rb; \
	sed -i '' "/darwin-amd64.tar.gz/,/sha256/{s/sha256 \".*\"/sha256 \"$$AMD64_SHA\"/;}" Formula/blockblox.rb; \
	sed -i '' "/darwin-arm64.tar.gz/,/sha256/{s/sha256 \".*\"/sha256 \"$$ARM64_SHA\"/;}" Formula/blockblox.rb
	@echo "Done"

release: update-homebrew-formula
	@echo "Release $(VERSION) ready"
	@echo "Binaries: dist/"
	@echo "Formula: Formula/blockblox.rb"
	@echo ""
	@echo "Next steps:"
	@echo "  1. git add -A && git commit -m 'Release $(VERSION)'"
	@echo "  2. git tag $(VERSION)"
	@echo "  3. git push && git push --tags"
	@echo "  4. gh release create $(VERSION) dist/*.tar.gz"
