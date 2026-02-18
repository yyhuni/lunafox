#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const root = process.cwd();
const targets = [
  "frontend/app",
  "frontend/components",
  "frontend/hooks",
  "frontend/lib",
  "frontend/services",
  "frontend/types",
  "frontend/messages",
];

const violations = [];

function walk(dir) {
  if (!fs.existsSync(dir)) return;
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    if (entry.name === "node_modules" || entry.name === ".next") continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
      continue;
    }

    if (!/\.(ts|tsx|css)$/.test(entry.name)) continue;
    if (/[A-Z]/.test(entry.name)) {
      violations.push(`文件名必须为 kebab-case（禁止大写）: ${full}`);
    }

    if (full.includes(`${path.sep}frontend${path.sep}services${path.sep}`)) {
      if (entry.name.endsWith(".api.ts")) {
        violations.push(`services 禁止 .api.ts 命名: ${full}`);
      }
      if (!entry.name.endsWith(".service.ts")) {
        violations.push(`services 统一使用 .service.ts: ${full}`);
      }
    }
  }
}

for (const rel of targets) {
  walk(path.join(root, rel));
}

if (violations.length > 0) {
  console.error("❌ Frontend 命名规范检查失败");
  for (const issue of violations) {
    console.error(issue);
  }
  process.exit(1);
}

console.log("✅ Frontend 命名规范检查通过");
