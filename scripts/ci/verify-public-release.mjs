#!/usr/bin/env node

/** Release identity, Compose closure, anonymous registry visibility, and provenance preflight. */

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const SCRIPT_DIR = path.dirname(fileURLToPath(import.meta.url));
const DEFAULT_ROOT = path.resolve(SCRIPT_DIR, "../..");
const DEFAULT_POLICY = path.join(SCRIPT_DIR, "public-release-policy.json");
const FIRST_TAG = "v0.0.1-alpha.57";
const RUNTIME_NAMES = ["server", "frontend", "nginx", "agent", "bootstrap"];
const DIGEST_RE = /^sha256:[a-f0-9]{64}$/;

function fail(message) { throw new Error(message); }

function parseArgs(argv) {
  const args = { root: DEFAULT_ROOT, policy: DEFAULT_POLICY, manifest: "", provenance: "", tag: FIRST_TAG, probe: false, json: false };
  const values = new Set(["--root-dir", "--policy", "--manifest", "--provenance", "--tag"]);
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--probe") args.probe = true;
    else if (arg === "--json") args.json = true;
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write("Usage: node scripts/ci/verify-public-release.mjs --manifest <file> [--provenance <file>] [--tag <tag>] [--probe] [--json]\\n");
      process.exit(0);
    } else if (values.has(arg)) {
      const value = argv[++index];
      if (!value || value.startsWith("--")) fail(`${arg} requires a value`);
      if (arg === "--root-dir") args.root = path.resolve(value);
      else if (arg === "--policy") args.policy = path.resolve(value);
      else if (arg === "--manifest") args.manifest = path.resolve(value);
      else if (arg === "--provenance") args.provenance = path.resolve(value);
      else args.tag = value;
    } else fail(`unknown argument: ${arg}`);
  }
  if (!args.manifest) fail("--manifest is required");
  return args;
}

function readJson(file, label) {
  try { return JSON.parse(fs.readFileSync(file, "utf8")); }
  catch (error) { fail(`cannot read ${label}: ${error.message}`); }
}

function sha256File(file) { return crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex"); }

function parseDigestRefs(text) {
  return [...text.matchAll(/(?:docker\.io|ghcr\.io)\/[^\s"']+@sha256:[a-f0-9]{64}/g)].map((match) => match[0]);
}

function parseRef(raw, field) {
  const match = raw.match(/^(docker\.io|ghcr\.io)\/([^/]+)\/([^@/]+)@(sha256:[a-f0-9]{64})$/);
  if (!match) fail(`${field} contains an invalid digest-qualified ref: ${raw}`);
  return { raw, registry: match[1], namespace: match[2], repository: match[3], digest: match[4] };
}

function parseRefsBlock(block, field) {
  const inline = block.match(/\brefs:\s*\[([^\]]*)\]/m);
  const refs = inline
    ? inline[1].split(",").map((ref) => ref.trim().replace(/^["']|["']$/g, "")).filter(Boolean)
    : [...block.matchAll(/^      - ["']?([^"'\s]+)["']?\s*$/gm)].map((match) => match[1]);
  if (refs.length !== 2) fail(`${field} must contain exactly Docker Hub and GHCR candidates`);
  const parsed = refs.map((ref, index) => parseRef(ref, `${field}[${index}]`));
  if (parsed[0].registry !== "docker.io" || parsed[1].registry !== "ghcr.io") fail(`${field} must order docker.io then ghcr.io candidates`);
  if (parsed[0].digest !== parsed[1].digest) fail(`${field} candidates must use the same manifest digest`);
  if (parsed[0].namespace !== parsed[1].namespace || parsed[0].repository !== parsed[1].repository) fail(`${field} candidates must preserve namespace and repository identity`);
  return parsed;
}

function parseRuntimeBlocks(text) {
  const section = text.match(/^runtimeImages:\s*$([\s\S]*?)^enginePackages:\s*$/m)?.[1];
  if (!section) fail("release manifest is missing runtimeImages/enginePackages sections");
  const blocks = [];
  const expression = /^  - name: ([A-Za-z0-9._-]+)[ \t]*\n([\s\S]*?)(?=^  - name:|(?![\s\S]))/gm;
  for (const match of section.matchAll(expression)) blocks.push({ name: match[1], refs: parseRefsBlock(match[2], `runtimeImages[${match[1]}].refs`) });
  if (blocks.length !== RUNTIME_NAMES.length) fail(`runtimeImages must contain exactly ${RUNTIME_NAMES.length} entries`);
  const seen = new Set();
  for (const block of blocks) {
    if (!RUNTIME_NAMES.includes(block.name)) fail(`runtimeImages contains unsupported image ${block.name}`);
    if (seen.has(block.name)) fail(`runtimeImages contains duplicate image ${block.name}`);
    seen.add(block.name);
    const expected = `lunafox-${block.name}`;
    if (block.refs.some((ref) => ref.repository !== expected)) fail(`runtimeImages[${block.name}] must use repository ${expected}`);
  }
  for (const name of RUNTIME_NAMES) if (!seen.has(name)) fail(`runtimeImages is missing required image ${name}`);
  return blocks;
}

function parseEngineBlocks(text, policy) {
  const section = text.match(/^enginePackages:\s*$([\s\S]*)$/m)?.[1];
  if (section === undefined) fail("release manifest is missing enginePackages");
  const blocks = [...section.matchAll(/^  - refs:[ \t]*([^\n]*)\n?([\s\S]*?)(?=^  - refs:|(?![\s\S]))/gm)].map((match, index) => parseRefsBlock(`refs:${match[1]}\n${match[2]}`, `enginePackages[${index}].refs`));
  if (blocks.length === 0) fail("enginePackages cannot be empty");
  const repositoryPattern = new RegExp(policy.engineRepositoryPattern);
  for (const [index, refs] of blocks.entries()) {
    if (refs.some((ref) => !repositoryPattern.test(ref.repository))) fail(`enginePackages[${index}] has an invalid Engine Runtime Image repository`);
  }
  return blocks;
}

function validateManifest(file, policy, tag) {
  if (!fs.existsSync(file)) fail(`release manifest is missing: ${file}`);
  const text = fs.readFileSync(file, "utf8");
  if (/0\.0\.0-dev|alpha\.46|IMAGE_REGISTRY=|IMAGE_NAMESPACE=|WORKER_IMAGE|lunafox-installer|checksums\.txt/.test(text)) fail("manifest contains retired or development identity");
  const releaseVersion = text.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1] ?? "";
  if (!/^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.+-]+)?$/.test(releaseVersion)) fail("release manifest releaseVersion is invalid");
  if (`v${releaseVersion}` !== tag) fail(`release manifest releaseVersion does not match ${tag}`);
  if (tag === FIRST_TAG && releaseVersion !== "0.0.1-alpha.57") fail("alpha.57 manifest must contain releaseVersion 0.0.1-alpha.57");
  const runtimeImages = parseRuntimeBlocks(text);
  const enginePackages = parseEngineBlocks(text, policy);
  const refs = [...runtimeImages.flatMap((entry) => entry.refs), ...enginePackages.flat()].map((entry) => entry.raw);
  const identities = new Set();
  for (const ref of [...runtimeImages.flatMap((entry) => entry.refs), ...enginePackages.flat()]) {
    if (ref.namespace !== policy.canonicalNamespace) fail(`canonical namespace drift: ${ref.raw}`);
    if (!DIGEST_RE.test(ref.digest)) fail(`invalid immutable digest: ${ref.raw}`);
    if (identities.has(ref.raw)) fail(`release manifest contains duplicate artifact identity: ${ref.raw}`);
    identities.add(ref.raw);
  }
  return { checked: true, refs, runtimeImages: runtimeImages.map((entry) => entry.name), enginePackageCount: enginePackages.length, sha256: sha256File(file) };
}

function validateProvenance(file, policy, manifestResult) {
  if (!file) return { checked: false };
  const provenance = readJson(file, "provenance");
  for (const field of policy.provenance.requiredFields ?? []) if (provenance[field] === undefined || provenance[field] === null || provenance[field] === "") fail(`provenance is missing ${field}`);
  if (policy.provenance.longLivedCosignKeysAllowed !== false) fail("provenance policy must prohibit long-lived Cosign keys");
  if (provenance.privateSourceRepository && provenance.privateSourceRepository !== policy.privateBuilderRepository) fail("provenance private source repository drift");
  if (provenance.exportManifest && !String(provenance.exportManifest).includes(manifestResult.sha256)) fail("provenance is not bound to the exact manifest digest");
  return { checked: true, schemaVersion: provenance.schemaVersion ?? null };
}

function registryEndpoint(ref) {
  const [location, digest] = ref.split("@");
  const [registry, namespace, repository] = location.split("/");
  return { registry, endpoint: registry === "docker.io" ? "registry-1.docker.io" : registry, repository: `${namespace}/${repository}`, digest };
}

function parseBearerChallenge(header) {
  const match = String(header ?? "").match(/^Bearer\s+(.+)$/i);
  if (!match) return null;
  return Object.fromEntries([...match[1].matchAll(/([A-Za-z]+)="([^"]*)"/g)].map((entry) => [entry[1], entry[2]]));
}

async function registryFetch(url, headers = {}) {
  return fetch(url, { method: "GET", headers: { Accept: "application/vnd.oci.image.index.v1+json, application/vnd.oci.image.manifest.v1+json", ...headers } });
}

async function probeRegistry(ref) {
  const target = registryEndpoint(ref);
  const url = `https://${target.endpoint}/v2/${target.repository}/manifests/${target.digest}`;
  let response = await registryFetch(url);
  if (response.status === 401) {
    const challenge = parseBearerChallenge(response.headers.get("www-authenticate"));
    if (!challenge?.realm) fail(`anonymous registry probe returned an unsupported authentication challenge: ${ref}`);
    const tokenURL = new URL(challenge.realm);
    if (challenge.service) tokenURL.searchParams.set("service", challenge.service);
    tokenURL.searchParams.set("scope", challenge.scope || `repository:${target.repository}:pull`);
    const tokenResponse = await fetch(tokenURL);
    if (!tokenResponse.ok) fail(`anonymous registry token request failed (${tokenResponse.status}): ${ref}`);
    const tokenPayload = await tokenResponse.json();
    const token = tokenPayload.token || tokenPayload.access_token;
    if (!token) fail(`anonymous registry token response is missing a token: ${ref}`);
    response = await registryFetch(url, { Authorization: `Bearer ${token}` });
  }
  if (!response.ok) fail(`anonymous registry probe failed (${response.status}): ${ref}`);
  return { ref, status: response.status };
}

async function verify(options) {
  const policy = readJson(options.policy, "release policy");
  if (policy.canonicalRepository !== "yyhuni/lunafox" || policy.privateBuilderRepository !== "yyhuni/lunafox-private") fail("release policy repository identities are invalid");
  if (policy.canonicalNamespace !== "yyhuni") fail("canonical namespace must be yyhuni");
  if (policy.firstRelease?.tag !== FIRST_TAG || policy.firstRelease?.channel !== "canary" || policy.firstRelease?.stableAllowed !== false) fail("first release policy must pin alpha.57 canary");
  if (policy.signer?.issuer !== "https://token.actions.githubusercontent.com" || !/^\^https:\/\/github\\\.com\/yyhuni\/lunafox\/\\\.github\/workflows\/public-validate\\\.yml@refs\/heads\/main\$$/.test(String(policy.signer?.identityPattern))) fail("public Runtime keyless signer policy is invalid");
  if (policy.binarySigner?.issuer !== "https://token.actions.githubusercontent.com" || !String(policy.binarySigner?.identityPattern).includes("yyhuni/lunafox-private") || !String(policy.binarySigner?.identityPattern).includes("refs/tags")) fail("private Agent binary signer policy is invalid");
  const manifest = validateManifest(options.manifest, policy, options.tag);
  const provenance = validateProvenance(options.provenance, policy, manifest);
  const probes = [];
  if (options.probe) for (const ref of manifest.refs) probes.push(await probeRegistry(ref));
  return { schemaVersion: 2, passed: true, tag: options.tag, canonicalRepository: policy.canonicalRepository, manifest, provenance, anonymousRegistryProbes: probes };
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const result = await verify(args);
  process.stdout.write(`${args.json ? JSON.stringify(result, null, 2) : "public Compose release preflight passed"}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main().catch((error) => { process.stderr.write(`public release preflight failed: ${error.message}\n`); process.exitCode = 1; });
}

export { parseDigestRefs, parseEngineBlocks, parseRuntimeBlocks, validateManifest, validateProvenance, verify };
