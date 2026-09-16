#!/usr/bin/env node
// Release-time packaging only; end users need Docker Compose, not Node/Python.
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { validateManifest, parseRuntimeBlocks, parseEngineBlocks } from './verify-public-release.mjs';

const defaultRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const releaseMetadataBaseURL = 'https://raw.githubusercontent.com/yyhuni/lunafox/release-channel';
const zipProgram = `import pathlib,sys,zipfile
root=pathlib.Path(sys.argv[1])
with zipfile.ZipFile(sys.argv[2], 'w', compression=zipfile.ZIP_STORED) as archive:
 for source in sorted(root.rglob('*')):
  if source.is_file():
   entry=zipfile.ZipInfo(source.relative_to(root).as_posix(), (1980,1,1,0,0,0))
   entry.create_system=3
   entry.external_attr=0o100644 << 16
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

function writeFiles(directory, files) {
 for (const [name, bytes] of files) {
  const target = path.join(directory, name);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, bytes);
 }
}

export function generate({ root = defaultRoot, manifest, tag, output, snapshot = '' }) {
 if (!/^v\d+\.\d+\.\d+(?:-(?:alpha|beta|rc)\.\d+)?$/.test(tag ?? '')) throw Error('invalid release tag');
 const policy = JSON.parse(regularFile(root, 'scripts/ci/public-release-policy.json'));
 validateManifest(manifest, policy, tag);
 const raw = fs.readFileSync(manifest, 'utf8');
 const runtime = parseRuntimeBlocks(raw);
 const engines = parseEngineBlocks(raw, policy);
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
   selected.set('ENGINE_INVENTORY_HOST_PATH', './engine-inventory.yaml');
   selected.set('LUNAFOX_SHARED_DATA_VOLUME_BIND', 'lunafox_data:/opt/lunafox:rw');
   const compose = template.replace(/\$\{([A-Z_]+)(?::[^}]*)?\}/g, (expression, key) => selected.get(key) ?? expression);
   for (const key of selected.keys()) {
    if (key !== 'RELEASE_REGISTRY' && compose.includes('${' + key)) throw Error(`unresolved release input: ${key}`);
   }
   const files = new Map(common);
   files.set('compose.yaml', Buffer.from(compose));
   files.set('engine-inventory.yaml', Buffer.from('enginePackages:\n' + engines.map(refs => {
    selectableRef(refs);
    return `  - refs:\n      - "${refs.find(ref => ref.registry === 'docker.io').raw}"\n      - "${refs.find(ref => ref.registry === 'ghcr.io').raw}"\n`;
   }).join('')));
   const directory = path.join(work, 'unified');
   writeFiles(directory, files);
   const name = `lunafox-${tag}.zip`;
   const archive = path.join(work, name);
   execFileSync('python3', ['-c', zipProgram, directory, archive]);
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
   writeFiles(snapshot, built.files);
  }
  return [{ name: built.name, sha256: built.sha256 }];
 } finally { fs.rmSync(work, { recursive: true, force: true }); }
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
 try {
  const args = {};
  for (let i = 2; i < process.argv.length; i += 2) {
   const key = process.argv[i];
   if (!['--root', '--manifest', '--tag', '--output', '--snapshot'].includes(key) || !process.argv[i + 1] || args[key.slice(2)]) throw Error(`invalid argument: ${key}`);
   args[key.slice(2)] = process.argv[i + 1];
  }
  if (!args.manifest || !args.output) throw Error('--manifest and --output are required');
  process.stdout.write(JSON.stringify(generate(args), null, 2) + '\n');
 } catch (error) { process.stderr.write(`Compose package generation failed: ${error.message}\n`); process.exitCode = 1; }
}
