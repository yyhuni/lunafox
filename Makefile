.DEFAULT_GOAL := help

.PHONY: help proto proto-check

help:
	@echo "Available targets:"
	@echo "  make proto       - Generate Go code from proto definitions"
	@echo "  make proto-check - Verify generated proto files are up to date"

proto:
	bash proto/scripts/gen-go.sh

proto-check:
	bash proto/scripts/check-generated.sh
