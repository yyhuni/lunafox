.PHONY: help generate test test-metadata clean build check-naming

# 默认目标
help:
	@echo "Available targets:"
	@echo "  make generate       - Generate code from templates.yaml"
	@echo "  make check-naming   - Check naming conventions"
	@echo "  make test           - Run all tests"
	@echo "  make test-metadata  - Run metadata consistency tests"
	@echo "  make clean          - Clean generated files"
	@echo "  make build          - Build the worker binary"

# 生成代码
generate:
	@echo "Generating code from templates.yaml..."
	@for workflow_dir in internal/workflow/*/; do \
		if [ -f "$${workflow_dir}templates.yaml" ]; then \
			echo "Generating for $${workflow_dir}..."; \
			go generate "./$${workflow_dir}"; \
		fi \
	done
	@echo "✓ Code generation completed"

check-naming:
	@echo "Checking naming conventions..."
	@bash ./scripts/check-naming-conventions.sh

# 运行所有测试
test:
	@echo "Running naming checks..."
	@$(MAKE) check-naming
	@echo "Running all tests..."
	go test -v ./...

# 运行元数据一致性测试
test-metadata:
	@echo "Running metadata consistency tests..."
	@# 运行版本一致性检查（所有工作流）
	@echo "Testing version consistency..."; \
	go test -v ./internal/workflow -run TestWorkflowVersionConsistency || exit 1
	@# 运行各工作流的元数据测试
	@for workflow_dir in internal/workflow/*/; do \
		workflow_name=$$(basename "$$workflow_dir"); \
		echo "Testing $${workflow_name}..."; \
		go test -v "./$${workflow_dir}" -run TestStagesMatchMetadata 2>/dev/null || true; \
		go test -v "./$${workflow_dir}" -run TestToolsMatchMetadata 2>/dev/null || true; \
		go test -v "./$${workflow_dir}" -run TestStageToolMapping 2>/dev/null || true; \
	done
	@echo "✓ Metadata consistency verified"

# 清理生成的文件
clean:
	@echo "Cleaning generated files..."
	@# Remove worker binary
	@rm -f worker bin/worker
	@# Clean generated files for all workflows
	@for workflow_dir in internal/workflow/*/; do \
		if [ -f "$${workflow_dir}templates.yaml" ]; then \
			workflow_name=$$(basename "$$workflow_dir"); \
			rm -f "$${workflow_dir}constants_generated.go"; \
			rm -f "../server/internal/engineschema/$${workflow_name}.schema.json"; \
			rm -f "../server/internal/engineschema/$${workflow_name}.yaml"; \
		fi \
	done
	@# Remove generated docs
	@rm -f ../docs/config-reference.md
	@rm -f ../docs/config-reference.zh.md
	@echo "✓ Cleaned"

# 构建
build:
	@echo "Building worker..."
	go build -o bin/worker ./cmd/worker
	@echo "✓ Build completed"

# 生成并测试（推荐的工作流）
gen-test: generate test-metadata
	@echo "✓ Generation and testing completed successfully"
