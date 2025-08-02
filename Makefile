# Makefile for abaper documentation generation

.PHONY: docs docs-man docs-markdown docs-yaml docs-rest docs-all docs-hugo clean-docs help

# Default target
help:
	@echo "Available targets:"
	@echo "  docs-man      - Generate man pages"
	@echo "  docs-markdown - Generate markdown documentation"
	@echo "  docs-yaml     - Generate YAML documentation"
	@echo "  docs-rest     - Generate reStructuredText documentation"
	@echo "  docs-hugo     - Generate Hugo-compatible documentation"
	@echo "  docs-all      - Generate all documentation formats"
	@echo "  docs          - Alias for docs-all"
	@echo "  clean-docs    - Remove generated documentation"
	@echo "  help          - Show this help message"

# Build the documentation generator
cmd/gendocs: cmd/gendocs.go
	@echo "Building documentation generator..."
	@go build -o cmd/gendocs cmd/gendocs.go

# Generate man pages
docs-man: cmd/gendocs
	@echo "Generating man pages..."
	@./cmd/gendocs man
	@echo "Man pages generated in docs/man/"

# Generate markdown documentation
docs-markdown: cmd/gendocs
	@echo "Generating markdown documentation..."
	@./cmd/gendocs markdown
	@echo "Markdown documentation generated in docs/markdown/"

# Generate YAML documentation
docs-yaml: cmd/gendocs
	@echo "Generating YAML documentation..."
	@./cmd/gendocs yaml
	@echo "YAML documentation generated in docs/yaml/"

# Generate reStructuredText documentation
docs-rest: cmd/gendocs
	@echo "Generating reStructuredText documentation..."
	@./cmd/gendocs rest
	@echo "reStructuredText documentation generated in docs/rest/"

# Generate Hugo-compatible documentation
docs-hugo: cmd/gendocs
	@echo "Generating Hugo-compatible documentation..."
	@chmod +x scripts/generate-hugo-docs.sh
	@./scripts/generate-hugo-docs.sh

# Generate all documentation formats
docs-all: cmd/gendocs
	@echo "Generating all documentation formats..."
	@./cmd/gendocs all
	@$(MAKE) docs-hugo

# Alias for docs-all
docs: docs-all

# Clean generated documentation
clean-docs:
	@echo "Cleaning generated documentation..."
	@rm -rf docs/
	@rm -rf hugo-docs/
	@rm -f cmd/gendocs
	@echo "Documentation cleaned."

# Install man pages (requires sudo)
install-man: docs-man
	@echo "Installing man pages..."
	@sudo cp docs/man/*.1 /usr/local/share/man/man1/
	@sudo mandb
	@echo "Man pages installed. Try: man abaper"

# Uninstall man pages (requires sudo)
uninstall-man:
	@echo "Uninstalling man pages..."
	@sudo rm -f /usr/local/share/man/man1/abaper*.1
	@sudo mandb
	@echo "Man pages uninstalled."
