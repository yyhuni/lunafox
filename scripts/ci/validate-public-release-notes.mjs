#!/usr/bin/env node

/**
 * Validate the reviewed, public-safe Markdown body used by one GitHub Release.
 * The same function is consumed by private export checks and the public tree
 * verifier so a retry cannot silently apply a different interpretation.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT_DIR = path.resolve(SCRIPT_DIR, "../..");
const RELEASE_NOTES_PREFIX = "release-notes";
const MAX_NOTES_BYTES = 64 * 1024;
const TAG_PATTERN = /^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const BILINGUAL_SECTION_NAMES = Object.freeze(["English", "简体中文"]);
const INTERNAL_PATTERNS = Object.freeze([
  { code: "private-repository-reference", pattern: /lunafox-private|yyhuni\/lunafox-private/i },
  { code: "private-workflow-reference", pattern: /\.github\/workflows\/release\.ya?ml|refs\/tags\/\*/i },
  { code: "credential-reference", pattern: /\b(?:GITHUB_TOKEN|GH_TOKEN|DOCKERHUB_TOKEN|COSIGN_PRIVATE_KEY|[A-Z][A-Z0-9_]*_PAT)\b|\bsecrets\.[A-Za-z_][A-Za-z0-9_]*/ },
  { code: "internal-release-marker", pattern: /PUBLIC_(?:EXPORT_MANIFEST|PROVENANCE)|chore\((?:export|deploy|workflow)\)|deployment snapshot/i },
]);

class ReleaseNotesValidationError extends Error {
  constructor(code, message) {
    super(message);
    this.name = "ReleaseNotesValidationError";
    this.code = code;
  }
}

function fail(code, message) {
  throw new ReleaseNotesValidationError(code, message);
}

function parseArgs(argv) {
  const options = { rootDir: DEFAULT_ROOT_DIR, tag: "", notesFile: "", json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    }
    if (!["--root-dir", "--tag", "--notes-file"].includes(arg)) fail("invalid-argument", `unknown argument: ${arg}`);
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) fail("invalid-argument", `${arg} requires a value`);
    if (arg === "--root-dir") options.rootDir = path.resolve(value);
    else if (arg === "--tag") options.tag = value;
    else options.notesFile = value;
    index += 1;
  }
  if (!options.tag) fail("missing-tag", "--tag is required");
  return options;
}

function usage() {
  return "Usage: node scripts/ci/validate-public-release-notes.mjs --tag <tag> [--root-dir <dir>] [--notes-file <path>] [--json]\n";
}

function expectedNotesPath(tag) {
  if (!TAG_PATTERN.test(tag)) fail("invalid-tag", `invalid release tag: ${tag}`);
  return `${RELEASE_NOTES_PREFIX}/${tag}.md`;
}

function normalizeNotesPath(value) {
  if (typeof value !== "string" || !value || value.includes("\\") || value.startsWith("/") || value.includes("\0")) {
    fail("invalid-notes-path", "notes file must be a relative POSIX path");
  }
  const normalized = path.posix.normalize(value);
  if (normalized !== value || normalized === "." || normalized.startsWith("../") || normalized.includes("/../")) {
    fail("invalid-notes-path", `notes file is not contained: ${value}`);
  }
  return normalized;
}

function decodeUtf8(bytes, relativePath) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (error) {
    fail("invalid-utf8", `${relativePath} is not valid UTF-8: ${error.message}`);
  }
}

function extractBilingualSections(text, relativePath = "release notes") {
  const headings = [...text.matchAll(/^##[ \t]+(English|简体中文)[ \t]*$/gmu)];
  const sections = {};
  for (const heading of headings) {
    const name = heading[1];
    if (sections[name]) fail("duplicate-language-section", `${relativePath} contains duplicate ## ${name} sections`);
    const nextHeading = /^##[ \t]+[^\n]+$/gmu;
    nextHeading.lastIndex = heading.index + heading[0].length;
    const next = nextHeading.exec(text);
    const body = text.slice(heading.index + heading[0].length, next?.index ?? text.length).trim();
    sections[name] = { present: true, nonEmpty: Boolean(body) };
    if (!body) fail("empty-language-section", `${relativePath} has an empty ## ${name} section`);
  }
  for (const name of BILINGUAL_SECTION_NAMES) {
    if (!sections[name]) fail("missing-language-section", `${relativePath} is missing required ## ${name} section`);
  }
  return sections;
}

function validateReleaseNotes({ rootDir = DEFAULT_ROOT_DIR, tag, notesFile = "" } = {}) {
  const relativePath = expectedNotesPath(tag);
  if (notesFile && normalizeNotesPath(notesFile) !== relativePath) {
    fail("tag-path-mismatch", `release notes path must be ${relativePath}, got ${notesFile}`);
  }
  const root = fs.realpathSync(rootDir);
  const absolutePath = path.join(root, ...relativePath.split("/"));
  let stat;
  try {
    stat = fs.lstatSync(absolutePath);
  } catch (error) {
    fail("missing-notes", `missing release notes for ${tag}: ${relativePath}`);
  }
  if (!stat.isFile() || stat.isSymbolicLink()) fail("not-regular-file", `${relativePath} must be a regular file`);
  const bytes = fs.readFileSync(absolutePath);
  if (bytes.length > MAX_NOTES_BYTES) fail("notes-too-large", `${relativePath} exceeds ${MAX_NOTES_BYTES} bytes`);
  const text = decodeUtf8(bytes, relativePath);
  if (!text.trim()) fail("notes-empty", `${relativePath} must not be empty`);
  if (/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/u.test(text)) {
    fail("control-character", `${relativePath} contains an unsupported control character`);
  }
  for (const { code, pattern } of INTERNAL_PATTERNS) {
    if (pattern.test(text)) fail(code, `${relativePath} contains a blocked internal marker (${code})`);
  }
  const bilingualSections = extractBilingualSections(text, relativePath);
  return {
    schemaVersion: 1,
    passed: true,
    releaseTag: tag,
    notesPath: relativePath,
    bytes: bytes.length,
    sha256: crypto.createHash("sha256").update(bytes).digest("hex"),
    bilingualSections,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = validateReleaseNotes(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `release notes verified: ${result.notesPath} (${result.bytes} bytes)`}\n`);
  return result;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`release notes validation failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export {
  INTERNAL_PATTERNS,
  BILINGUAL_SECTION_NAMES,
  MAX_NOTES_BYTES,
  RELEASE_NOTES_PREFIX,
  TAG_PATTERN,
  expectedNotesPath,
  extractBilingualSections,
  validateReleaseNotes,
};
