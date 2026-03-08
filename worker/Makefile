.PHONY: help generate workflow-contracts-gen-all workflow-contracts-gen-workflow workflow-contracts-ci-check test test-metadata clean build check-naming

# 默认目标
help:
	@echo "Available targets:"
	@echo "  make generate       - Generate workflow contracts (authoritative global entry)"
	@echo "  make workflow-contracts-gen-all      - Authoritative global generation entry"
	@echo "      optional vars: SERVER_SCHEMA_DIR, SERVER_MANIFEST_DIR, SERVER_PROFILE_DIR, DOCS_DIR, WORKER_SCHEMA_BASE_DIR, MIRROR_SCHEMA_DIR"
	@echo "  make workflow-contracts-gen-workflow - Incremental generation for one workflow (WORKFLOW=<name>)"
	@echo "  make workflow-contracts-ci-check      - CI gate: regenerate and verify no diffs"
	@echo "  make check-naming   - Check naming conventions"
	@echo "  make test           - Run all tests"
	@echo "  make test-metadata  - Run contract consistency tests"
	@echo "  make clean          - Clean generated files"
	@echo "  make build          - Build the worker binary"

# 权威全局生成入口
workflow-contracts-gen-all:
	@SERVER_SCHEMA_DIR="$(SERVER_SCHEMA_DIR)" \
		SERVER_MANIFEST_DIR="$(SERVER_MANIFEST_DIR)" \
		DOCS_DIR="$(DOCS_DIR)" \
		WORKER_SCHEMA_BASE_DIR="$(WORKER_SCHEMA_BASE_DIR)" \
		MIRROR_SCHEMA_DIR="$(MIRROR_SCHEMA_DIR)" \
		bash ./scripts/gen-workflow-contracts.sh

# 对外统一入口（权威）
generate: workflow-contracts-gen-all

# 本地增量入口（按 workflow 直接生成）
workflow-contracts-gen-workflow:
	@if [ -z "$(WORKFLOW)" ]; then \
		echo "Usage: make workflow-contracts-gen-workflow WORKFLOW=<workflow_name>"; \
		exit 1; \
	fi
	@go run ./cmd/workflow-contract-gen \
		-workflow $(WORKFLOW) \
		-worker-schema-dir ./internal/workflow/$(WORKFLOW)/generated \
		-worker-manifest-dir ./internal/workflow/$(WORKFLOW)/generated \
		-server-schema-dir ../server/internal/workflow/schema \
		-server-manifest-dir ../server/internal/workflow/manifest \
		-server-profile-dir ../server/internal/workflow/profile/profiles \
		-docs-dir ../docs/config-reference \
		-typed-go-output ./internal/workflow/$(WORKFLOW)/config_typed_generated.go \
		-typed-go-package $(WORKFLOW)

# CI 门禁：确保生成产物是最新的
workflow-contracts-ci-check: workflow-contracts-gen-all
	@git diff --exit-code -- \
		./internal/workflow \
		../server/internal/workflow/schema \
		../server/internal/workflow/manifest \
		../server/internal/workflow/profile/profiles \
		../docs/config-reference

check-naming:
	@echo "Checking naming conventions..."
	@bash ./scripts/check-naming-conventions.sh

# 运行所有测试
test:
	@echo "Running naming checks..."
	@$(MAKE) check-naming
	@echo "Running all tests..."
	go test -v ./...

# 运行契约一致性测试
test-metadata:
	@echo "Running workflow contract consistency tests..."
	@go test -v ./internal/workflow -run 'TestWorkflowSchemaConsistency|TestWorkflowCatalogConsistencyBetweenWorkerAndServer|TestCodeFirstToolchain_MigratedFromTemplateGenerators|TestCodeFirstToolchain_HasAuthoritativeGlobalGenerationEntry|TestCodeFirstToolchain_MetadataAndCleanUseDynamicWorkflowDiscovery|TestCodeFirstToolchain_NoPerWorkflowContractAssetsBoilerplate|TestAllWorkflowsImportFileMatchesWorkflowDirectories'
	@for workflow_dir in internal/workflow/*/; do \
		if [ -f "$${workflow_dir}contract_definition.go" ]; then \
			pkg="./$${workflow_dir%/}"; \
			go test -v "$$pkg" -run 'TestCodeFirstMigration_NoLegacyTemplateArtifacts|TestContractDefinitionMatchesGeneratedSchemaAndDocs|TestStageMetadataDerivedFromContract|TestGeneratedTypedConfigMatchesContract'; \
		fi \
	done
	@echo "✓ Contract consistency verified"

# 清理生成的文件
clean:
	@echo "Cleaning generated files..."
	@# Remove worker binary
	@rm -f worker bin/worker
	@# Clean generated workflow contract artifacts
	@for workflow_dir in internal/workflow/*/; do \
		if [ -f "$${workflow_dir}contract_definition.go" ]; then \
			workflow_name="$$(basename "$${workflow_dir%/}")"; \
			rm -f "$${workflow_dir}generated/$${workflow_name}-"*.schema.json; \
			rm -f "$${workflow_dir}generated/$${workflow_name}.manifest.json"; \
			rm -f "$${workflow_dir}config_typed_generated.go"; \
			rm -f "../docs/config-reference/$${workflow_name}.md"; \
			rm -f ../server/internal/workflow/schema/$${workflow_name}-*.schema.json; \
			rm -f ../server/internal/workflow/manifest/$${workflow_name}.manifest.json; \
			rm -f ../server/internal/workflow/profile/profiles/$${workflow_name}.yaml; \
		fi \
	done
	@echo "✓ Cleaned"

# 构建
build:
	@echo "Building worker..."
	go build -o bin/worker ./cmd/worker
	@echo "✓ Build completed"

# 生成并测试（推荐的工作流）
gen-test: generate test-metadata
	@echo "✓ Generation and testing completed successfully"
