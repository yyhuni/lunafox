#!/usr/bin/env node

/**
 * Static license and deferred-security scope audit.
 *
 * It is intentionally usable in both the private release job (where an
 * optional source diff is checked) and the secretless public validation job.
 */

import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const TEXT_EXTENSIONS = new Set([".md", ".txt", ".json", ".jsonl", ".yaml", ".yml", ".sh", ".mjs"]);
const FORBIDDEN_DIFF_PREFIXES = [
  "agent/internal/identity/",
  "agent/internal/config/",
  "agent/internal/agentcontrol/transport",
  "server/internal/grpc/agentdata/",
];
const PUBLIC_ENGINE_ARTIFACT_STATEMENT = "Engine Runtime Images and Engine Packages are public artifacts.";
const CLOSED_ENGINE_ARTIFACT_STATEMENT = "Engine Runtime Images and Engine Packages are closed artifacts.";
const FIRST_PARTY_ENGINE_LICENSE_PATTERN = /First-party\s+Engine\s+source\s+is\s+GPL-3\.0-only\./;
const THIRD_PARTY_ENGINE_LICENSE_PATTERN = /Included\s+third-party\s+components\s+remain\s+governed\s+by\s+their\s+applicable\s+licenses\s+and\s+attribution\s+terms\./;

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const args = { root: DEFAULT_ROOT, sourceRoot: "", base: "", json: false };
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--root-dir") args.root = path.resolve(argv[++i] ?? "");
    else if (arg === "--source-root") args.sourceRoot = path.resolve(argv[++i] ?? "");
    else if (arg === "--base") args.base = argv[++i] ?? "";
    else if (arg === "--json") args.json = true;
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/audit-public-security-scope.mjs [--root-dir <dir>] [--source-root <dir> --base <revision>] [--json]\n");
      process.exit(0);
    } else fail(`unknown argument: ${arg}`);
  }
  return args;
}

function walkText(root) {
  const files = [];
  const visit = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (entry.name === ".git" || entry.name === "node_modules") continue;
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) visit(full);
      else if (entry.isFile() && TEXT_EXTENSIONS.has(path.extname(entry.name))) files.push(full);
    }
  };
  visit(root);
  return files;
}

function runDiffAudit(sourceRoot, base) {
  if (!sourceRoot || !base) return { checked: false, changedPaths: [] };
  let output;
  try { output = execFileSync("git", ["-C", sourceRoot, "diff", "--name-only", `${base}...HEAD`], { encoding: "utf8" }); }
  catch (error) { fail(`cannot inspect source diff: ${error.stderr?.toString().trim() || error.message}`); }
  const changedPaths = output.split(/\r?\n/).filter(Boolean);
  const forbidden = changedPaths.filter((name) => FORBIDDEN_DIFF_PREFIXES.some((prefix) => name.startsWith(prefix)));
  if (forbidden.length) fail(`deferred Agent credential/transport scope changed: ${forbidden.join(", ")}`);
  return { checked: true, changedPaths };
}

function containsAuthenticatedAgentClaim(text) {
  // The public notice intentionally contains negated statements such as
  // "does not claim authenticated Agent ...".  Match only affirmative
  // capability claims and keep the check line-oriented so a disclaimer
  // cannot be mistaken for an implementation promise.
  const claimPattern = /\b(?:Agent(?:\s+control-plane)?|authenticated\s+Agent(?:\s+control-plane)?)\b[^.\n]{0,140}\b(?:identity|TLS|mTLS|certificate(?:-chain)?|server identity)\b[^.\n]{0,100}\b(?:is|are|has|provides|supports|uses|verified|authenticated|complete|enabled)\b/i;
  const lines = text.split(/\r?\n/);
  for (const line of lines) {
    if (!claimPattern.test(line)) continue;
    if (/\b(?:does\s+not|do\s+not|not|never|without|no|lacks|unsupported|未|不|无)\b/i.test(line)) continue;
    return true;
  }
  return false;
}

function containsEmbeddedAgentCredentialLiteral(text) {
  // The public Bootstrap source may pass a runtime-created token into the
  // closed Agent. Only an inline long token would make the public tree itself
  // credential-bearing; shell variables and command substitutions are not
  // embedded credentials.
  return /(?:AGENT_AUTHENTICATION_TOKEN|agentAuthenticationToken)"?\s*[:=]\s*(?!["']?\$)["']?[A-Za-z0-9._~+/=-]{20,}/.test(text);
}

function audit(root, sourceRoot, base) {
  const licensePath = path.join(root, "LICENSE");
  const noticePath = path.join(root, "NOTICE-CLOSED-ARTIFACTS.md");
  const readmePath = path.join(root, "README.md");
  const docsPath = path.join(root, "docs/public-deployment.md");
  for (const file of [licensePath, noticePath, readmePath, docsPath]) if (!fs.existsSync(file)) fail(`required policy file is missing: ${path.relative(root, file)}`);
  const license = fs.readFileSync(licensePath, "utf8");
  if (!license.includes("SPDX-License-Identifier: GPL-3.0-only")) fail("license must declare SPDX GPL-3.0-only");
  if (/GPL-3\.0-or-later/i.test(license)) fail("GPL-3.0-or-later is not allowed");
  if (!/first-party public Runtime and Engine source/i.test(license)) {
    fail("license must include first-party public Runtime and Engine source");
  }
  const policyDocuments = [
    { path: noticePath, text: fs.readFileSync(noticePath, "utf8") },
    { path: readmePath, text: fs.readFileSync(readmePath, "utf8") },
    { path: docsPath, text: fs.readFileSync(docsPath, "utf8") },
  ];
  for (const document of policyDocuments) {
    const label = path.relative(root, document.path);
    if (!document.text.includes(PUBLIC_ENGINE_ARTIFACT_STATEMENT)) {
      fail(`${label} must state that Engine Runtime Images and Engine Packages are public artifacts`);
    }
    if (!FIRST_PARTY_ENGINE_LICENSE_PATTERN.test(document.text)) {
      fail(`${label} must state the GPL-3.0-only license for first-party Engine source`);
    }
    if (!THIRD_PARTY_ENGINE_LICENSE_PATTERN.test(document.text)) {
      fail(`${label} must preserve third-party Engine license and attribution terms`);
    }
    if (document.text.includes(CLOSED_ENGINE_ARTIFACT_STATEMENT)) {
      fail(`${label} must not classify Engine Runtime Images and Engine Packages as closed artifacts`);
    }
  }
  const allPolicyText = policyDocuments.map((document) => document.text).join("\n");
  if (/GPL-3\.0-or-later/i.test(allPolicyText)) fail("public notices must not imply GPL-3.0-or-later");
  for (const phrase of ["long-lived", "bearer", "eight-character", "TLS", "separate security change"]) {
    if (!allPolicyText.toLowerCase().includes(phrase.toLowerCase())) fail(`deferred security limitation is not documented: ${phrase}`);
  }
  if (containsAuthenticatedAgentClaim(allPolicyText)) {
    fail("public documentation overclaims authenticated Agent control-plane identity");
  }
  if (!/self-hosted|自有或受控/i.test(allPolicyText)) fail("self-hosted boundary is not documented");
  if (!/written (?:authorization|commercial agreement)|书面授权|商业协议/i.test(allPolicyText)) fail("commercial/redistribution authorization boundary is not documented");
  const diff = runDiffAudit(sourceRoot, base);
  // A public projection must never contain an alternate credential protocol or
  // a TLS implementation claim, even if a future source file is accidentally
  // added to the allowlist.
  for (const file of walkText(root)) {
    const rel = path.relative(root, file).split(path.sep).join("/");
    const text = fs.readFileSync(file, "utf8");
    if (containsEmbeddedAgentCredentialLiteral(text)) fail(`possible long-lived credential literal in public file: ${rel}`);
    // Require the complete PEM delimiter so the validator's own detection
    // strings (for example, `BEGIN RSA PRIVATE KEY`) are not mistaken for
    // embedded key material.
    if (/-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/.test(text)) fail(`private key material in public file: ${rel}`);
  }
  return {
    schemaVersion: 1,
    passed: true,
    license: "GPL-3.0-only",
    publicEngineArtifacts: true,
    publicEngineLicenseDisclosure: true,
    deferredAgentTLS: true,
    deferredAgentCredentialProtocol: true,
    diff,
  };
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const result = audit(fs.realpathSync(args.root), args.sourceRoot, args.base);
  process.stdout.write(`${args.json ? JSON.stringify(result, null, 2) : "public license/security scope audit passed"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try { main(); } catch (error) { process.stderr.write(`public scope audit failed: ${error.message}\n`); process.exitCode = 1; }
}

export {
  audit,
  containsAuthenticatedAgentClaim,
  containsEmbeddedAgentCredentialLiteral,
  runDiffAudit,
  PUBLIC_ENGINE_ARTIFACT_STATEMENT,
  CLOSED_ENGINE_ARTIFACT_STATEMENT,
  FIRST_PARTY_ENGINE_LICENSE_PATTERN,
  THIRD_PARTY_ENGINE_LICENSE_PATTERN,
};
