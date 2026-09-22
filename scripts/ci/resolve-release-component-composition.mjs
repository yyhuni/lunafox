#!/usr/bin/env node

/**
 * Resolve the immutable component composition used by the release and upgrade
 * control planes.
 *
 * This module deliberately has no registry or GitHub dependency.  It hashes
 * the effective build input closure and only promotes an earlier artifact when
 * that fingerprint and the earlier verification evidence both match.  OCI
 * digests remain artifact identities; they are never used as a source-change
 * classifier.
 */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");

export const COMPOSITION_SCHEMA_VERSION = 1;
export const COMPOSITION_KIND = "lunafox.runtime-composition";
export const COMPOSITION_PLAN_KIND = "lunafox.runtime-composition-plan";
// The closure schema is versioned independently from the composition envelope.
// Version 2 records Unix permission bits so a mode-only Docker COPY change
// cannot be mistaken for an unchanged build input.
export const FINGERPRINT_SCHEMA_VERSION = 2;
export const FINGERPRINT_ALGORITHM = "sha256-canonical-json-v1";
export const COMPOSITION_DIGEST_ALGORITHM = "sha256-canonical-json-v1";
export const PLATFORM_SET = Object.freeze(["linux/amd64", "linux/arm64"]);
export const DISPOSITIONS = Object.freeze(["built", "reused"]);

export const DEFAULT_COMPONENTS = Object.freeze([
  Object.freeze({
    id: "runtime.server",
    kind: "runtime",
    name: "server",
    contextPath: ".",
    dockerfile: "server/Dockerfile",
    dockerignore: ".dockerignore",
  }),
  Object.freeze({
    id: "runtime.frontend",
    kind: "runtime",
    name: "frontend",
    contextPath: "frontend",
    dockerfile: "frontend/Dockerfile",
    dockerignore: "frontend/.dockerignore",
  }),
  Object.freeze({
    id: "runtime.nginx",
    kind: "runtime",
    name: "nginx",
    contextPath: "docker/nginx",
    dockerfile: "docker/nginx/Dockerfile",
    dockerignore: "",
  }),
  Object.freeze({
    id: "runtime.bootstrap",
    kind: "runtime",
    name: "bootstrap",
    contextPath: ".",
    dockerfile: "docker/bootstrap/Dockerfile",
    dockerignore: ".dockerignore",
  }),
  Object.freeze({
    id: "runtime.agent",
    kind: "runtime",
    name: "agent",
    contextPath: ".",
    dockerfile: "docker/agent/Dockerfile",
    dockerignore: ".dockerignore",
  }),
]);

const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;
const RELEASE_TAG_RE = /^v?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/;
const COMPONENT_ID_RE = /^(?:runtime\.[a-z][a-z0-9_-]*|agent\.[a-z][a-z0-9_-]*|engine\.[a-z][a-z0-9_.-]*\.(?:runtime|package))$/;
const OCI_DIGEST_REF_RE = /^(?:[A-Za-z0-9.-]+(?::\d+)?\/)?[A-Za-z0-9._/-]+@sha256:[a-f0-9]{64}$/;
function fail(message) {
  throw new Error(message);
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function canonicalize(value) {
  if (value === null || typeof value === "string" || typeof value === "boolean") return value;
  if (typeof value === "number") {
    if (!Number.isFinite(value)) fail("canonical JSON cannot contain a non-finite number");
    return value;
  }
  if (Array.isArray(value)) return value.map(canonicalize);
  if (!isPlainObject(value)) fail(`unsupported value in canonical JSON: ${typeof value}`);
  return Object.fromEntries(Object.keys(value).sort().map((key) => [key, canonicalize(value[key])]));
}

export function canonicalJson(value) {
  return JSON.stringify(canonicalize(value));
}

export function sha256Digest(value) {
  const bytes = typeof value === "string" || Buffer.isBuffer(value) ? value : canonicalJson(value);
  return `sha256:${crypto.createHash("sha256").update(bytes).digest("hex")}`;
}

function sha256File(filePath) {
  return sha256Digest(fs.readFileSync(filePath));
}

function normalizeRelative(relative, label, { allowDot = false } = {}) {
  const normalized = String(relative ?? "").replaceAll("\\", "/");
  assert(normalized && !path.posix.isAbsolute(normalized), `${label} must be a relative path`);
  if (allowDot && normalized === ".") return normalized;
  const parts = normalized.split("/");
  assert(parts.every((part) => part && part !== "." && part !== ".."), `${label} contains an unsafe path`);
  return parts.join("/");
}

function safeJoin(root, relative, label) {
  const normalized = normalizeRelative(relative, label);
  const resolvedRoot = path.resolve(root);
  const resolved = path.resolve(resolvedRoot, ...normalized.split("/"));
  assert(resolved === resolvedRoot || resolved.startsWith(`${resolvedRoot}${path.sep}`), `${label} escapes repository root`);
  return resolved;
}

function normalizeDigest(value, label) {
  assert(DIGEST_RE.test(String(value ?? "")), `${label} must be a sha256 digest`);
  return value;
}

function normalizePlatforms(platforms) {
  const values = platforms ?? PLATFORM_SET;
  assert(Array.isArray(values) && values.length > 0, "platforms must be a non-empty array");
  const normalized = [...new Set(values.map((value) => String(value).trim()))].sort();
  assert(normalized.every((value) => /^[a-z0-9]+\/[a-z0-9]+$/.test(value)), "platforms contain an invalid target");
  return normalized;
}

function normalizeMap(value, label) {
  if (value === undefined || value === null) return {};
  assert(isPlainObject(value), `${label} must be an object`);
  return Object.fromEntries(Object.keys(value).sort().map((key) => [key, String(value[key])]));
}

function normalizeEvidence(value, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const keys = ["image", "provenance", "sbom", "signature"];
  const normalized = {};
  for (const key of keys) {
    const item = value[key];
    if (item === undefined || item === null || item === "") continue;
    if (Array.isArray(item)) {
      assert(item.length > 0 && item.every((entry) => typeof entry === "string" && entry.trim()), `${label}.${key} must contain non-empty references`);
      normalized[key] = [...new Set(item.map((entry) => entry.trim()))].sort();
    } else {
      assert(typeof item === "string" && item.trim(), `${label}.${key} must be a non-empty reference`);
      normalized[key] = item.trim();
    }
  }
  for (const required of keys) assert(Object.hasOwn(normalized, required), `${label}.${required} is required`);
  return normalized;
}

function normalizeSourceRelease(value, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const tag = String(value.tag ?? "");
  assert(RELEASE_TAG_RE.test(tag), `${label}.tag is invalid`);
  const normalized = { tag };
  for (const key of ["publicMergeCommit", "workflowRun", "compositionDigest"]) {
    if (value[key] !== undefined && value[key] !== "") normalized[key] = String(value[key]);
  }
  if (normalized.compositionDigest !== undefined) normalizeDigest(normalized.compositionDigest, `${label}.compositionDigest`);
  return normalized;
}

function normalizeArtifact(value, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  const ref = String(value.ref ?? value.image ?? "");
  const digest = String(value.digest ?? "");
  assert(OCI_DIGEST_REF_RE.test(ref), `${label}.ref must be an immutable digest-qualified OCI reference`);
  assert(DIGEST_RE.test(digest), `${label}.digest must be a sha256 digest`);
  assert(ref.endsWith(`@${digest}`), `${label}.ref digest does not match ${label}.digest`);
  const normalized = { ref, digest };
  if (value.platforms !== undefined) normalized.platforms = normalizePlatforms(value.platforms);
  return normalized;
}

function parseDockerignore(content) {
  return String(content ?? "").split(/\r?\n/).map((line) => line.trim()).filter((line) => line && !line.startsWith("#")).map((line) => {
    const negated = line.startsWith("!");
    const pattern = (negated ? line.slice(1) : line).replaceAll("\\", "/").replace(/^\//, "").replace(/\/+$|\/$/g, "");
    return { pattern, negated };
  }).filter((rule) => rule.pattern);
}

function globRegex(pattern, matchBasename) {
  let expression = "";
  for (let index = 0; index < pattern.length; index += 1) {
    const character = pattern[index];
    if (character === "*") {
      if (pattern[index + 1] === "*") {
        index += 1;
        expression += ".*";
      } else {
        expression += "[^/]*";
      }
    } else if (character === "?") {
      expression += "[^/]";
    } else {
      expression += /[\\^$+?.()|{}[\]]/.test(character) ? `\\${character}` : character;
    }
  }
  return new RegExp(matchBasename ? `(?:^|/)${expression}(?:/.*)?$` : `^${expression}(?:/.*)?$`);
}

function ignoredByDockerRules(relative, rules) {
  let ignored = false;
  for (const rule of rules) {
    const hasSlash = rule.pattern.includes("/");
    if (globRegex(rule.pattern, !hasSlash).test(relative)) ignored = !rule.negated;
  }
  return ignored;
}

function walkFiles(root, ignoreRules = []) {
  const files = [];
  const visit = (directory, relativeDirectory = "") => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (entry.name === ".git") continue;
      const fullPath = path.join(directory, entry.name);
      const relative = relativeDirectory ? `${relativeDirectory}/${entry.name}` : entry.name;
      // Ignored dependency trees (for example frontend/node_modules) are not
      // part of the Docker context and may contain workspace symlinks. Do not
      // inspect them before applying the ignore rules.
      if (ignoredByDockerRules(relative, ignoreRules)) continue;
      // Docker archives symlinks as directory-entry metadata. Keep the link
      // target in the fingerprint without following it (following a workspace
      // link can escape the build context and make the digest host-dependent).
      if (entry.isSymbolicLink()) { files.push(fullPath); continue; }
      if (entry.isDirectory()) visit(fullPath, relative);
      else if (entry.isFile()) files.push(fullPath);
    }
  };
  visit(root);
  return files;
}

function readDockerfileInstructions(source, dockerfile) {
  const logicalLines = [];
  let pending = "";
  for (const rawLine of String(source).split(/\r?\n/)) {
    const line = rawLine.trimEnd();
    if (!pending && /^\s*#/.test(line)) continue;
    const continued = /(^|[^\\])\\$/.test(line);
    const body = continued ? line.slice(0, -1) : line;
    pending += `${pending ? " " : ""}${body.trim()}`;
    if (!continued) {
      if (pending) logicalLines.push(pending);
      pending = "";
    }
  }
  assert(!pending, `${dockerfile} ends with an unterminated continuation`);
  return logicalLines.map((line, index) => {
    const match = /^([A-Za-z]+)(?:\s+([\s\S]*))?$/.exec(line);
    assert(match, `${dockerfile} instruction ${index + 1} is malformed`);
    return { keyword: match[1].toUpperCase(), args: match[2] ?? "", line: index + 1 };
  });
}

function readDockerToken(input, start = 0) {
  let index = start;
  while (index < input.length && /\s/.test(input[index])) index += 1;
  if (index >= input.length) return null;
  const begin = index;
  let value = "";
  let quote = "";
  let escaped = false;
  for (; index < input.length; index += 1) {
    const character = input[index];
    if (escaped) {
      value += character;
      escaped = false;
      continue;
    }
    if (character === "\\") {
      escaped = true;
      continue;
    }
    if (quote) {
      if (character === quote) quote = "";
      else value += character;
      continue;
    }
    if (character === "'" || character === '"') {
      quote = character;
      continue;
    }
    if (/\s/.test(character)) break;
    value += character;
  }
  assert(!quote && !escaped, `malformed Dockerfile token near ${input.slice(begin)}`);
  return { value, end: index };
}

function dockerTokens(input) {
  const tokens = [];
  let index = 0;
  while (true) {
    const token = readDockerToken(input, index);
    if (!token) return tokens;
    tokens.push(token.value);
    index = token.end;
  }
}

function consumeDockerOptions(input) {
  const options = {};
  let index = 0;
  while (true) {
    const token = readDockerToken(input, index);
    if (!token || !token.value.startsWith("--")) break;
    index = token.end;
    const option = token.value;
    if (option === "--from") {
      const value = readDockerToken(input, index);
      assert(value, "Dockerfile --from option requires a value");
      options.from = value.value;
      index = value.end;
    } else if (option.startsWith("--from=")) {
      assert(option.slice("--from=".length), "Dockerfile --from option requires a value");
      options.from = option.slice("--from=".length);
    }
  }
  return { options, rest: input.slice(index).trim() };
}

function parseCopySources(instruction) {
  const { options, rest } = consumeDockerOptions(instruction.args);
  let sources;
  if (rest.startsWith("[")) {
    let parsed;
    try { parsed = JSON.parse(rest); }
    catch (error) { fail(`${instruction.keyword} at line ${instruction.line} has invalid JSON arguments: ${error.message}`); }
    assert(Array.isArray(parsed) && parsed.length >= 2 && parsed.every((value) => typeof value === "string"), `${instruction.keyword} at line ${instruction.line} must contain source and destination strings`);
    sources = parsed.slice(0, -1);
  } else {
    const tokens = dockerTokens(rest);
    assert(tokens.length >= 2, `${instruction.keyword} at line ${instruction.line} must contain a source and destination`);
    sources = tokens.slice(0, -1);
  }
  assert(sources.length > 0, `${instruction.keyword} at line ${instruction.line} has no source`);
  return { ...options, sources };
}

function parseArgAndEnvVariables(instructions, supplied) {
  // BuildKit injects these global ARGs even when a Dockerfile does not
  // redeclare them. Keep deterministic symbolic values in the fingerprint;
  // the target platform set is recorded separately and captures the complete
  // multi-architecture build identity.
  const variables = {
    BUILDPLATFORM: "linux/amd64",
    TARGETPLATFORM: "linux/amd64",
    BUILDOS: "linux",
    BUILDARCH: "amd64",
    BUILDVARIANT: "",
    TARGETOS: "linux",
    TARGETARCH: "amd64",
    TARGETVARIANT: "",
    ...normalizeMap(supplied, "buildArgs"),
  };
  for (const instruction of instructions) {
    if (instruction.keyword !== "ARG" && instruction.keyword !== "ENV") continue;
    const tokens = dockerTokens(instruction.args);
    if (!tokens.length) continue;
    if (instruction.keyword === "ARG") {
      const [name, defaultValue] = tokens[0].split(/=(.*)/s);
      assert(/^[A-Za-z_][A-Za-z0-9_]*$/.test(name), `invalid ARG name ${name}`);
      if (!Object.hasOwn(variables, name) && defaultValue !== undefined) variables[name] = defaultValue;
    } else {
      for (const token of tokens) {
        const match = /^([A-Za-z_][A-Za-z0-9_]*)(?:=(.*))?$/.exec(token);
        assert(match, `invalid ENV assignment ${token}`);
        if (match[2] !== undefined) variables[match[1]] = match[2];
      }
    }
  }
  return variables;
}

function interpolateDockerValue(value, variables, label) {
  return String(value).replace(/\$(?:\{([A-Za-z_][A-Za-z0-9_]*)\}|([A-Za-z_][A-Za-z0-9_]*))/g, (match, braced, bare) => {
    const name = braced ?? bare;
    assert(Object.hasOwn(variables, name), `${label} contains unresolved Dockerfile variable ${name}`);
    return variables[name];
  });
}

function parseDockerfileModel(root, dockerfile, suppliedBuildArgs) {
  const filePath = safeJoin(root, dockerfile, "dockerfile");
  assert(fs.existsSync(filePath) && fs.statSync(filePath).isFile(), `dockerfile does not exist: ${dockerfile}`);
  const instructions = readDockerfileInstructions(fs.readFileSync(filePath, "utf8"), dockerfile);
  const variables = parseArgAndEnvVariables(instructions, suppliedBuildArgs);
  const stages = [];
  let currentStage = null;
  for (const instruction of instructions) {
    if (instruction.keyword === "FROM") {
      const { rest } = consumeDockerOptions(instruction.args);
      const tokens = dockerTokens(rest);
      assert(tokens.length >= 1, `${dockerfile} FROM at line ${instruction.line} is missing an image`);
      const rawRef = tokens[0];
      const ref = interpolateDockerValue(rawRef, variables, `${dockerfile} FROM at line ${instruction.line}`);
      let alias = "";
      if (tokens.length > 1) {
        assert(tokens.length === 3 && tokens[1].toUpperCase() === "AS" && /^[A-Za-z_][A-Za-z0-9_.-]*$/.test(tokens[2]), `${dockerfile} FROM at line ${instruction.line} has an invalid stage alias`);
        alias = tokens[2];
      }
      currentStage = { index: stages.length, alias, ref, rawRef, instructions: [] };
      stages.push(currentStage);
    } else {
      // Docker permits global ARG declarations before the first FROM. They are
      // consumed while interpolating every later base-image reference, but are
      // not instructions that belong to a build stage.
      if (!currentStage && instruction.keyword === "ARG") continue;
      assert(currentStage, `${dockerfile} instruction ${instruction.line} appears before the first FROM`);
      currentStage.instructions.push(instruction);
    }
  }
  assert(stages.length > 0, `${dockerfile} must declare at least one base image`);
  const aliases = new Map(stages.filter((stage) => stage.alias).map((stage) => [stage.alias.toLowerCase(), stage.index]));
  return { instructions, variables, stages, aliases };
}

function stageIndexForReference(reference, model) {
  if (/^\d+$/.test(reference)) {
    const index = Number(reference);
    return index < model.stages.length ? index : null;
  }
  return model.aliases.get(String(reference).toLowerCase()) ?? null;
}

function normalizeNamedContexts(root, supplied) {
  if (supplied === undefined || supplied === null) return {};
  assert(isPlainObject(supplied), "namedContexts must be an object");
  return Object.fromEntries(Object.keys(supplied).sort().map((name) => {
    assert(/^[A-Za-z_][A-Za-z0-9_.-]*$/.test(name), `namedContexts contains an invalid name: ${name}`);
    const raw = supplied[name];
    const descriptor = typeof raw === "string" ? { path: raw } : raw;
    assert(isPlainObject(descriptor), `namedContexts.${name} must be a path or object`);
    const contextPath = normalizeRelative(descriptor.path, `namedContexts.${name}.path`, { allowDot: true });
    const dockerignore = descriptor.dockerignore ? normalizeRelative(descriptor.dockerignore, `namedContexts.${name}.dockerignore`) : "";
    return [name, { contextPath, dockerignore }];
  }));
}

function contextDescriptor(root, contextPath, dockerignore, label) {
  const normalizedContextPath = normalizeRelative(contextPath || ".", `${label}.contextPath`, { allowDot: true });
  const contextRoot = normalizedContextPath === "." ? path.resolve(root) : safeJoin(root, normalizedContextPath, `${label}.contextPath`);
  assert(fs.existsSync(contextRoot) && fs.statSync(contextRoot).isDirectory(), `build context does not exist: ${contextPath}`);
  let ignorePath = "";
  if (dockerignore) {
    ignorePath = safeJoin(root, dockerignore, `${label}.dockerignore`);
    assert(fs.existsSync(ignorePath) && fs.statSync(ignorePath).isFile(), `${label}.dockerignore does not exist: ${dockerignore}`);
  } else {
    const candidate = path.join(contextRoot, ".dockerignore");
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) ignorePath = candidate;
  }
  const ignoreRules = ignorePath ? parseDockerignore(fs.readFileSync(ignorePath, "utf8")) : [];
  return { contextPath: normalizedContextPath, contextRoot, ignorePath, ignoreRules };
}

function contextInventory(root, descriptor, virtualPrefix = "") {
  const files = [];
  const byRelativePath = new Map();
  for (const fullPath of walkFiles(descriptor.contextRoot, descriptor.ignoreRules)) {
    const relative = path.relative(descriptor.contextRoot, fullPath).split(path.sep).join("/");
    if (ignoredByDockerRules(relative, descriptor.ignoreRules)) continue;
    const pathLabel = virtualPrefix ? `${virtualPrefix}/${relative}` : path.relative(root, fullPath).split(path.sep).join("/");
    const stat = fs.lstatSync(fullPath);
    const symlink = stat.isSymbolicLink() ? fs.readlinkSync(fullPath) : undefined;
    const entry = {
      path: pathLabel,
      digest: symlink === undefined ? sha256File(fullPath) : sha256Digest(`symlink:${symlink}`),
      size: symlink === undefined ? stat.size : symlink.length,
      mode: stat.mode & 0o7777,
      ...(symlink === undefined ? {} : { symlink }),
    };
    files.push(entry);
    byRelativePath.set(relative, { ...entry, fullPath });
  }
  return { files, byRelativePath };
}

function sourceGlobRegex(pattern) {
  let expression = "";
  for (let index = 0; index < pattern.length; index += 1) {
    const character = pattern[index];
    if (character === "*") {
      if (pattern[index + 1] === "*") { index += 1; expression += ".*"; }
      else expression += "[^/]*";
    } else if (character === "?") expression += "[^/]";
    else if (character === "[") {
      const end = pattern.indexOf("]", index + 1);
      assert(end > index + 1, `invalid source glob ${pattern}`);
      const body = pattern.slice(index + 1, end);
      expression += `[${body.replaceAll("\\", "\\\\")}]`;
      index = end;
    } else expression += /[\\^$+?.()|{}]/.test(character) ? `\\${character}` : character;
  }
  return new RegExp(`^${expression}(?:/.*)?$`);
}

function normalizeContextSource(source, label) {
  const normalized = String(source).replaceAll("\\", "/").replace(/^\.\//, "");
  assert(normalized && !normalized.startsWith("/") && !normalized.split("/").includes(".."), `${label} is unsafe: ${source}`);
  return normalized;
}

function resolveContextSource(source, descriptor, inventory, label, records) {
  const normalized = normalizeContextSource(source, label);
  const hasGlob = /[*?[]/.test(normalized);
  const matches = [];
  if (normalized === "." || normalized === "") {
    matches.push(...inventory.byRelativePath.values());
  } else if (hasGlob) {
    const regex = sourceGlobRegex(normalized);
    for (const [relative, entry] of inventory.byRelativePath) if (regex.test(relative)) matches.push(entry);
  } else {
    const exact = inventory.byRelativePath.get(normalized);
    if (exact) matches.push(exact);
    else {
      const prefix = `${normalized.replace(/\/$/, "")}/`;
      for (const [relative, entry] of inventory.byRelativePath) if (relative.startsWith(prefix)) matches.push(entry);
    }
  }
  assert(matches.length > 0, `${label} does not resolve to an effective build-context file: ${source}`);
  for (const entry of matches) {
    records.set(entry.path, {
      path: entry.path,
      digest: entry.digest,
      size: entry.size,
      mode: entry.mode,
      ...(entry.symlink === undefined ? {} : { symlink: entry.symlink }),
    });
  }
}

function parseMountSources(args) {
  const mounts = [];
  const pattern = /--mount(?:=|\s+)([^\s]+)/g;
  for (const match of String(args).matchAll(pattern)) {
    const values = Object.fromEntries(match[1].split(",").map((item) => {
      const separator = item.indexOf("=");
      return separator < 0 ? [item, ""] : [item.slice(0, separator), item.slice(separator + 1)];
    }));
    if (values.type === "bind") mounts.push(values);
  }
  return mounts;
}

function collectEffectiveInputs(root, contextPath, dockerfile, dockerignore, spec) {
  const model = parseDockerfileModel(root, dockerfile, spec.buildArgs);
  const mainContext = contextDescriptor(root, contextPath, dockerignore, "main context");
  const namedContexts = normalizeNamedContexts(root, spec.namedContexts);
  const contextCache = new Map();
  const getContext = (name, descriptor, virtualPrefix) => {
    const key = `${name}:${descriptor.contextPath}:${descriptor.dockerignore}:${virtualPrefix}`;
    if (!contextCache.has(key)) contextCache.set(key, { descriptor, inventory: contextInventory(root, descriptor, virtualPrefix) });
    return contextCache.get(key);
  };
  const main = getContext("main", mainContext, "");
  const named = new Map(Object.entries(namedContexts).map(([name, value]) => [name, getContext(name, contextDescriptor(root, value.contextPath, value.dockerignore, `named context ${name}`), `named-contexts/${name}`)]));
  // Start with the Dockerfile and ignore file. Context files are added only
  // when a reachable COPY/ADD/bind mount resolves them; hashing the whole
  // context would make unrelated edits look like deployable changes.
  const records = new Map();
  if (main.descriptor.ignorePath) {
    const ignorePath = path.relative(root, main.descriptor.ignorePath).split(path.sep).join("/");
    const stat = fs.lstatSync(main.descriptor.ignorePath);
    records.set(ignorePath, { path: ignorePath, digest: sha256File(main.descriptor.ignorePath), size: stat.size, mode: stat.mode & 0o7777 });
  }
  const dockerfileRelative = normalizeRelative(dockerfile, "dockerfile");
  const dockerfilePath = safeJoin(root, dockerfileRelative, "dockerfile");
  const dockerfileStat = fs.lstatSync(dockerfilePath);
  records.set(dockerfileRelative, { path: dockerfileRelative, digest: sha256File(dockerfilePath), size: dockerfileStat.size, mode: dockerfileStat.mode & 0o7777 });
  const reachable = new Set();
  const externalBaseRefs = new Set();
  const visited = new Set();
  const visitStage = (index) => {
    assert(Number.isInteger(index) && index >= 0 && index < model.stages.length, `Dockerfile stage reference is invalid: ${index}`);
    if (visited.has(index)) return;
    visited.add(index);
    const stage = model.stages[index];
    const baseStage = stageIndexForReference(stage.ref, model);
    if (baseStage !== null) visitStage(baseStage);
    else externalBaseRefs.add(stage.ref);
    for (const instruction of stage.instructions) {
      if (instruction.keyword === "COPY" || instruction.keyword === "ADD") {
        const parsed = parseCopySources(instruction);
        if (parsed.from !== undefined) {
          const from = interpolateDockerValue(parsed.from, model.variables, `${dockerfile} ${instruction.keyword} at line ${instruction.line}`);
          const sourceStage = stageIndexForReference(from, model);
          if (sourceStage !== null) visitStage(sourceStage);
          else if (named.has(from)) {
            for (const source of parsed.sources) resolveContextSource(interpolateDockerValue(source, model.variables, `${dockerfile} ${instruction.keyword} at line ${instruction.line}`), named.get(from).descriptor, named.get(from).inventory, `${dockerfile} ${instruction.keyword} at line ${instruction.line}`, records);
          } else {
            externalBaseRefs.add(from);
          }
        } else {
          if (instruction.keyword === "ADD" && parsed.sources.some((source) => /^(?:https?|git|ssh):\/\//i.test(source))) fail(`${dockerfile} ADD at line ${instruction.line} uses a remote source; provide a pinned generated input instead`);
          for (const source of parsed.sources) resolveContextSource(interpolateDockerValue(source, model.variables, `${dockerfile} ${instruction.keyword} at line ${instruction.line}`), main.descriptor, main.inventory, `${dockerfile} ${instruction.keyword} at line ${instruction.line}`, records);
        }
      } else if (instruction.keyword === "RUN") {
        for (const mount of parseMountSources(instruction.args)) {
          const from = mount.from ? interpolateDockerValue(mount.from, model.variables, `${dockerfile} RUN mount at line ${instruction.line}`) : "";
          const source = interpolateDockerValue(mount.source || ".", model.variables, `${dockerfile} RUN mount at line ${instruction.line}`);
          if (from) {
            const sourceStage = stageIndexForReference(from, model);
            if (sourceStage !== null) visitStage(sourceStage);
            else if (named.has(from)) resolveContextSource(source, named.get(from).descriptor, named.get(from).inventory, `${dockerfile} RUN mount at line ${instruction.line}`, records);
            else externalBaseRefs.add(from);
          } else resolveContextSource(source, main.descriptor, main.inventory, `${dockerfile} RUN mount at line ${instruction.line}`, records);
        }
      }
    }
    reachable.add(index);
  };
  const target = spec.target === undefined || spec.target === "" ? model.stages.length - 1 : stageIndexForReference(String(spec.target), model);
  assert(target !== null, `${dockerfile} target stage does not exist: ${spec.target}`);
  visitStage(target);
  for (const context of named.values()) {
    if (!context.descriptor.ignorePath) continue;
    const ignorePath = path.relative(root, context.descriptor.ignorePath).split(path.sep).join("/");
    const stat = fs.lstatSync(context.descriptor.ignorePath);
    records.set(ignorePath, { path: ignorePath, digest: sha256File(context.descriptor.ignorePath), size: stat.size, mode: stat.mode & 0o7777 });
  }
  return {
    files: [...records.values()].sort((a, b) => a.path.localeCompare(b.path)),
    model,
    externalBaseRefs,
    reachable,
    namedContexts,
  };
}

function resolveBaseImages(refs, supplied) {
  const mapping = normalizeMap(supplied, "baseImageIdentities");
  return [...refs].sort().map((ref) => {
    const identity = mapping[ref] ?? (DIGEST_RE.test(ref.slice(ref.lastIndexOf("@") + 1)) ? ref : "");
    return { ref, identity, resolved: Boolean(identity) };
  });
}

function normalizeGeneratedInputs(root, values) {
  if (values === undefined || values === null) return [];
  assert(Array.isArray(values), "generatedInputs must be an array");
  return values.map((entry, index) => {
    assert(isPlainObject(entry), `generatedInputs[${index}] must be an object`);
    const name = normalizeRelative(entry.name, `generatedInputs[${index}].name`);
    const file = entry.file ? safeJoin(root, entry.file, `generatedInputs[${index}].file`) : "";
    const digest = entry.digest ?? (file ? sha256File(file) : "");
    normalizeDigest(digest, `generatedInputs[${index}].digest`);
    return { name, digest };
  }).sort((a, b) => a.name.localeCompare(b.name));
}

const INPUT_CLOSURE_KEYS = Object.freeze([
  "schemaVersion", "componentId", "kind", "contextPath", "dockerfile", "dockerignore",
  "files", "namedContexts", "buildArgs", "platforms", "baseImages", "baseImagesResolved",
  "builderPolicy", "generatedInputs",
]);

function normalizeInputClosure(value, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  assertAllowedKeys(value, new Set(INPUT_CLOSURE_KEYS), label);
  for (const key of INPUT_CLOSURE_KEYS) assert(Object.hasOwn(value, key), `${label}.${key} is required`);
  assert(value.schemaVersion === FINGERPRINT_SCHEMA_VERSION, `${label}.schemaVersion is unsupported`);
  assert(COMPONENT_ID_RE.test(String(value.componentId ?? "")), `${label}.componentId is invalid`);
  assert(["runtime", "agent", "engine"].includes(value.kind), `${label}.kind is unsupported`);
  assert(value.kind === String(value.componentId).split(".")[0], `${label}.kind must match componentId`);
  assert(typeof value.contextPath === "string" && value.contextPath, `${label}.contextPath is required`);
  assert(typeof value.dockerfile === "string" && value.dockerfile, `${label}.dockerfile is required`);
  assert(typeof value.dockerignore === "string", `${label}.dockerignore is required`);
  assert(Array.isArray(value.files), `${label}.files must be an array`);
  for (const [index, file] of value.files.entries()) {
    assert(isPlainObject(file), `${label}.files[${index}] must be an object`);
    assertAllowedKeys(file, new Set(["path", "digest", "size", "mode", "symlink"]), `${label}.files[${index}]`);
    assert(typeof file.path === "string" && file.path, `${label}.files[${index}].path is required`);
    normalizeDigest(file.digest, `${label}.files[${index}].digest`);
    assert(Number.isSafeInteger(file.size) && file.size >= 0, `${label}.files[${index}].size is invalid`);
    assert(Number.isSafeInteger(file.mode) && file.mode >= 0 && file.mode <= 0o7777, `${label}.files[${index}].mode is invalid`);
    if (file.symlink !== undefined) assert(typeof file.symlink === "string", `${label}.files[${index}].symlink is invalid`);
  }
  assert(isPlainObject(value.namedContexts), `${label}.namedContexts must be an object`);
  for (const [name, context] of Object.entries(value.namedContexts)) {
    assert(isPlainObject(context), `${label}.namedContexts.${name} must be an object`);
    assertAllowedKeys(context, new Set(["contextPath", "dockerignore"]), `${label}.namedContexts.${name}`);
    assert(typeof context.contextPath === "string" && context.contextPath, `${label}.namedContexts.${name}.contextPath is required`);
    assert(typeof context.dockerignore === "string", `${label}.namedContexts.${name}.dockerignore is required`);
  }
  assert(isPlainObject(value.buildArgs), `${label}.buildArgs must be an object`);
  for (const [key, item] of Object.entries(value.buildArgs)) assert(typeof item === "string", `${label}.buildArgs.${key} must be a string`);
  const normalizedPlatforms = normalizePlatforms(value.platforms);
  assert(JSON.stringify(value.platforms) === JSON.stringify(normalizedPlatforms), `${label}.platforms must be unique and sorted`);
  assert(Array.isArray(value.baseImages), `${label}.baseImages must be an array`);
  for (const [index, image] of value.baseImages.entries()) {
    assert(isPlainObject(image), `${label}.baseImages[${index}] must be an object`);
    assertAllowedKeys(image, new Set(["ref", "identity", "resolved"]), `${label}.baseImages[${index}]`);
    assert(typeof image.ref === "string" && image.ref, `${label}.baseImages[${index}].ref is required`);
    assert(typeof image.identity === "string", `${label}.baseImages[${index}].identity is required`);
    assert(typeof image.resolved === "boolean", `${label}.baseImages[${index}].resolved is required`);
  }
  assert(typeof value.baseImagesResolved === "boolean", `${label}.baseImagesResolved is required`);
  assert(value.baseImagesResolved === value.baseImages.every((image) => image.resolved), `${label}.baseImagesResolved does not match baseImages`);
  assert(isPlainObject(value.builderPolicy), `${label}.builderPolicy must be an object`);
  canonicalize(value.builderPolicy);
  assert(Array.isArray(value.generatedInputs), `${label}.generatedInputs must be an array`);
  for (const [index, input] of value.generatedInputs.entries()) {
    assert(isPlainObject(input), `${label}.generatedInputs[${index}] must be an object`);
    assertAllowedKeys(input, new Set(["name", "digest"]), `${label}.generatedInputs[${index}]`);
    assert(typeof input.name === "string" && input.name, `${label}.generatedInputs[${index}].name is required`);
    normalizeDigest(input.digest, `${label}.generatedInputs[${index}].digest`);
  }
  return canonicalize(value);
}

export function buildInputFingerprint(root, spec) {
  assert(isPlainObject(spec), "component specification must be an object");
  const id = String(spec.id ?? "");
  assert(COMPONENT_ID_RE.test(id), `component id is invalid: ${id}`);
  const kind = String(spec.kind ?? id.split(".")[0]);
  assert(["runtime", "agent", "engine"].includes(kind), `component ${id} has unsupported kind ${kind}`);
  const contextPath = normalizeRelative(spec.contextPath ?? ".", `${id}.contextPath`, { allowDot: true });
  const dockerfile = normalizeRelative(spec.dockerfile, `${id}.dockerfile`);
  const dockerignore = spec.dockerignore ? normalizeRelative(spec.dockerignore, `${id}.dockerignore`) : "";
  const effective = collectEffectiveInputs(root, contextPath, dockerfile, dockerignore, spec);
  const files = effective.files;
  const baseImages = resolveBaseImages(effective.externalBaseRefs, spec.baseImageIdentities);
  const generatedInputs = normalizeGeneratedInputs(root, spec.generatedInputs);
  const inputs = {
    schemaVersion: FINGERPRINT_SCHEMA_VERSION,
    componentId: id,
    kind,
    contextPath,
    dockerfile,
    dockerignore,
    files,
    namedContexts: Object.fromEntries(Object.entries(effective.namedContexts).map(([name, descriptor]) => [name, descriptor])),
    buildArgs: normalizeMap(spec.buildArgs, `${id}.buildArgs`),
    platforms: normalizePlatforms(spec.platforms),
    baseImages,
    baseImagesResolved: baseImages.every((entry) => entry.resolved),
    builderPolicy: canonicalize(spec.builderPolicy ?? {}),
    generatedInputs,
  };
  return {
    version: FINGERPRINT_SCHEMA_VERSION,
    algorithm: FINGERPRINT_ALGORITHM,
    digest: sha256Digest(inputs),
    inputs,
  };
}

function normalizeInputFingerprint(value, label) {
  assert(isPlainObject(value), `${label} must be an object`);
  assertAllowedKeys(value, new Set(["version", "algorithm", "digest", "baseImagesResolved", "inputs"]), label);
  assert(value.version === FINGERPRINT_SCHEMA_VERSION, `${label}.version is unsupported`);
  assert(value.algorithm === FINGERPRINT_ALGORITHM, `${label}.algorithm is unsupported`);
  normalizeDigest(value.digest, `${label}.digest`);
  assert(typeof value.baseImagesResolved === "boolean", `${label}.baseImagesResolved is required`);
  const inputs = normalizeInputClosure(value.inputs, `${label}.inputs`);
  assert(value.baseImagesResolved === inputs.baseImagesResolved, `${label}.baseImagesResolved does not match inputs`);
  const expected = sha256Digest(inputs);
  assert(expected === value.digest, `${label}.digest does not match inputs`);
  return {
    version: value.version,
    algorithm: value.algorithm,
    digest: value.digest,
    baseImagesResolved: value.baseImagesResolved,
    inputs,
  };
}

function normalizeComponent(component, index, options = {}) {
  const label = `components[${index}]`;
  assert(isPlainObject(component), `${label} must be an object`);
  const id = String(component.id ?? "");
  assert(COMPONENT_ID_RE.test(id), `${label}.id is invalid: ${id}`);
  const kind = String(component.kind ?? "");
  assert(["runtime", "agent", "engine"].includes(kind), `${label}.kind is unsupported`);
  const idKind = id.split(".")[0];
  assert(kind === idKind, `${label}.kind must match its component id`);
  const name = String(component.name ?? id.split(".").at(-1));
  assert(name && /^[a-z][a-z0-9_.-]*$/.test(name), `${label}.name is invalid`);
  const disposition = String(component.disposition ?? "");
  assert(DISPOSITIONS.includes(disposition), `${label}.disposition must be built or reused`);
  const inputFingerprint = normalizeInputFingerprint(component.inputFingerprint, `${label}.inputFingerprint`);
  const artifact = normalizeArtifact(component.artifact, `${label}.artifact`);
  const sourceRelease = normalizeSourceRelease(component.sourceRelease, `${label}.sourceRelease`);
  const evidence = normalizeEvidence(component.evidence, `${label}.evidence`);
  if (disposition === "reused") {
    assert(sourceRelease.tag !== options.releaseTag, `${label} reused artifact must point to its original release`);
    assert(sourceRelease.tag, `${label} reused artifact source release is required`);
    assert(sourceRelease.compositionDigest, `${label} reused artifact source composition digest is required`);
  } else {
    assert(sourceRelease.tag === options.releaseTag, `${label} built component must originate from the current release`);
  }
  return { id, kind, name, inputFingerprint, artifact, disposition, sourceRelease, evidence };
}

/**
 * The manifest must bind the stable deployable inventory, while the final
 * composition also binds the exact manifest bytes. Keeping that reverse
 * binding outside this payload prevents a manifest/composition hash cycle.
 */
export function compositionCorePayload(composition) {
  const { manifestBinding: _binding, compositionDigest: _digest, ...payload } = composition;
  return payload;
}

function assertAllowedKeys(value, allowed, label) {
  const unexpected = Object.keys(value).filter((key) => !allowed.has(key));
  assert(unexpected.length === 0, `${label} contains unknown fields: ${unexpected.sort().join(", ")}`);
}

export function validateComposition(composition, options = {}) {
  assert(isPlainObject(composition), "runtime composition must be a JSON object");
  assertAllowedKeys(composition, new Set([
    "schemaVersion", "kind", "releaseTag", "sourceRevisionDigest", "publicMergeCommit",
    "components", "capabilities", "compositionDigest", "manifestBinding",
  ]), "runtime composition");
  assert(composition.schemaVersion === COMPOSITION_SCHEMA_VERSION, "runtime composition schemaVersion is unsupported");
  assert(composition.kind === COMPOSITION_KIND, "runtime composition kind is invalid");
  const releaseTag = String(composition.releaseTag ?? "");
  assert(RELEASE_TAG_RE.test(releaseTag), "runtime composition releaseTag is invalid");
  if (composition.sourceRevisionDigest !== undefined) normalizeDigest(composition.sourceRevisionDigest, "runtime composition sourceRevisionDigest");
  if (composition.publicMergeCommit !== undefined) assert(/^[0-9a-f]{40}$/.test(composition.publicMergeCommit), "runtime composition publicMergeCommit is invalid");
  assert(Array.isArray(composition.components) && composition.components.length > 0, "runtime composition must contain components");
  const seen = new Set();
  const components = composition.components.map((component, index) => {
    const normalized = normalizeComponent(component, index, { releaseTag });
    assert(!seen.has(normalized.id), `runtime composition contains duplicate component ${normalized.id}`);
    seen.add(normalized.id);
    return normalized;
  }).sort((a, b) => a.id.localeCompare(b.id));
  if (options.expectedComponentIds) {
    const expected = [...new Set(options.expectedComponentIds)].sort();
    assert(JSON.stringify(expected) === JSON.stringify(components.map((component) => component.id)), "runtime composition component inventory is incomplete or contains unknown components");
  }
  const capabilities = composition.capabilities ?? {};
  assert(isPlainObject(capabilities), "runtime composition capabilities must be an object");
  for (const [key, value] of Object.entries(capabilities)) assert(typeof value === "boolean" || typeof value === "string" || typeof value === "number", `runtime composition capability ${key} must be scalar`);
  if (composition.manifestBinding !== undefined) {
    assert(isPlainObject(composition.manifestBinding), "runtime composition manifestBinding must be an object");
    assertAllowedKeys(composition.manifestBinding, new Set(["manifestDigest"]), "runtime composition manifestBinding");
    normalizeDigest(composition.manifestBinding.manifestDigest, "runtime composition manifestBinding.manifestDigest");
  } else if (options.requireManifestBinding) {
    fail("runtime composition manifestBinding is required");
  }
  normalizeDigest(composition.compositionDigest, "runtime composition compositionDigest");
  const normalizedCore = {
    schemaVersion: COMPOSITION_SCHEMA_VERSION,
    kind: COMPOSITION_KIND,
    releaseTag,
    ...(composition.sourceRevisionDigest ? { sourceRevisionDigest: composition.sourceRevisionDigest } : {}),
    ...(composition.publicMergeCommit ? { publicMergeCommit: composition.publicMergeCommit } : {}),
    components,
    capabilities: canonicalize(capabilities),
  };
  assert(composition.compositionDigest === sha256Digest(compositionCorePayload(normalizedCore)), "runtime composition compositionDigest does not match canonical core payload");
  return {
    ...normalizedCore,
    ...(composition.manifestBinding ? { manifestBinding: canonicalize(composition.manifestBinding) } : {}),
    compositionDigest: composition.compositionDigest,
  };
}

export function bindCompositionToManifest(composition, manifestDigest) {
  const normalized = validateComposition(composition);
  normalizeDigest(manifestDigest, "manifest digest");
  const bound = {
    ...normalized,
    manifestBinding: { manifestDigest },
  };
  // bindCompositionToManifest must never recalculate the core digest: the
  // manifest records that digest and recalculation here would reintroduce a
  // circular identity dependency.
  return validateComposition(bound, { requireManifestBinding: true });
}

function compositionPlanPayload(plan) {
  const { planDigest: _digest, ...payload } = plan;
  return payload;
}

function normalizePlanComponent(component, index, options = {}) {
  const label = `components[${index}]`;
  assert(isPlainObject(component), `${label} must be an object`);
  assertAllowedKeys(component, new Set([
    "id", "kind", "name", "inputFingerprint", "disposition", "sourceRelease", "artifact", "evidence",
  ]), label);
  const id = String(component.id ?? "");
  assert(COMPONENT_ID_RE.test(id), `${label}.id is invalid: ${id}`);
  const kind = String(component.kind ?? "");
  assert(["runtime", "agent", "engine"].includes(kind) && kind === id.split(".")[0], `${label}.kind must match its component id`);
  const name = String(component.name ?? id.split(".").at(-1));
  assert(name && /^[a-z][a-z0-9_.-]*$/.test(name), `${label}.name is invalid`);
  const disposition = String(component.disposition ?? "");
  assert(DISPOSITIONS.includes(disposition), `${label}.disposition must be built or reused`);
  const inputFingerprint = normalizeInputFingerprint(component.inputFingerprint, `${label}.inputFingerprint`);
  const sourceRelease = normalizeSourceRelease(component.sourceRelease, `${label}.sourceRelease`);
  const hasArtifact = component.artifact !== undefined;
  const hasEvidence = component.evidence !== undefined;
  if (disposition === "reused") {
    assert(hasArtifact && hasEvidence, `${label} reused component requires immutable artifact and evidence`);
    assert(sourceRelease.tag !== options.releaseTag, `${label} reused artifact must point to its original release`);
    return {
      id, kind, name, inputFingerprint, disposition, sourceRelease,
      artifact: normalizeArtifact(component.artifact, `${label}.artifact`),
      evidence: normalizeEvidence(component.evidence, `${label}.evidence`),
    };
  }
  assert(!hasArtifact && !hasEvidence, `${label} built component must not claim artifact evidence before build completion`);
  assert(sourceRelease.tag === options.releaseTag, `${label} built component must originate from the current release`);
  return { id, kind, name, inputFingerprint, disposition, sourceRelease };
}

export function validateCompositionPlan(plan, options = {}) {
  assert(isPlainObject(plan), "runtime composition plan must be a JSON object");
  assertAllowedKeys(plan, new Set([
    "schemaVersion", "kind", "releaseTag", "sourceRevisionDigest", "publicMergeCommit",
    "components", "capabilities", "planDigest",
  ]), "runtime composition plan");
  assert(plan.schemaVersion === COMPOSITION_SCHEMA_VERSION, "runtime composition plan schemaVersion is unsupported");
  assert(plan.kind === COMPOSITION_PLAN_KIND, "runtime composition plan kind is invalid");
  const releaseTag = String(plan.releaseTag ?? "");
  assert(RELEASE_TAG_RE.test(releaseTag), "runtime composition plan releaseTag is invalid");
  if (plan.sourceRevisionDigest !== undefined) normalizeDigest(plan.sourceRevisionDigest, "runtime composition plan sourceRevisionDigest");
  if (plan.publicMergeCommit !== undefined) assert(/^[0-9a-f]{40}$/.test(plan.publicMergeCommit), "runtime composition plan publicMergeCommit is invalid");
  assert(Array.isArray(plan.components) && plan.components.length > 0, "runtime composition plan must contain components");
  const seen = new Set();
  const components = plan.components.map((component, index) => {
    const normalized = normalizePlanComponent(component, index, { releaseTag });
    assert(!seen.has(normalized.id), `runtime composition plan contains duplicate component ${normalized.id}`);
    seen.add(normalized.id);
    return normalized;
  }).sort((left, right) => left.id.localeCompare(right.id));
  if (options.expectedComponentIds) {
    const expected = [...new Set(options.expectedComponentIds)].sort();
    assert(JSON.stringify(expected) === JSON.stringify(components.map((component) => component.id)), "runtime composition plan component inventory is incomplete or contains unknown components");
  }
  const capabilities = plan.capabilities ?? {};
  assert(isPlainObject(capabilities), "runtime composition plan capabilities must be an object");
  for (const [key, value] of Object.entries(capabilities)) assert(typeof value === "boolean" || typeof value === "string" || typeof value === "number", `runtime composition plan capability ${key} must be scalar`);
  normalizeDigest(plan.planDigest, "runtime composition plan planDigest");
  const normalized = {
    schemaVersion: COMPOSITION_SCHEMA_VERSION,
    kind: COMPOSITION_PLAN_KIND,
    releaseTag,
    ...(plan.sourceRevisionDigest ? { sourceRevisionDigest: plan.sourceRevisionDigest } : {}),
    ...(plan.publicMergeCommit ? { publicMergeCommit: plan.publicMergeCommit } : {}),
    components,
    capabilities: canonicalize(capabilities),
    planDigest: plan.planDigest,
  };
  assert(normalized.planDigest === sha256Digest(compositionPlanPayload(normalized)), "runtime composition plan planDigest does not match canonical payload");
  return normalized;
}

function priorComponentMap(previous) {
  if (!previous) return { components: new Map(), compositionDigest: "" };
  const validated = validateComposition(previous);
  return {
    components: new Map(validated.components.map((component) => [component.id, component])),
    compositionDigest: validated.compositionDigest,
  };
}

function assertReusableEvidence(component, currentReleaseTag) {
  normalizeArtifact(component.artifact, `${component.id}.artifact`);
  normalizeSourceRelease(component.sourceRelease, `${component.id}.sourceRelease`);
  normalizeEvidence(component.evidence, `${component.id}.evidence`);
  assert(component.sourceRelease.tag !== currentReleaseTag, `${component.id} reuse source release must differ from current release`);
  assert(component.disposition === "built" || component.disposition === "reused", `${component.id} has invalid prior disposition`);
}

export function resolveCompositionPlan({
  root = DEFAULT_ROOT,
  releaseTag,
  sourceRevisionDigest,
  publicMergeCommit,
  components,
  previous = null,
  capabilities = {},
}) {
  const currentTag = String(releaseTag ?? "");
  assert(RELEASE_TAG_RE.test(currentTag), `invalid release tag: ${currentTag}`);
  if (sourceRevisionDigest !== undefined) normalizeDigest(sourceRevisionDigest, "sourceRevisionDigest");
  if (publicMergeCommit !== undefined) assert(/^[0-9a-f]{40}$/.test(publicMergeCommit), "publicMergeCommit must be a 40-character commit SHA");
  assert(Array.isArray(components) && components.length > 0, "components must be a non-empty array");
  const prior = priorComponentMap(previous);
  const resolved = [];
  for (const spec of components) {
    const fingerprint = buildInputFingerprint(root, spec);
    const candidate = {
      id: spec.id,
      kind: spec.kind,
      name: spec.name ?? spec.id.split(".").at(-1),
      inputFingerprint: {
        version: fingerprint.version,
        algorithm: fingerprint.algorithm,
        digest: fingerprint.digest,
        baseImagesResolved: fingerprint.inputs.baseImagesResolved,
        // Keep the full closure in the durable composition. The host verifies
        // this exact schema and uses the canonical bytes to reject a digest
        // that was not calculated from the recorded build inputs.
        inputs: fingerprint.inputs,
      },
      disposition: "built",
      sourceRelease: spec.sourceRelease ?? { tag: currentTag, ...(publicMergeCommit ? { publicMergeCommit } : {}) },
    };
    const priorEntry = prior.components.get(spec.id);
    const priorInputsResolved = priorEntry?.inputFingerprint?.baseImagesResolved === true;
    if (priorEntry && priorEntry.inputFingerprint.digest === fingerprint.digest && fingerprint.inputs.baseImagesResolved && priorInputsResolved) {
      assertReusableEvidence(priorEntry, currentTag);
      candidate.artifact = priorEntry.artifact;
      candidate.sourceRelease = {
        ...priorEntry.sourceRelease,
        ...(prior.compositionDigest ? { compositionDigest: prior.compositionDigest } : {}),
      };
      candidate.evidence = priorEntry.evidence;
      candidate.disposition = "reused";
    }
    // Validation is intentionally performed before the next component is
    // emitted, so incomplete evidence can never silently become reusable.
    resolved.push(normalizePlanComponent(candidate, resolved.length, { releaseTag: currentTag }));
  }
  const plan = {
    schemaVersion: COMPOSITION_SCHEMA_VERSION,
    kind: COMPOSITION_PLAN_KIND,
    releaseTag: currentTag,
    ...(sourceRevisionDigest ? { sourceRevisionDigest } : {}),
    ...(publicMergeCommit ? { publicMergeCommit } : {}),
    components: resolved.sort((a, b) => a.id.localeCompare(b.id)),
    capabilities: canonicalize(capabilities),
  };
  plan.planDigest = sha256Digest(compositionPlanPayload(plan));
  return validateCompositionPlan(plan);
}

export function finalizeCompositionPlan(plan, builtComponents) {
  const normalizedPlan = validateCompositionPlan(plan);
  assert(isPlainObject(builtComponents), "built component receipts must be an object keyed by component id");
  const builtIds = normalizedPlan.components.filter((component) => component.disposition === "built").map((component) => component.id).sort();
  const receiptIds = Object.keys(builtComponents).sort();
  assert(JSON.stringify(receiptIds) === JSON.stringify(builtIds), "built component receipts must cover exactly the built plan components");
  const components = normalizedPlan.components.map((planned, index) => {
    if (planned.disposition === "reused") {
      return normalizeComponent(planned, index, { releaseTag: normalizedPlan.releaseTag });
    }
    const receipt = builtComponents[planned.id];
    assert(isPlainObject(receipt), `built component receipt ${planned.id} must be an object`);
    assertAllowedKeys(receipt, new Set(["artifact", "evidence"]), `built component receipt ${planned.id}`);
    return normalizeComponent({
      ...planned,
      artifact: receipt.artifact,
      evidence: receipt.evidence,
    }, index, { releaseTag: normalizedPlan.releaseTag });
  }).sort((left, right) => left.id.localeCompare(right.id));
  const composition = {
    schemaVersion: COMPOSITION_SCHEMA_VERSION,
    kind: COMPOSITION_KIND,
    releaseTag: normalizedPlan.releaseTag,
    ...(normalizedPlan.sourceRevisionDigest ? { sourceRevisionDigest: normalizedPlan.sourceRevisionDigest } : {}),
    ...(normalizedPlan.publicMergeCommit ? { publicMergeCommit: normalizedPlan.publicMergeCommit } : {}),
    components,
    capabilities: normalizedPlan.capabilities,
  };
  composition.compositionDigest = sha256Digest(compositionCorePayload(composition));
  return validateComposition(composition);
}

// Backwards-compatible convenience entrypoint for callers that already hold
// every newly built artifact and evidence receipt. Release workflows should
// normally use resolveCompositionPlan followed by finalizeCompositionPlan.
export function resolveComposition(options) {
  const plan = resolveCompositionPlan(options);
  const builtComponents = Object.fromEntries((options.components ?? [])
    .filter((spec) => plan.components.find((component) => component.id === spec.id)?.disposition === "built")
    .map((spec) => [spec.id, { artifact: spec.artifact, evidence: spec.evidence }]));
  return finalizeCompositionPlan(plan, builtComponents);
}

function readJson(filePath, label) {
  try { return JSON.parse(fs.readFileSync(filePath, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

export function parseArgs(argv) {
  const options = { root: DEFAULT_ROOT, spec: "", previous: "", plan: "", builtComponents: "", output: "", mode: "final", json: false };
  const values = new Set(["--root-dir", "--spec", "--previous", "--plan", "--built-components", "--output", "--mode"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--json") { options.json = true; continue; }
    if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/resolve-release-component-composition.mjs --mode <plan|final> --spec <json> --output <json> [--previous <json>] [--plan <json> --built-components <json>] [--root-dir <dir>] [--json]\n");
      process.exit(0);
    }
    if (!values.has(arg)) fail(`unknown argument: ${arg}`);
    const value = argv[++index];
    if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
    if (arg === "--root-dir") options.root = path.resolve(value);
    else if (arg === "--spec") options.spec = path.resolve(value);
    else if (arg === "--previous") options.previous = path.resolve(value);
    else if (arg === "--plan") options.plan = path.resolve(value);
    else if (arg === "--built-components") options.builtComponents = path.resolve(value);
    else if (arg === "--mode") options.mode = value;
    else options.output = path.resolve(value);
  }
  if (options.mode !== "plan" && options.mode !== "final") fail("--mode must be plan or final");
  if (options.plan) {
    if (options.mode !== "final" || options.spec || options.previous || !options.builtComponents) {
      fail("--plan requires --mode final and --built-components without --spec or --previous");
    }
  } else if (!options.spec) {
    fail("--spec is required unless --plan is used");
  }
  if (!options.output) fail("--output is required");
  return options;
}

function writeJsonAtomic(filePath, value) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  const temporary = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o644 });
  fs.renameSync(temporary, filePath);
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = options.plan
    ? finalizeCompositionPlan(
      readJson(options.plan, "runtime composition plan"),
      readJson(options.builtComponents, "built component receipts"),
    )
    : options.mode === "plan"
      ? resolveCompositionPlan({
        root: options.root,
        ...readJson(options.spec, "composition specification"),
        previous: options.previous ? readJson(options.previous, "previous runtime composition") : null,
      })
      : resolveComposition({
        root: options.root,
        ...readJson(options.spec, "composition specification"),
        previous: options.previous ? readJson(options.previous, "previous runtime composition") : null,
      });
  writeJsonAtomic(options.output, result);
  process.stdout.write(`${options.json ? JSON.stringify(result, null, 2) : `runtime composition ${result.kind === COMPOSITION_PLAN_KIND ? "plan" : "resolved"}: ${result.components.length} components (${result.components.filter((component) => component.disposition === "reused").length} reused)`}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  try { main(); }
  catch (error) { process.stderr.write(`runtime component composition resolution failed: ${error.message}\n`); process.exitCode = 1; }
}

export {
  COMPONENT_ID_RE,
  DIGEST_RE,
  OCI_DIGEST_REF_RE,
  canonicalize,
  normalizeArtifact,
  normalizeEvidence,
  normalizeSourceRelease,
};
