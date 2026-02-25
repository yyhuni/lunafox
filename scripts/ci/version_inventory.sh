#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_PATH="${1:-${REPO_ROOT}/artifacts/version-inventory.json}"

mkdir -p "$(dirname "${OUTPUT_PATH}")"

python3 - "$REPO_ROOT" "$OUTPUT_PATH" <<'PY'
import json
import re
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from urllib.parse import quote

repo_root = Path(sys.argv[1])
output_path = Path(sys.argv[2])


def fetch_json(url: str):
    req = urllib.request.Request(url, headers={"User-Agent": "lunafox-quarterly-review"})
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            return json.load(resp)
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
        return None


def read_text(relative_path: str) -> str:
    path = repo_root / relative_path
    return path.read_text(encoding="utf-8")


def parse_semver(value: str):
    if not value:
        return None
    normalized = value.strip()
    if normalized.startswith("go"):
        normalized = normalized[2:]
    if normalized.startswith("v"):
        normalized = normalized[1:]
    match = re.search(r"(\d+)\.(\d+)(?:\.(\d+))?", normalized)
    if not match:
        return None
    major = int(match.group(1))
    minor = int(match.group(2))
    patch = int(match.group(3) or 0)
    return (major, minor, patch)


def compare_versions(current: str, latest: str):
    cur = parse_semver(current)
    lat = parse_semver(latest)
    if not cur or not lat:
        return "unknown"
    if cur == lat:
        return "up_to_date"
    if cur < lat:
        return "outdated"
    return "ahead"


def risk_from_status(status: str, current: str, latest: str):
    if status in ("unknown", "ahead"):
        return "info"
    if status == "up_to_date":
        return "low"
    cur = parse_semver(current)
    lat = parse_semver(latest)
    if not cur or not lat:
        return "info"
    major_gap = lat[0] - cur[0]
    if major_gap >= 1:
        return "medium"
    return "low"


def extract_go_version(go_mod_relative_path: str):
    text = read_text(go_mod_relative_path)
    for line in text.splitlines():
        line = line.strip()
        if line.startswith("go "):
            return line.split()[1]
    return None


def extract_first_from_image(dockerfile_relative_path: str):
    text = read_text(dockerfile_relative_path)
    for line in text.splitlines():
        line = line.strip()
        if line.startswith("FROM "):
            parts = line.split()
            if len(parts) >= 2:
                return parts[1]
    return None


def extract_node_version_from_frontend_dockerfile():
    image = extract_first_from_image("frontend/Dockerfile")
    if not image:
        return None
    match = re.match(r"node:([^\s-]+)", image)
    if match:
        return match.group(1)
    return None


def extract_pnpm_version():
    package_json = json.loads(read_text("frontend/package.json"))
    package_manager = package_json.get("packageManager", "")
    match = re.match(r"pnpm@(.+)", package_manager)
    if match:
        return match.group(1)
    return None


def extract_scanner_go_tools():
    text = read_text("worker/Dockerfile")
    tools = []
    pattern = re.compile(r"install_tool\s+([^\s@]+)@([^;\s]+)")

    def canonical_module(module_path: str) -> str:
        if "/cmd/" in module_path:
            return module_path.split("/cmd/", 1)[0]
        return module_path

    def display_name(module_path: str) -> str:
        module_root = canonical_module(module_path)
        parts = module_root.rstrip("/").split("/")
        if not parts:
            return module_path
        tail = parts[-1]
        if re.fullmatch(r"v\d+", tail) and len(parts) >= 2:
            return parts[-2]
        return tail

    for module_path, version in pattern.findall(text):
        module_root = canonical_module(module_path)
        tool_name = display_name(module_path)
        tools.append(
            {
                "name": tool_name,
                "module": module_path,
                "module_root": module_root,
                "current": version,
                "source": "worker/Dockerfile",
            }
        )
    return tools


def latest_go_version():
    payload = fetch_json("https://go.dev/dl/?mode=json")
    if not isinstance(payload, list):
        return None
    for item in payload:
        if isinstance(item, dict) and item.get("stable"):
            version = str(item.get("version", ""))
            if version.startswith("go"):
                return version[2:]
            return version
    return None


def latest_node_lts_version():
    payload = fetch_json("https://nodejs.org/dist/index.json")
    if not isinstance(payload, list):
        return None
    for item in payload:
        if not isinstance(item, dict):
            continue
        if item.get("lts"):
            version = str(item.get("version", ""))
            return version.lstrip("v")
    return None


def latest_pnpm_version():
    payload = fetch_json("https://registry.npmjs.org/pnpm/latest")
    if not isinstance(payload, dict):
        return None
    return payload.get("version")


def latest_go_module_version(module_path: str):
    encoded = quote(module_path, safe="/")
    payload = fetch_json(f"https://proxy.golang.org/{encoded}/@latest")
    if not isinstance(payload, dict):
        return None
    return payload.get("Version")


def extract_unpinned_items():
    text = read_text("worker/Dockerfile")
    findings = []

    if re.search(r"\bnmap\b", text):
        findings.append(
            {
                "name": "nmap",
                "source": "worker/Dockerfile",
                "reason": "apt package installed without explicit version pin",
                "recommendation": "Pin apt package version or move nmap to a dedicated scanner image pinned by digest.",
            }
        )

    if re.search(r"\bmasscan\b", text):
        findings.append(
            {
                "name": "masscan",
                "source": "worker/Dockerfile",
                "reason": "apt package installed without explicit version pin",
                "recommendation": "Pin apt package version or move masscan to a dedicated scanner image pinned by digest.",
            }
        )

    pipx_pattern = re.compile(r"pipx install ([A-Za-z0-9_.-]+(?:==[A-Za-z0-9_.-]+)?)")
    seen = set()
    for raw_pkg in pipx_pattern.findall(text):
        pkg_name = raw_pkg.split("==", 1)[0]
        if pkg_name in seen:
            continue
        seen.add(pkg_name)
        if "==" in raw_pkg:
            continue
        findings.append(
            {
                "name": pkg_name,
                "source": "worker/Dockerfile",
                "reason": "pipx package installed without explicit version pin",
                "recommendation": f"Pin pipx package version, e.g. pipx install {pkg_name}==<version>.",
            }
        )

    return findings


def collect_docker_bases():
    dockerfiles = [
        "server/Dockerfile",
        "agent/Dockerfile",
        "worker/Dockerfile",
        "frontend/Dockerfile",
        "docker/nginx/Dockerfile",
    ]
    rows = []
    for dockerfile in dockerfiles:
        image = extract_first_from_image(dockerfile)
        rows.append(
            {
                "name": dockerfile,
                "current": image,
                "latest": None,
                "status": "manual_review",
                "risk": "info",
                "source": dockerfile,
                "note": "Latest tag lookup is manual for Docker base images.",
            }
        )
    return rows


def parse_compose_service_images(compose_relative_path: str, target_services):
    text = read_text(compose_relative_path)
    images = {}
    current_service = None

    service_header = re.compile(r"^\s{2}([A-Za-z0-9_-]+):\s*$")
    image_line = re.compile(r"^\s{4}image:\s*(.+?)\s*$")

    for line in text.splitlines():
        header_match = service_header.match(line)
        if header_match:
            current_service = header_match.group(1)
            continue

        image_match = image_line.match(line)
        if image_match and current_service in target_services:
            image_ref = image_match.group(1).strip().strip("'\"")
            images[current_service] = image_ref

    rows = []
    for service in target_services:
        rows.append(
            {
                "name": service,
                "current": images.get(service),
                "latest": None,
                "status": "manual_review",
                "risk": "info",
                "source": compose_relative_path,
                "note": "Version extracted from docker-compose service image.",
            }
        )
    return rows


latest_go = latest_go_version()
latest_node = latest_node_lts_version()
latest_pnpm = latest_pnpm_version()

components = []
runtime_targets = [
    ("go(server)", "server/go.mod"),
    ("go(agent)", "agent/go.mod"),
    ("go(worker)", "worker/go.mod"),
    ("go(tools/installer)", "tools/installer/go.mod"),
]

for name, path in runtime_targets:
    current = extract_go_version(path)
    status = compare_versions(current, latest_go)
    components.append(
        {
            "name": name,
            "category": "runtime",
            "current": current,
            "latest": latest_go,
            "status": status,
            "risk": risk_from_status(status, current, latest_go),
            "source": path,
        }
    )

node_current = extract_node_version_from_frontend_dockerfile()
node_status = compare_versions(node_current, latest_node)
components.append(
    {
        "name": "node(frontend)",
        "category": "runtime",
        "current": node_current,
        "latest": latest_node,
        "status": node_status,
        "risk": risk_from_status(node_status, node_current, latest_node),
        "source": "frontend/Dockerfile",
    }
)

pnpm_current = extract_pnpm_version()
pnpm_status = compare_versions(pnpm_current, latest_pnpm)
components.append(
    {
        "name": "pnpm(frontend)",
        "category": "package-manager",
        "current": pnpm_current,
        "latest": latest_pnpm,
        "status": pnpm_status,
        "risk": risk_from_status(pnpm_status, pnpm_current, latest_pnpm),
        "source": "frontend/package.json",
    }
)

scanner_tools = []
for tool in extract_scanner_go_tools():
    latest = latest_go_module_version(tool["module_root"])
    status = compare_versions(tool["current"], latest)
    scanner_tools.append(
        {
            "name": tool["name"],
            "module": tool["module"],
            "current": tool["current"],
            "latest": latest,
            "status": status,
            "risk": risk_from_status(status, tool["current"], latest),
            "source": tool["source"],
        }
    )

unpinned = extract_unpinned_items()

summary = {
    "outdated_components": sum(1 for c in components if c["status"] == "outdated"),
    "outdated_scanner_tools": sum(1 for t in scanner_tools if t["status"] == "outdated"),
    "unpinned_items": len(unpinned),
    "compose_service_images": 3,
}

payload = {
    "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "components": components,
    "scanner_tools": scanner_tools,
    "docker_bases": collect_docker_bases(),
    "compose_service_images": parse_compose_service_images(
        "docker/docker-compose.yml",
        ["loki", "redis", "postgres"],
    ),
    "unpinned": unpinned,
    "summary": summary,
}

output_path.write_text(json.dumps(payload, indent=2, sort_keys=True), encoding="utf-8")
PY

echo "wrote inventory to ${OUTPUT_PATH}"
