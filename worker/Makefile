.PHONY: help generate test test-metadata clean build check-naming

# 默认目标
help:
	@echo "Available targets:"
	@echo "  make generate       - Generate workflow contracts from code-first definitions"
	@echo "  make check-naming   - Check naming conventions"
	@echo "  make test           - Run all tests"
	@echo "  make test-metadata  - Run contract consistency tests"
	@echo "  make clean          - Clean generated files"
	@echo "  make build          - Build the worker binary"

# 生成代码
generate:
	@echo "Generating workflow contracts from code-first definitions..."
	@for workflow_dir in internal/workflow/*/; do \
		if [ -f "$${workflow_dir}contract_definition.go" ]; then \
			echo "Generating for $${workflow_dir}..."; \
			go generate "./$${workflow_dir}"; \
		fi \
	done
	@echo "✓ Workflow contract generation completed"

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
	@go test -v ./internal/workflow -run 'TestWorkflowVersionConsistency|TestCodeFirstToolchain_MigratedFromTemplateGenerators'
	@go test -v ./internal/workflow/subdomain_discovery -run TestCodeFirstMigration_NoLegacyTemplateArtifacts
	@echo "✓ Contract consistency verified"

# 清理生成的文件
clean:
	@echo "Cleaning generated files..."
	@# Remove worker binary
	@rm -f worker bin/worker
	@# Clean generated workflow contract artifacts
	@rm -f internal/workflow/subdomain_discovery/schema_generated.json
	@rm -f ../server/internal/engineschema/subdomain_discovery-v1-1.0.0.schema.json
	@rm -f ../docs/config-reference/subdomain_discovery.md
	@echo "✓ Cleaned"

# 构建
build:
	@echo "Building worker..."
	go build -o bin/worker ./cmd/worker
	@echo "✓ Build completed"

# 生成并测试（推荐的工作流）
gen-test: generate test-metadata
	@echo "✓ Generation and testing completed successfully"
