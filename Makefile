.DEFAULT_GOAL := help

.PHONY: help proto proto-check gen-release-manifest verify-release-contract

MANIFEST_OUTPUT ?= release.manifest.yaml
MANIFEST_VERIFY_ENV ?=

help:
	@echo "Available targets:"
	@echo "  make proto       - Generate Go code from proto definitions"
	@echo "  make proto-check - Verify generated proto files are up to date"
	@echo "  make gen-release-manifest - Generate release manifest from env vars"
	@echo "  make verify-release-contract - Validate release.manifest.yaml and docker/.env contract"

proto:
	bash proto/scripts/gen-go.sh

proto-check:
	bash proto/scripts/check-generated.sh

gen-release-manifest:
	bash scripts/ci/generate-release-manifest.sh --output "$(MANIFEST_OUTPUT)" --verify-contract $(if $(MANIFEST_VERIFY_ENV),--verify-env "$(MANIFEST_VERIFY_ENV)",)

verify-release-contract:
	bash scripts/ci/verify-release-contract.sh
