#!/usr/bin/env node

/**
 * Validate the public Runtime Image Docker contexts without contacting a
 * registry.  The public workflow performs the real `push: false` builds; this
 * guard proves that every Dockerfile and every local COPY source stays inside
 * the generated public tree and that no Agent/private context is referenced.
 */

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-export-policy.json");
const COMPONENTS = ["server", "frontend", "nginx", "bootstrap"];

function fail(message) {
  throw new Error(message);
}

function parseArgs(argv) {
  const args = { root: DEFAULT_ROOT, policy: DEFAULT_POLICY, component: "", json: false };
  const valueFlags = new Set(["--root-dir", "--policy", "--component"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") {
      args.json = true;
      continue;
    }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-public-runtime-contexts.mjs [--root-dir <dir>] [--policy <file>] [--component <name>] [--json]\n");
      process.exit(0);
    }
    if (!valueFlags.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--root-dir") args.root = path.resolve(value);
    else if (arg === "--policy") args.policy = path.resolve(value);
    else args.component = value;
  }
  if (args.component && !COMPONENTS.includes(args.component)) fail(`unsupported Runtime Image component: ${args.component}`);
  return args;
}

function readJson(file, label) {
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    fail(`cannot read ${label} ${file}: ${error.message}`);
  }
}

function relativePath(root, candidate, label, { allowRoot = false } = {}) {
  const resolvedRoot = path.resolve(root);
  const resolved = path.resolve(candidate);
  const relative = path.relative(resolvedRoot, resolved).split(path.sep).join("/");
  if ((!relative || relative === ".") && !allowRoot || relative.startsWith("../") || path.isAbsolute(relative)) {
    fail(`${label} escapes the public export root: ${candidate}`);
  }
  return relative;
}

function tokenizeCopyInstruction(line) {
  // Docker's COPY grammar is intentionally handled conservatively here.  A
  // quoted path is unquoted, flags are skipped, and wildcard sources are
  // checked by their parent directory below.  The actual Docker parser still
  // remains authoritative during the push:false build.
  const tokens = [];
  const matcher = /"([^"\\]*(?:\\.[^"\\]*)*)"|'([^'\\]*(?:\\.[^'\\]*)*)'|([^\s]+)/g;
  for (const match of line.matchAll(matcher)) tokens.push(match[1] ?? match[2] ?? match[3]);
  return tokens;
}

function copySources(dockerfileText) {
  const logicalLines = dockerfileText.replace(/\\\r?\n/g, " ").split(/\r?\n/);
  const sources = [];
  for (const rawLine of logicalLines) {
    const line = rawLine.trim();
    if (!/^COPY(?:\s|$)/i.test(line) || line.startsWith("#")) continue;
    const tokens = tokenizeCopyInstruction(line).slice(1);
    // A multi-stage COPY reads from an image stage, not from the host
    // context. Only host-local sources need containment and existence checks.
    if (tokens.some((token) => /^--from=/i.test(token))) continue;
    const filtered = tokens.filter((token) => !token.startsWith("--"));
    if (filtered.length < 2) fail(`COPY instruction has no source and destination: ${line}`);
    sources.push(...filtered.slice(0, -1));
  }
  return sources;
}

function assertSafeCopySource(source, contextRoot, root, component) {
  const normalized = source.replace(/^\.\//, "");
  if (!normalized || normalized === ".") return;
  if (normalized.startsWith("/") || normalized.includes("\\") || normalized.split("/").includes("..")) {
    fail(`${component} Dockerfile contains an escaping COPY source: ${source}`);
  }
  if (/^(?:agent|private|credentials?|secrets?)(?:\/|$)/i.test(normalized) || /(?:^|\/)agent(?:\/|$)/i.test(normalized)) {
    fail(`${component} Dockerfile references a private Agent source: ${source}`);
  }
  if (/(?:^|\/)(?:\.env(?:\..*)?|.*\.(?:pem|key|crt))$/i.test(normalized) || /(?:^|\/)ssl(?:\/|$)/i.test(normalized)) {
    fail(`${component} Dockerfile references host secret/certificate material: ${source}`);
  }

  const contextRelative = normalized.replace(/[*?\[]/g, "").replace(/\/$/, "");
  if (!contextRelative) return;
  const candidate = path.resolve(contextRoot, contextRelative);
  relativePath(root, candidate, `${component} COPY source`);
  // A wildcard may have no match in a fixture, but its containing directory
  // must exist.  Literal sources must be present in the generated tree.
  const hasWildcard = /[*?\[]/.test(normalized);
  if (hasWildcard) {
    if (!fs.existsSync(path.dirname(candidate))) fail(`${component} COPY wildcard parent is missing: ${source}`);
  } else if (!fs.existsSync(candidate)) {
    fail(`${component} COPY source is missing from the public tree: ${source}`);
  }
}

function validateComponent(root, descriptor) {
  const component = descriptor.component;
  const context = descriptor.context;
  const dockerfile = descriptor.dockerfile;
  if (typeof component !== "string" || typeof context !== "string" || typeof dockerfile !== "string") {
    fail("publicRuntimeImages descriptors must declare component, context, and dockerfile");
  }
  const contextRoot = path.resolve(root, context);
  const dockerfilePath = path.resolve(root, dockerfile);
  const contextRelative = relativePath(root, contextRoot, `${component} Docker context`, { allowRoot: true });
  const dockerfileRelative = relativePath(root, dockerfilePath, `${component} Dockerfile`);
  if (!fs.existsSync(contextRoot) || !fs.statSync(contextRoot).isDirectory()) fail(`${component} Docker context is missing: ${context}`);
  if (!fs.existsSync(dockerfilePath) || !fs.statSync(dockerfilePath).isFile()) fail(`${component} Dockerfile is missing: ${dockerfile}`);
  if (!dockerfileRelative.endsWith("Dockerfile")) fail(`${component} descriptor must point to a Dockerfile`);
  for (const required of descriptor.requiredPaths ?? []) {
    const requiredPath = path.resolve(root, required);
    relativePath(root, requiredPath, `${component} required path`);
    if (!fs.existsSync(requiredPath)) fail(`${component} required public build input is missing: ${required}`);
  }
  const text = fs.readFileSync(dockerfilePath, "utf8");
  for (const source of copySources(text)) assertSafeCopySource(source, contextRoot, root, component);
  if (component === "nginx" && fs.existsSync(path.join(contextRoot, "ssl"))) {
    fail("nginx Docker context must not contain an ssl directory");
  }
  return { component, context: contextRelative === "." ? "." : contextRelative, dockerfile: dockerfileRelative, copySources: copySources(text).length };
}

function validate(options) {
  const root = fs.realpathSync(options.root);
  const policy = readJson(options.policy, "public export policy");
  const descriptors = policy.publicRuntimeImages;
  if (!Array.isArray(descriptors) || descriptors.length !== COMPONENTS.length) {
    fail(`public export policy must declare exactly ${COMPONENTS.length} Runtime Image contexts`);
  }
  const seen = new Set();
  const selected = descriptors.filter((descriptor) => !options.component || descriptor.component === options.component);
  if (selected.length !== (options.component ? 1 : COMPONENTS.length)) fail(`public Runtime Image component descriptor is missing: ${options.component}`);
  const components = selected.map((descriptor) => {
    if (seen.has(descriptor.component)) fail(`duplicate public Runtime Image component: ${descriptor.component}`);
    seen.add(descriptor.component);
    return validateComponent(root, descriptor);
  });
  return { schemaVersion: 1, passed: true, root, components };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = validate(options);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `public Runtime Image contexts verified: ${result.components.map((item) => item.component).join(", ")}`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`public Runtime Image context verification failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

export { COMPONENTS, copySources, parseArgs, validate, validateComponent };
