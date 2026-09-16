#!/usr/bin/env node

/**
 * Materialize the one-time public-main deployment bootstrap from the last
 * verified dual-package Release. The export publisher consumes it only when
 * the destination .env sentinel is absent.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const options = { root: ROOT, package: "", output: "", json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/prepare-legacy-public-deployment.mjs --package <zip> --output <dir> [--root <dir>] [--json]\n");
      process.exit(0);
    }
    if (!["--root", "--package", "--output"].includes(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--root") options.root = path.resolve(value);
    else if (arg === "--package") options.package = path.resolve(value);
    else options.output = path.resolve(value);
  }
  if (!options.package || !fs.existsSync(options.package) || !fs.statSync(options.package).isFile()) fail("legacy package is missing");
  if (!options.output) fail("--output is required");
  return options;
}

function readPolicy(root) {
  const policyPath = path.join(root, "scripts", "ci", "public-release-policy.json");
  const policy = JSON.parse(fs.readFileSync(policyPath, "utf8"));
  const bootstrap = policy.legacyDeploymentBootstrap;
  if (!bootstrap || typeof bootstrap !== "object" ||
      JSON.stringify(Object.keys(bootstrap).sort()) !== JSON.stringify(["packageName", "paths", "releaseTag", "sha256"])) {
    fail("public release policy is missing the exact legacy deployment bootstrap");
  }
  if (!/^v\d+\.\d+\.\d+-alpha\.\d+$/.test(bootstrap.releaseTag) ||
      bootstrap.packageName !== `lunafox-${bootstrap.releaseTag}-dockerhub.zip` ||
      !/^[a-f0-9]{64}$/.test(bootstrap.sha256) ||
      !Array.isArray(bootstrap.paths) || bootstrap.paths.length === 0 || !bootstrap.paths.includes(".env")) {
    fail("legacy deployment bootstrap policy is invalid");
  }
  return bootstrap;
}

function listArchive(archive) {
  const raw = execFileSync("unzip", ["-Z1", archive], { encoding: "utf8" });
  const entries = raw.split(/\r?\n/).filter(Boolean);
  for (const entry of entries) {
    const normalized = path.posix.normalize(entry);
    if (!entry || entry.startsWith("/") || entry.includes("\\") || normalized !== entry || entry.split("/").some((part) => part === "." || part === "..")) {
      fail(`legacy package contains an unsafe path: ${entry}`);
    }
  }
  return new Set(entries);
}

function prepareLegacyDeployment({ root = ROOT, package: packagePath, output }) {
  const bootstrap = readPolicy(root);
  if (path.basename(packagePath) !== bootstrap.packageName) fail("legacy package name does not match policy");
  const packageBytes = fs.readFileSync(packagePath);
  const digest = crypto.createHash("sha256").update(packageBytes).digest("hex");
  if (digest !== bootstrap.sha256) fail("legacy package digest does not match policy");
  const entries = listArchive(packagePath);
  for (const relative of bootstrap.paths) {
    if (!entries.has(relative)) fail(`legacy package is missing deployment path: ${relative}`);
  }
  if (fs.existsSync(output) && fs.readdirSync(output).length > 0) fail("legacy bootstrap output must be empty");
  fs.mkdirSync(output, { recursive: true });
  for (const relative of bootstrap.paths) {
    const bytes = execFileSync("unzip", ["-p", packagePath, relative], { encoding: "buffer", maxBuffer: 32 * 1024 * 1024 });
    const target = path.join(output, ...relative.split("/"));
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, bytes, { mode: 0o644 });
  }
  const env = fs.readFileSync(path.join(output, ".env"));
  const envExample = fs.readFileSync(path.join(output, ".env.example"));
  if (!env.equals(envExample)) fail("legacy package .env and .env.example differ");
  const manifest = fs.readFileSync(path.join(output, "release.manifest.yaml"), "utf8");
  const version = manifest.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1];
  if (`v${version}` !== bootstrap.releaseTag) fail("legacy package manifest version does not match policy");
  return { schemaVersion: 1, releaseTag: bootstrap.releaseTag, packageName: bootstrap.packageName, sha256: digest, paths: bootstrap.paths, output };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = prepareLegacyDeployment(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : result.output}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) {
    process.stderr.write(`legacy public deployment bootstrap failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export { parseArgs, prepareLegacyDeployment };
