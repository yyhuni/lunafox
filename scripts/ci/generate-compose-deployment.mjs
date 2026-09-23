#!/usr/bin/env node
// Release-time packaging only; end users need Docker Compose, not Node/Python.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { validateManifest, parseRuntimeBlocks, parseEngineBlocks } from './verify-public-release.mjs';
import { validateComposition } from './resolve-release-component-composition.mjs';

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const releaseMetadataBaseURL = 'https://raw.githubusercontent.com/yyhuni/lunafox/release-channel';
// The public lifecycle scripts are authored once under deploy/lifecycle and are
// delivered at the deployment root. The ZIP and the public main projection both
// read these files, so the two entry points stay byte-identical.
export const lifecycleScripts = Object.freeze([
 ['install.sh', 'deploy/lifecycle/install.sh'],
 ['start.sh', 'deploy/lifecycle/start.sh'],
 ['restart.sh', 'deploy/lifecycle/restart.sh'],
 ['stop.sh', 'deploy/lifecycle/stop.sh'],
 ['status.sh', 'deploy/lifecycle/status.sh'],
 ['logs.sh', 'deploy/lifecycle/logs.sh'],
 ['uninstall.sh', 'deploy/lifecycle/uninstall.sh'],
 ['lunafox-lifecycle.sh', 'deploy/lifecycle/lunafox-lifecycle.sh'],
]);
const zipProgram = `import json,pathlib,sys,zipfile
root=pathlib.Path(sys.argv[1])
modes=json.loads(sys.argv[3])
with zipfile.ZipFile(sys.argv[2], 'w', compression=zipfile.ZIP_STORED) as archive:
 for source in sorted(root.rglob('*')):
  if source.is_file():
   relative=source.relative_to(root).as_posix()
   entry=zipfile.ZipInfo(relative, (1980,1,1,0,0,0))
   entry.create_system=3
   entry.external_attr=(0o100000 | modes.get(relative, 0o644)) << 16
   archive.writestr(entry,source.read_bytes())
`;

function regularFile(root, relative) {
 const parts = relative.split('/');
 if (path.isAbsolute(relative) || parts.some(p => !p || p === '.' || p === '..')) throw Error(`unsafe package path: ${relative}`);
 let current = root;
 for (const part of parts) {
  current = path.join(current, part);
  if (fs.lstatSync(current).isSymbolicLink()) throw Error(`symlinked package input: ${relative}`);
 }
 if (!fs.statSync(current).isFile()) throw Error(`package input must be a file: ${relative}`);
 return fs.readFileSync(current);
}

function regularExternalFile(filePath, label) {
 if (!filePath || !fs.existsSync(filePath)) throw Error(`${label} is missing: ${filePath || '(missing)'}`);
 const info = fs.lstatSync(filePath);
 if (!info.isFile() || info.isSymbolicLink()) throw Error(`${label} must be a regular file: ${filePath}`);
 const bytes = fs.readFileSync(filePath);
 if (!bytes.length) throw Error(`${label} is empty: ${filePath}`);
 return bytes;
}

function legacyBootstrapPolicy(policy) {
 const bootstrap = policy?.legacyDeploymentBootstrap;
 const keys = bootstrap && typeof bootstrap === 'object' ? Object.keys(bootstrap).sort() : [];
 if (JSON.stringify(keys) !== JSON.stringify(['manifestSha256', 'packageName', 'paths', 'releaseTag', 'sha256'])) {
  throw Error('legacy bootstrap policy is malformed');
 }
 if (!/^v\d+\.\d+\.\d+-alpha\.\d+$/.test(bootstrap.releaseTag) ||
     !/^[a-f0-9]{64}$/.test(bootstrap.sha256) ||
     !/^[a-f0-9]{64}$/.test(bootstrap.manifestSha256) ||
     !Array.isArray(bootstrap.paths) || bootstrap.paths.includes('runtime-composition.json')) {
  throw Error('legacy bootstrap policy is invalid');
 }
 return bootstrap;
}

function validateLegacyBootstrapManifest(manifest, tag, policy, legacyBootstrap) {
 const bootstrap = legacyBootstrapPolicy(policy);
 const optionKeys = legacyBootstrap && typeof legacyBootstrap === 'object' ? Object.keys(legacyBootstrap).sort() : [];
 if (JSON.stringify(optionKeys) !== JSON.stringify(['manifestSha256', 'releaseTag']) ||
     legacyBootstrap.releaseTag !== bootstrap.releaseTag ||
     legacyBootstrap.manifestSha256 !== bootstrap.manifestSha256 ||
     tag !== bootstrap.releaseTag) {
  throw Error('legacy bootstrap request does not match the policy-pinned release');
 }
 const bytes = regularExternalFile(manifest, 'legacy bootstrap release manifest');
 const raw = bytes.toString('utf8');
 const sha256 = crypto.createHash('sha256').update(bytes).digest('hex');
 if (sha256 !== bootstrap.manifestSha256) throw Error('legacy bootstrap manifest digest does not match policy');
 if (/^runtimeComposition:[ \t]*$/m.test(raw)) throw Error('legacy bootstrap manifest must remain a v1 manifest without composition evidence');
 const releaseVersion = raw.match(/^releaseVersion:\s*["']?([^"'\s]+)["']?/m)?.[1] ?? '';
 if (`v${releaseVersion}` !== tag) throw Error('legacy bootstrap manifest version does not match policy');
 const runtime = parseRuntimeBlocks(raw);
 const engines = parseEngineBlocks(raw, policy);
 const identities = new Set();
 for (const ref of [...runtime.flatMap((entry) => entry.refs), ...engines.flat()]) {
  if (ref.namespace !== policy.canonicalNamespace || identities.has(ref.raw)) {
   throw Error('legacy bootstrap manifest has an invalid immutable artifact inventory');
  }
  identities.add(ref.raw);
 }
 return { raw, runtime, engines };
}

function writeFiles(directory, files, modes = new Map()) {
 for (const [name, bytes] of files) {
  const target = path.join(directory, name);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, bytes);
  fs.chmodSync(target, modes.get(name) ?? 0o644);
 }
}

export function generate({ root = defaultRoot, manifest, tag, output, snapshot = '', runtimeComposition = '', releaseProfile = '', legacyBootstrap = null }) {
 if (!/^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$/.test(tag ?? '')) throw Error('invalid release tag');
 const policy = JSON.parse(regularFile(root, 'scripts/ci/public-release-policy.json'));
 let raw;
 let runtime;
 let engines;
 let compositionBytes = null;
 if (legacyBootstrap) {
  // The exact alpha.114 bootstrap predates composition evidence. This private
  // call path is intentionally non-CLI and can only materialize its v1/full
  // deployment group; it never grants a selective-upgrade capability.
  if (runtimeComposition) throw Error('legacy bootstrap must not supply a runtime composition asset');
  ({ raw, runtime, engines } = validateLegacyBootstrapManifest(manifest, tag, policy, legacyBootstrap));
 } else {
  const manifestResult = validateManifest(manifest, policy, tag, releaseProfile);
  raw = fs.readFileSync(manifest, 'utf8');
  const compositionPath = runtimeComposition || path.join(path.dirname(manifest), 'runtime-composition.json');
  compositionBytes = regularExternalFile(compositionPath, 'runtime composition asset');
  let composition;
  try { composition = JSON.parse(compositionBytes.toString('utf8')); }
  catch (error) { throw Error(`runtime composition asset is not valid JSON: ${error.message}`); }
  let normalizedComposition;
  try { normalizedComposition = validateComposition(composition, { requireManifestBinding: true }); }
  catch (error) { throw Error(`runtime composition asset is invalid: ${error.message}`); }
  // The alpha.164 bridge omits this old-client-unknown binding from YAML, but
  // the separately published composition must still bind the exact Manifest.
  if (manifestResult.runtimeComposition && normalizedComposition.compositionDigest !== manifestResult.runtimeComposition.sha256) {
   throw Error(`runtime composition canonical digest does not match manifest: expected ${manifestResult.runtimeComposition.sha256}, got ${normalizedComposition.compositionDigest}`);
  }
  if (normalizedComposition.manifestBinding.manifestDigest !== `sha256:${manifestResult.sha256}`) {
   throw Error('runtime composition manifest binding does not match the release manifest bytes');
  }
  if (normalizedComposition.releaseTag.replace(/^v/, '') !== tag.replace(/^v/, '')) {
   throw Error('runtime composition release tag does not match the requested deployment tag');
  }
  runtime = parseRuntimeBlocks(raw);
  engines = parseEngineBlocks(raw, policy);
 }
 const template = regularFile(root, 'deploy/compose.template.yaml').toString();
 const configuration = regularFile(root, 'deploy/.env.example');
 const common = new Map([
  ['.env.example', configuration], ['.env', configuration],
  ['LICENSE', regularFile(root, 'LICENSE')],
  ['NOTICE-CLOSED-ARTIFACTS.md', regularFile(root, 'NOTICE-CLOSED-ARTIFACTS.md')],
  ['release.manifest.yaml', Buffer.from(raw)],
  ['README.md', regularFile(root, 'docs/public-deployment.md')],
  ['resources/loki/loki-config.yaml', regularFile(root, 'resources/loki/loki-config.yaml')],
  ['resources/alloy/config.alloy', regularFile(root, 'resources/alloy/config.alloy')],
  ['resources/fingerprints/web_fingerprint_v4.json', regularFile(root, 'resources/fingerprints/web_fingerprint_v4.json')],
  ['resources/wordlists/manifest.json', regularFile(root, 'resources/wordlists/manifest.json')],
 ]);
 if (compositionBytes) common.set('runtime-composition.json', compositionBytes);
 const wordlists = JSON.parse(common.get('resources/wordlists/manifest.json'));
 if (!Array.isArray(wordlists.wordlists) || wordlists.wordlists.length === 0) throw Error('missing wordlist inventory');
 for (const entry of wordlists.wordlists) {
  if (typeof entry.fileName !== 'string' || entry.fileName.includes('/') || entry.fileName.includes('\\') || !entry.required) throw Error('invalid required wordlist');
  const name = `resources/wordlists/${entry.fileName}`;
  if (common.has(name)) throw Error('duplicate wordlist');
  common.set(name, regularFile(root, name));
 }
 const work = fs.mkdtempSync(path.join(os.tmpdir(), 'lunafox-compose-package-'));
 const registryExpression = '${RELEASE_REGISTRY:-docker.io}';
 const selectableRef = (refs) => {
  const docker = refs.find(ref => ref.registry === 'docker.io');
  const ghcr = refs.find(ref => ref.registry === 'ghcr.io');
  if (!docker || !ghcr || docker.digest !== ghcr.digest || docker.namespace !== ghcr.namespace || docker.repository !== ghcr.repository) {
   throw Error('release references are not equivalent across Docker Hub and GHCR');
  }
  return `${registryExpression}/${docker.namespace}/${docker.repository}@${docker.digest}`;
 };
 try {
   const selected = new Map(runtime.map(block => [`${block.name.toUpperCase()}_IMAGE_REF`, selectableRef(block.refs)]));
   selected.set('RELEASE_VERSION', tag.slice(1));
   selected.set('AGENT_VERSION', tag.slice(1));
   selected.set('RELEASE_CHANNEL', tag.includes('-') ? 'canary' : 'stable');
   selected.set('RELEASE_METADATA_BASE_URL', releaseMetadataBaseURL);
   selected.set('RELEASE_REGISTRY', registryExpression);
   selected.set('ENGINE_INSTALL_REGISTRY', registryExpression);
   // The published direct-Compose package has no host-side candidate resolver.
   // Keep its reviewed default explicit while the private root lifecycle owns
   // the opt-in CF candidate projection.
   selected.set('ENGINE_INSTALL_CF_ACCELERATION', 'false');
   selected.set('ENGINE_INVENTORY_HOST_PATH', './engine-inventory.yaml');
   selected.set('LUNAFOX_SHARED_DATA_VOLUME_BIND', 'lunafox_data:/opt/lunafox:rw');
   const compose = template.replace(/\$\{([A-Z_]+)(?::[^}]*)?\}/g, (expression, key) => selected.get(key) ?? expression);
   for (const key of selected.keys()) {
    if (key !== 'RELEASE_REGISTRY' && compose.includes('${' + key)) throw Error(`unresolved release input: ${key}`);
   }
   const files = new Map(common);
   const modes = new Map();
   for (const [name, source] of lifecycleScripts) {
    if (files.has(name)) throw Error(`duplicate deployment file: ${name}`);
    files.set(name, regularFile(root, source));
    // Git 100755 and ZIP regular-file 0755 are the delivered contract: users run
    // ./install.sh directly from the extracted package.
    modes.set(name, 0o755);
   }
   files.set('compose.yaml', Buffer.from(compose));
   files.set('engine-inventory.yaml', Buffer.from('enginePackages:\n' + engines.map(refs => {
    selectableRef(refs);
    return `  - refs:\n      - "${refs.find(ref => ref.registry === 'docker.io').raw}"\n      - "${refs.find(ref => ref.registry === 'ghcr.io').raw}"\n`;
   }).join('')));
   const directory = path.join(work, 'unified');
   writeFiles(directory, files, modes);
   const name = `lunafox-${tag}.zip`;
   const archive = path.join(work, name);
   execFileSync('python3', ['-c', zipProgram, directory, archive, JSON.stringify(Object.fromEntries(modes))]);
   const bytes = fs.readFileSync(archive);
   const built = { name, bytes, sha256: crypto.createHash('sha256').update(bytes).digest('hex'), files };
  // Validate the complete closure before exposing publication output.
  fs.mkdirSync(output, { recursive: true });
  const target = path.join(output, built.name);
  if (fs.existsSync(target) && !fs.readFileSync(target).equals(built.bytes)) throw Error(`immutable package conflict: ${built.name}`);
  fs.writeFileSync(target, built.bytes);
  if (snapshot) {
   if (fs.existsSync(snapshot) && fs.readdirSync(snapshot).length > 0) throw Error(`snapshot directory must be empty: ${snapshot}`);
   fs.mkdirSync(snapshot, { recursive: true });
   writeFiles(snapshot, built.files, modes);
  }
  return [{ name: built.name, sha256: built.sha256 }];
 } finally { fs.rmSync(work, { recursive: true, force: true }); }
}

export { validateLegacyBootstrapManifest };

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
 try {
  const args = {};
  for (let i = 2; i < process.argv.length; i += 2) {
   const key = process.argv[i];
   if (!['--root', '--manifest', '--tag', '--output', '--snapshot', '--runtime-composition', '--release-profile'].includes(key) || !process.argv[i + 1] || args[key.slice(2)]) throw Error(`invalid argument: ${key}`);
   const value = process.argv[i + 1];
   if (key === '--runtime-composition') args.runtimeComposition = value;
   else if (key === '--release-profile') args.releaseProfile = value;
   else args[key.slice(2)] = value;
  }
  if (!args.manifest || !args.output) throw Error('--manifest and --output are required');
  process.stdout.write(JSON.stringify(generate(args), null, 2) + '\n');
 } catch (error) { process.stderr.write(`Compose package generation failed: ${error.message}\n`); process.exitCode = 1; }
}
