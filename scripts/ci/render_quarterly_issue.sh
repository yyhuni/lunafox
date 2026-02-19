#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
INPUT_PATH="${1:-${REPO_ROOT}/artifacts/version-inventory.json}"
OUTPUT_PATH="${2:-${REPO_ROOT}/artifacts/quarterly-version-review.md}"

python3 - "$INPUT_PATH" "$OUTPUT_PATH" <<'PY'
import json
import sys
from pathlib import Path

input_path = Path(sys.argv[1])
output_path = Path(sys.argv[2])

data = json.loads(input_path.read_text(encoding="utf-8"))


def fmt(value):
    return value if value else "N/A"


def sorted_rows(rows):
    return sorted(rows, key=lambda row: row.get("name", ""))


lines = []
lines.append("<!-- quarterly-version-review -->")
lines.append("")
lines.append("# Quarterly Version Review")
lines.append("")
lines.append(f"Generated at (UTC): `{fmt(data.get('generated_at'))}`")
lines.append("")
lines.append("This report is reminder-only. It does not block CI or release workflows.")
lines.append("")

lines.append("## Current Versions")
lines.append("")
lines.append("### Runtime and Package Manager")
lines.append("")
lines.append("| Component | Current | Source |")
lines.append("| --- | --- | --- |")
for row in sorted_rows(data.get("components", [])):
    lines.append(f"| {row['name']} | {fmt(row.get('current'))} | `{row['source']}` |")
lines.append("")

lines.append("### Scanner Tools")
lines.append("")
lines.append("| Tool | Current | Source |")
lines.append("| --- | --- | --- |")
for row in sorted_rows(data.get("scanner_tools", [])):
    lines.append(f"| {row['name']} | {fmt(row.get('current'))} | `{row['source']}` |")
lines.append("")

lines.append("### Docker Base Images")
lines.append("")
lines.append("| Dockerfile | Base Image |")
lines.append("| --- | --- |")
for row in sorted_rows(data.get("docker_bases", [])):
    lines.append(f"| `{row['name']}` | `{fmt(row.get('current'))}` |")
lines.append("")

lines.append("## Latest Versions")
lines.append("")
lines.append("### Runtime and Package Manager")
lines.append("")
lines.append("| Component | Latest Stable |")
lines.append("| --- | --- |")
for row in sorted_rows(data.get("components", [])):
    lines.append(f"| {row['name']} | {fmt(row.get('latest'))} |")
lines.append("")

lines.append("### Scanner Tools")
lines.append("")
lines.append("| Tool | Latest Stable |")
lines.append("| --- | --- |")
for row in sorted_rows(data.get("scanner_tools", [])):
    lines.append(f"| {row['name']} | {fmt(row.get('latest'))} |")
lines.append("")

lines.append("## Version Gap and Risk")
lines.append("")
lines.append("### Runtime and Package Manager")
lines.append("")
lines.append("| Component | Current | Latest | Status | Risk |")
lines.append("| --- | --- | --- | --- | --- |")
for row in sorted_rows(data.get("components", [])):
    lines.append(
        f"| {row['name']} | {fmt(row.get('current'))} | {fmt(row.get('latest'))} | {fmt(row.get('status'))} | {fmt(row.get('risk'))} |"
    )
lines.append("")

lines.append("### Scanner Tools")
lines.append("")
lines.append("| Tool | Current | Latest | Status | Risk |")
lines.append("| --- | --- | --- | --- | --- |")
for row in sorted_rows(data.get("scanner_tools", [])):
    lines.append(
        f"| {row['name']} | {fmt(row.get('current'))} | {fmt(row.get('latest'))} | {fmt(row.get('status'))} | {fmt(row.get('risk'))} |"
    )
lines.append("")

lines.append("### Docker Base Images")
lines.append("")
lines.append("| Dockerfile | Base Image | Latest | Status | Note |")
lines.append("| --- | --- | --- | --- | --- |")
for row in sorted_rows(data.get("docker_bases", [])):
    lines.append(
        f"| `{row['name']}` | `{fmt(row.get('current'))}` | {fmt(row.get('latest'))} | {fmt(row.get('status'))} | {fmt(row.get('note'))} |"
    )
lines.append("")

lines.append("## UNPINNED High Priority Items")
lines.append("")
unpinned = data.get("unpinned", [])
if unpinned:
    for row in sorted_rows(unpinned):
        lines.append(f"- [HIGH][UNPINNED] `{row['name']}` in `{row['source']}`")
        lines.append(f"  - reason: {row['reason']}")
        lines.append(f"  - recommendation: {row['recommendation']}")
else:
    lines.append("- No unpinned items detected.")
lines.append("")

summary = data.get("summary", {})
lines.append("## Suggested Actions")
lines.append("")
lines.append(f"- Outdated runtime/package-manager components: {summary.get('outdated_components', 0)}")
lines.append(f"- Outdated scanner tools: {summary.get('outdated_scanner_tools', 0)}")
lines.append(f"- Unpinned items: {summary.get('unpinned_items', 0)}")
lines.append("- Keep this issue as a reminder artifact for manual upgrade planning.")
lines.append("- Do not block CI/release based on this report.")

output_path.parent.mkdir(parents=True, exist_ok=True)
output_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

echo "wrote quarterly issue markdown to ${OUTPUT_PATH}"
