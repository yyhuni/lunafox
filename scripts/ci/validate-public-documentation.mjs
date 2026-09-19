#!/usr/bin/env node

/**
 * Validate the source-owned English/Simplified Chinese user-document pairs.
 * The check is intentionally structural; translation quality remains a human
 * review responsibility.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT_DIR = path.resolve(SCRIPT_DIR, "../..");
const MAX_DOCUMENT_BYTES = 512 * 1024;
const DOCUMENT_PAIRS = Object.freeze([
  { english: "README.md", chinese: "README.zh-CN.md" },
  { english: "docs/public-deployment.md", chinese: "docs/public-deployment.zh-CN.md" },
]);

class DocumentationValidationError extends Error {
  constructor(code, message) {
    super(message);
    this.name = "DocumentationValidationError";
    this.code = code;
  }
}

function fail(code, message) {
  throw new DocumentationValidationError(code, message);
}

function parseArgs(argv) {
  const options = { rootDir: DEFAULT_ROOT_DIR, json: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      options.json = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/validate-public-documentation.mjs [--root-dir <dir>] [--json]\n");
      process.exit(0);
    }
    if (arg !== "--root-dir") fail("invalid-argument", `unknown argument: ${arg}`);
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) fail("invalid-argument", "--root-dir requires a value");
    options.rootDir = path.resolve(value);
    index += 1;
  }
  return options;
}

function normalizeLines(value) {
  return value.replace(/\r\n?/g, "\n").replace(/[ \t]+$/gm, "").trim();
}

function decodeUtf8(bytes, relativePath) {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (error) {
    fail("invalid-utf8", `${relativePath} is not valid UTF-8: ${error.message}`);
  }
}

function readDocument(root, relativePath) {
  const absolutePath = path.join(root, ...relativePath.split("/"));
  let stat;
  try {
    stat = fs.lstatSync(absolutePath);
  } catch (error) {
    fail("missing-document", `missing required document: ${relativePath}`);
  }
  if (!stat.isFile() || stat.isSymbolicLink()) {
    fail("not-regular-file", `${relativePath} must be a regular file`);
  }
  const bytes = fs.readFileSync(absolutePath);
  if (bytes.length > MAX_DOCUMENT_BYTES) {
    fail("document-too-large", `${relativePath} exceeds ${MAX_DOCUMENT_BYTES} bytes`);
  }
  const text = decodeUtf8(bytes, relativePath);
  if (!text.trim()) fail("document-empty", `${relativePath} must not be empty`);
  return {
    path: relativePath,
    text,
    bytes: bytes.length,
    sha256: crypto.createHash("sha256").update(bytes).digest("hex"),
  };
}

function extractCodeBlocks(text) {
  const blocks = [];
  const pattern = /```([^\n]*)\n([\s\S]*?)\n```/g;
  for (const match of text.matchAll(pattern)) {
    blocks.push({ language: match[1].trim(), body: normalizeLines(match[2]) });
  }
  return blocks;
}

function withoutCodeBlocks(text) {
  return text.replace(/```[^\n]*\n[\s\S]*?\n```/g, "");
}

function normalizeInlineToken(token, pair) {
  if (!pair) return token;
  if (new Set([pair.english, pair.chinese]).has(token)) return "<language-peer-token>";
  if (new Set(["docs/public-deployment.md", "docs/public-deployment.zh-CN.md"]).has(token)) {
    return "<deployment-guide-token>";
  }
  return token;
}

function extractInlineTokens(text, pair) {
  const tokens = [];
  const body = withoutCodeBlocks(text);
  for (const match of body.matchAll(/`([^`\n]+)`/g)) {
    const token = match[1].trim();
    if (token) tokens.push(normalizeInlineToken(token, pair));
  }
  return tokens.sort();
}

function splitLinkDestination(destination) {
  const normalized = destination.trim().replace(/[),.;]+$/g, "");
  const match = normalized.match(/^([^?#]*)([?#].*)?$/);
  return { path: match?.[1] ?? normalized, suffix: match?.[2] ?? "" };
}

function resolveRelativeLink(documentPath, destination) {
  const { path: linkPath, suffix } = splitLinkDestination(destination);
  if (!linkPath || linkPath.startsWith("#") || /^[a-z][a-z0-9+.-]*:/i.test(linkPath) || linkPath.startsWith("//")) {
    return `${linkPath}${suffix}`;
  }
  return `${path.posix.normalize(path.posix.join(path.posix.dirname(documentPath), linkPath))}${suffix}`;
}

function normalizeLink(destination, pair, documentPath = pair.english) {
  const resolved = resolveRelativeLink(documentPath, destination);
  const { path: resolvedPath, suffix } = splitLinkDestination(resolved);
  const languagePaths = new Set([pair.english, pair.chinese]);
  if (languagePaths.has(resolvedPath)) return "<language-peer>";
  if (resolvedPath === "docs/public-deployment.md" || resolvedPath === "docs/public-deployment.zh-CN.md") {
    return "<deployment-guide>";
  }
  return `${resolvedPath}${suffix}`;
}

function extractLinks(text, pair, documentPath = pair.english) {
  const links = [];
  const body = withoutCodeBlocks(text);
  for (const match of body.matchAll(/!?\[[^\]]*\]\(([^)\s]+)(?:\s+[^)]*)?\)/g)) {
    links.push(normalizeLink(match[1], pair, documentPath));
  }
  for (const match of body.matchAll(/https?:\/\/[^\s)>\]]+/g)) {
    links.push(normalizeLink(match[0], pair, documentPath));
  }
  return links.sort();
}

function extractHeadingLevels(text) {
  return [...text.matchAll(/^(#{1,6})\s+/gm)].map((match) => match[1].length);
}

function assertEqual(label, english, chinese, pair) {
  if (JSON.stringify(english) !== JSON.stringify(chinese)) {
    fail("structure-drift", `${pair.english} and ${pair.chinese} differ in ${label}: ${JSON.stringify({ english, chinese })}`);
  }
}

function hasLanguagePeerLink(document, pair) {
  return extractLinks(document.text, pair, document.path).includes("<language-peer>");
}

function validatePair(root, pair) {
  const english = readDocument(root, pair.english);
  const chinese = readDocument(root, pair.chinese);
  if (!/[\u3400-\u9fff]/u.test(chinese.text)) {
    fail("missing-chinese-text", `${pair.chinese} must contain Simplified Chinese text`);
  }
  if (!hasLanguagePeerLink(english, pair) || !hasLanguagePeerLink(chinese, pair)) {
    fail("language-link-mismatch", `${pair.english} and ${pair.chinese} must link to each other`);
  }
  assertEqual("code blocks", extractCodeBlocks(english.text), extractCodeBlocks(chinese.text), pair);
  assertEqual("technical inline tokens", extractInlineTokens(english.text, pair), extractInlineTokens(chinese.text, pair), pair);
  assertEqual("links", extractLinks(english.text, pair, english.path), extractLinks(chinese.text, pair, chinese.path), pair);
  assertEqual("heading levels", extractHeadingLevels(english.text), extractHeadingLevels(chinese.text), pair);
  return {
    english: { path: english.path, bytes: english.bytes, sha256: english.sha256 },
    chinese: { path: chinese.path, bytes: chinese.bytes, sha256: chinese.sha256 },
    codeBlockCount: extractCodeBlocks(english.text).length,
    inlineTokenCount: extractInlineTokens(english.text, pair).length,
    linkCount: extractLinks(english.text, pair).length,
    headingCount: extractHeadingLevels(english.text).length,
  };
}

function validatePublicDocumentation({ rootDir = DEFAULT_ROOT_DIR } = {}) {
  const root = fs.realpathSync(rootDir);
  const documents = DOCUMENT_PAIRS.map((pair) => validatePair(root, pair));
  return {
    schemaVersion: 1,
    passed: true,
    documents,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = validatePublicDocumentation(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `public documentation verified: ${result.documents.length} language pairs`}\n`);
  return result;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`public documentation validation failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export {
  DOCUMENT_PAIRS,
  MAX_DOCUMENT_BYTES,
  extractCodeBlocks,
  extractHeadingLevels,
  extractInlineTokens,
  extractLinks,
  validatePublicDocumentation,
};
