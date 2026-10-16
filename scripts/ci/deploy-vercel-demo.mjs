#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const VERCEL_CLI_VERSION = "64.0.2";
const ENV_FILE = path.join(".vercel", ".env.production.local");
const REQUIRED_FLAGS = ["DEMO_SITE", "NEXT_PUBLIC_USE_MOCK", "NEXT_PUBLIC_SKIP_AUTH"];
const FORBIDDEN_KEYS = ["BACKEND_URL", "NEXT_PUBLIC_BACKEND_URL"];

function envAssignment(line) {
  const match = /^(\s*)([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$/.exec(line);
  if (!match) return null;
  return { key: match[2], rawValue: match[3] };
}

function configuredValue(rawValue) {
  const trimmed = rawValue.trim();
  if (
    (trimmed.startsWith("\"") && trimmed.endsWith("\""))
    || (trimmed.startsWith("'") && trimmed.endsWith("'"))
  ) {
    return trimmed.slice(1, -1).trim();
  }
  return trimmed;
}

/**
 * Force the demo flags and refuse a real backend that arrived from the Vercel project.
 * An empty value is treated as unset so a blank dashboard field does not become a URL.
 */
export function prepareVercelDemoEnvFile(content) {
  const lines = String(content ?? "").split(/\r?\n/);
  const seen = new Set();
  const next = [];
  for (const line of lines) {
    const assignment = envAssignment(line);
    if (!assignment) {
      next.push(line);
      continue;
    }
    if (FORBIDDEN_KEYS.includes(assignment.key) && configuredValue(assignment.rawValue)) {
      throw new Error(`Vercel demo builds must not configure ${assignment.key}.`);
    }
    if (REQUIRED_FLAGS.includes(assignment.key)) {
      if (seen.has(assignment.key)) continue;
      seen.add(assignment.key);
      next.push(`${assignment.key}="true"`);
      continue;
    }
    next.push(line);
  }
  for (const key of REQUIRED_FLAGS) {
    if (!seen.has(key)) next.push(`${key}="true"`);
  }
  while (next.length > 0 && next.at(-1) === "") next.pop();
  return `${next.join("\n")}\n`;
}

export function assertVercelDemoCredentials(env) {
  for (const key of ["VERCEL_TOKEN", "VERCEL_ORG_ID", "VERCEL_PROJECT_ID"]) {
    if (!env[key]?.trim()) {
      throw new Error(`Vercel demo deployment is missing ${key}.`);
    }
  }
  for (const key of FORBIDDEN_KEYS) {
    if (env[key]?.trim()) {
      throw new Error(`Vercel demo builds must not configure ${key}.`);
    }
  }
}

function runVercel(cwd, env, args) {
  const result = spawnSync("pnpm", ["dlx", `vercel@${VERCEL_CLI_VERSION}`, ...args], {
    cwd,
    env,
    stdio: "inherit",
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(`vercel ${args[0]} exited with status ${result.status ?? "unknown"}.`);
  }
}

export function deployVercelDemo({
  cwd = process.cwd(),
  env = process.env,
  run = runVercel,
  readFile = fs.readFileSync,
  writeFile = fs.writeFileSync,
} = {}) {
  assertVercelDemoCredentials(env);
  run(cwd, env, ["pull", "--yes", "--environment=production"]);
  const filePath = path.join(cwd, ENV_FILE);
  const current = fs.existsSync(filePath) ? readFile(filePath, "utf8") : "";
  writeFile(filePath, prepareVercelDemoEnvFile(current));
  run(cwd, env, ["build", "--prod"]);
  run(cwd, env, ["deploy", "--prebuilt", "--prod", "--yes"]);
}

const isDirect = process.argv[1] && path.resolve(process.argv[1]) === path.resolve(fileURLToPath(import.meta.url));
if (isDirect) {
  try {
    deployVercelDemo();
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exit(1);
  }
}
