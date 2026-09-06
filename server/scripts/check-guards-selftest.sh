#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SELFTEST_SCRIPTS=(
	"check-dto-boundaries-selftest.sh"
	"check-repository-boundaries-selftest.sh"
	"check-layer-dependencies-selftest.sh"
	"check-router-boundaries-selftest.sh"
	"check-handler-boundaries-selftest.sh"
	"check-wiring-conventions-selftest.sh"
)

guard_label() {
	local script="$1"
	case "$script" in
	check-dto-boundaries-selftest.sh) echo "DTO guard self-test" ;;
	check-repository-boundaries-selftest.sh) echo "Repository guard self-test" ;;
	check-layer-dependencies-selftest.sh) echo "Layer dependency guard self-test" ;;
	check-router-boundaries-selftest.sh) echo "Router boundary guard self-test" ;;
	check-handler-boundaries-selftest.sh) echo "Handler boundary guard self-test" ;;
	check-wiring-conventions-selftest.sh) echo "Wiring conventions guard self-test" ;;
	*) echo "$script" ;;
	esac
}

TMP_DIR="$(mktemp -d)"
cleanup() {
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT

begin_group() {
	local title="$1"
	if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
		echo "::group::$title"
	else
		echo "========== $title =========="
	fi
}

end_group() {
	if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
		echo "::endgroup::"
	fi
}

for script in "${SELFTEST_SCRIPTS[@]}"; do
	if [[ ! -x "$ROOT_DIR/scripts/$script" ]]; then
		echo "❌ Missing executable guard self-test script: $ROOT_DIR/scripts/$script"
		exit 1
	fi
done

declare -a PASSED_SCRIPTS=()
declare -a FAILED_SCRIPTS=()

total=${#SELFTEST_SCRIPTS[@]}
index=0

for script in "${SELFTEST_SCRIPTS[@]}"; do
	index=$((index + 1))
	log_file="$TMP_DIR/$script.log"
	label="$(guard_label "$script")"

	echo "[$index/$total] Running $label ($script) ..."
	begin_group "$label"
	if bash "$ROOT_DIR/scripts/$script" >"$log_file" 2>&1; then
		PASSED_SCRIPTS+=("$script")
		cat "$log_file"
		end_group
		echo "  ✅ Passed"
	else
		FAILED_SCRIPTS+=("$script")
		cat "$log_file"
		end_group
		echo "  ❌ Failed (detailed logs were printed above)"
	fi
done

echo
echo "========== Guard Self-test Summary =========="
echo "Total: $total"
echo "Passed: ${#PASSED_SCRIPTS[@]}"
echo "Failed: ${#FAILED_SCRIPTS[@]}"

if [[ ${#PASSED_SCRIPTS[@]} -gt 0 ]]; then
	echo "- Passed list:"
	for script in "${PASSED_SCRIPTS[@]}"; do
		echo "  - $(guard_label "$script") ($script)"
	done
fi

if [[ ${#FAILED_SCRIPTS[@]} -gt 0 ]]; then
	echo
	echo "❌ The following guard self-tests failed. Detailed logs:"
	for script in "${FAILED_SCRIPTS[@]}"; do
		log_file="$TMP_DIR/$script.log"
		begin_group "[FAIL] $(guard_label "$script")"
		cat "$log_file"
		end_group
	done
	exit 1
fi

echo
echo "✅ All guard self-tests passed"
