import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { generate } from './generate-compose-deployment.mjs';
import { bindCompositionToManifest, compositionCorePayload, FINGERPRINT_SCHEMA_VERSION, sha256Digest } from './resolve-release-component-composition.mjs';
import { validatePreheatManifest } from './preheat-manifest.mjs';

function fingerprint(componentId) {
 const inputs = {
  schemaVersion: FINGERPRINT_SCHEMA_VERSION, componentId, kind: componentId.split('.')[0], contextPath: '.',
  dockerfile: `${componentId.replaceAll('.', '/')}/Dockerfile`, dockerignore: '',
  files: [], namedContexts: {}, buildArgs: {}, platforms: ['linux/amd64', 'linux/arm64'],
  baseImages: [], baseImagesResolved: true, builderPolicy: {}, generatedInputs: [],
 };
 return { version: FINGERPRINT_SCHEMA_VERSION, algorithm: 'sha256-canonical-json-v1', digest: sha256Digest(inputs), baseImagesResolved: true, inputs };
}

const root = path.resolve(import.meta.dirname, '../..');
const lifecycleScripts = [
 ['install.sh', 'deploy/lifecycle/install.sh'],
 ['start.sh', 'deploy/lifecycle/start.sh'],
 ['restart.sh', 'deploy/lifecycle/restart.sh'],
 ['stop.sh', 'deploy/lifecycle/stop.sh'],
 ['status.sh', 'deploy/lifecycle/status.sh'],
 ['logs.sh', 'deploy/lifecycle/logs.sh'],
 ['uninstall.sh', 'deploy/lifecycle/uninstall.sh'],
 ['lunafox-lifecycle.sh', 'deploy/lifecycle/lunafox-lifecycle.sh'],
];
function fixture(t, version = '1.2.3') {
 const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'compose-package-test-'));
 t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
 const manifest = path.join(dir, 'release.yaml');
 const runtimeComposition = path.join(dir, 'runtime-composition.json');
 const composition = {
  schemaVersion: 1,
  kind: 'lunafox.runtime-composition',
  releaseTag: `v${version}`,
  components: [
   ...['server', 'frontend', 'nginx', 'agent', 'bootstrap'].map((name) => ({
    id: `runtime.${name}`, kind: 'runtime', name,
    inputFingerprint: fingerprint(`runtime.${name}`),
    artifact: { ref: `ghcr.io/yyhuni/lunafox-${name}@sha256:${'a'.repeat(64)}`, digest: `sha256:${'a'.repeat(64)}`, platforms: ['linux/amd64', 'linux/arm64'] },
    disposition: 'built', sourceRelease: { tag: `v${version}` },
    evidence: { image: 'image.json', provenance: 'provenance.json', sbom: 'sbom.json', signature: 'signature.json' },
   })),
   {
    id: 'engine.lunafox.port_scan.runtime', kind: 'engine', name: 'port_scan',
    inputFingerprint: fingerprint('engine.lunafox.port_scan.runtime'),
    artifact: { ref: `ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'b'.repeat(64)}`, digest: `sha256:${'b'.repeat(64)}`, platforms: ['linux/amd64', 'linux/arm64'] },
    disposition: 'built', sourceRelease: { tag: `v${version}` },
    evidence: { image: 'image.json', provenance: 'provenance.json', sbom: 'sbom.json', signature: 'signature.json' },
   },
   {
    id: 'engine.lunafox.port_scan.package', kind: 'engine', name: 'port_scan',
    inputFingerprint: fingerprint('engine.lunafox.port_scan.package'),
    artifact: { ref: `ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'c'.repeat(64)}`, digest: `sha256:${'c'.repeat(64)}`, platforms: ['linux/amd64', 'linux/arm64'] },
    disposition: 'built', sourceRelease: { tag: `v${version}` },
    evidence: { image: 'image.json', provenance: 'provenance.json', sbom: 'sbom.json', signature: 'signature.json' },
   },
  ],
  capabilities: { dynamicFrontendUpstream: true },
 };
 composition.components.sort((left, right) => left.id.localeCompare(right.id));
 composition.compositionDigest = sha256Digest(compositionCorePayload(composition));
 const runtime = ['server','frontend','nginx','agent','bootstrap'].map(name => `  - name: ${name}\n    refs:\n      - docker.io/yyhuni/lunafox-${name}@sha256:${'a'.repeat(64)}\n      - ghcr.io/yyhuni/lunafox-${name}@sha256:${'a'.repeat(64)}\n`).join('');
 const releaseNotes = '## English\n\n- Test release notes.\n\n## 简体中文\n\n- 测试发布说明。\n';
 const releaseNotesDigest = '4406112ce062dd05feacce5f519b8cb7250fd01c7237c43e0f7335da912e8188';
 fs.writeFileSync(manifest, `releaseVersion: "${version}"\nreleaseNotes:\n  digest: "sha256:${releaseNotesDigest}"\n  body: |\n${releaseNotes.trimEnd().split('\n').map(line => `    ${line}`).join('\n')}\nruntimeImages:\n${runtime}enginePackages:\n  - refs:\n      - docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'c'.repeat(64)}\n      - ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'c'.repeat(64)}\nruntimeComposition:\n  schemaVersion: 1\n  asset: "runtime-composition.json"\n  sha256: "${composition.compositionDigest}"\n`);
 composition.manifestBinding = {
  manifestDigest: sha256Digest(fs.readFileSync(manifest)),
 };
 fs.writeFileSync(runtimeComposition, `${JSON.stringify(composition, null, 2)}\n`);
 return { root, manifest, runtimeComposition, tag:`v${version}`, output:path.join(dir,'output'), dir };
}

function renderCompose(dir, compose, configuration, databaseMode, overrides = {}) {
 const composePath = path.join(dir, `compose-${databaseMode}.yaml`);
 const envPath = path.join(dir, `.env-${databaseMode}`);
 const environment = { ...process.env, ...overrides };
 delete environment.DATABASE_MODE;
 delete environment.COMPOSE_PROFILES;
 fs.writeFileSync(composePath, compose);
 fs.writeFileSync(envPath, configuration.replace(/^DATABASE_MODE=.*$/m, `DATABASE_MODE=${databaseMode}`));
 return JSON.parse(execFileSync('docker', ['compose', '--env-file', envPath, '-f', composePath, 'config', '--format', 'json'], {
  encoding: 'utf8',
  env: environment,
 }));
}

test('one package is reproducible, registry-selectable, and matches its snapshot',t=>{
 const options=fixture(t);
 const first=generate(options);
 const snapshot=path.join(options.dir,'snapshot');
 const second=generate({...options,output:path.join(options.dir,'other'),snapshot});
 assert.deepEqual(first,second);
 assert.equal(first.length,1);
 assert.equal(first[0].name,'lunafox-v1.2.3.zip');
const script=`import zipfile,sys,json\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(json.dumps({n:z.read(n).decode() for n in ['compose.yaml','.env','.env.example','README.md','engine-inventory.yaml','release.manifest.yaml','third-party-image-policy.json','runtime-composition.json','preheat-manifest.json','install.sh','start.sh','restart.sh','stop.sh','status.sh','logs.sh','uninstall.sh','lunafox-lifecycle.sh']}))`;
 const artifact=first[0];
 const content=JSON.parse(execFileSync('python3',['-c',script,path.join(options.output,artifact.name)],{encoding:'utf8'}));
 for(const [name,value] of Object.entries(content)) assert.equal(fs.readFileSync(path.join(snapshot,name),'utf8'),value);
 assert.equal(content['third-party-image-policy.json'], fs.readFileSync(path.join(root, 'scripts/ci/third-party-image-policy.json'), 'utf8'));
  assert.match(content['.env'],/^DB_PASSWORD=$/m);
  assert.match(content['.env'],/^JWT_SECRET=$/m);
  assert.match(content['.env'],/^DATABASE_MODE=embedded$/m);
  assert.match(content['.env'],/^COMPOSE_PROFILES=\$\{DATABASE_MODE:-embedded\}$/m);
  // The external database recipe stays visible but must not be an active
  // assignment, so an embedded installation never has to edit it.
  assert.match(content['.env'],/^#DB_HOST=database\.example$/m);
  assert.match(content['.env'],/^#DB_PORT=5432$/m);
  assert.match(content['.env'],/^#DB_USER=postgres$/m);
  assert.match(content['.env'],/^#DB_NAME=lunafox$/m);
  assert.match(content['.env'],/^#DB_SSLMODE=require$/m);
  for (const key of ['DB_HOST','DB_PORT','DB_USER','DB_NAME','DB_SSLMODE']) {
   assert.doesNotMatch(content['.env'], new RegExp(`^${key}=`, 'm'));
  }
  assert.deepEqual(
   content['.env'].split('\n').filter(line => /^[A-Za-z_][A-Za-z0-9_]*=/.test(line)).map(line => line.split('=')[0]).sort(),
   ['COMPOSE_PROFILES','DATABASE_MODE','DB_PASSWORD','JWT_SECRET','LUNAFOX_PREHEAT_MANIFEST_DIGEST','PUBLIC_HOST','PUBLIC_PORT','RELEASE_REGISTRY'],
  );
  const preheatManifest = validatePreheatManifest(JSON.parse(content['preheat-manifest.json']));
  assert.equal(content['.env'].match(/^LUNAFOX_PREHEAT_MANIFEST_DIGEST=(sha256:[a-f0-9]{64})$/m)?.[1], preheatManifest.manifestDigest);
  assert.equal(preheatManifest.release.manifestDigest, `sha256:${sha256Digest(fs.readFileSync(options.manifest)).slice('sha256:'.length)}`);
  assert.equal(preheatManifest.entries.some(entry => entry.repository === 'yyhuni/lunafox-engine-runtime-port-scan'), true);
  assert.match(content['.env'],/^RELEASE_REGISTRY=docker\.io$/m);
  assert.equal(content['.env.example'], content['.env']);
  assert.doesNotMatch(content['.env'],/^PUBLIC_URL=/m);
  assert.match(content['README.md'],/Docker Compose 2\.24\.0/);
  assert.match(content['README.md'],/DATABASE_MODE=external/);
  assert.match(content['compose.yaml'], /https:\/\/127\.0\.0\.1\/healthChecks\/current/);
  assert.doesNotMatch(content['compose.yaml'], /https:\/\/localhost\/healthChecks\/current/);
  assert.match(content['compose.yaml'],/^  config-init:$/m);
  assert.match(content['compose.yaml'],/^  engine-preheater:$/m);
  assert.match(content['compose.yaml'],/^  upgrader:$/m);
  assert.match(content['compose.yaml'],/environment: DB_PASSWORD/);
  assert.match(content['compose.yaml'],/environment: JWT_SECRET/);
  assert.doesNotMatch(content['compose.yaml'],/PUBLIC_URL:/);
  assert.doesNotMatch(content['compose.yaml'],/POSTGRES_PASSWORD:|\n      DB_PASSWORD:|\n      JWT_SECRET:/);
  assert.doesNotMatch(content['compose.yaml'],/\$\{(?:SERVER_IMAGE_REF|AGENT_IMAGE_REF|ENGINE_INSTALL_REGISTRY)/);
  assert.match(content['compose.yaml'],/--image-ref "\$\$AGENT_IMAGE_REF"/);
  assert.doesNotMatch(content['compose.yaml'],/--image-ref "\$(?:docker\.io|ghcr\.io)\//);
  assert.match(content['compose.yaml'],/\$\{RELEASE_REGISTRY:-docker\.io\}\/yyhuni\/lunafox-agent@sha256:/);
  assert.match(content['compose.yaml'],/RELEASE_CHANNEL: stable/);
  assert.match(content['compose.yaml'],/RELEASE_METADATA_BASE_URL: https:\/\/raw\.githubusercontent\.com\/yyhuni\/lunafox\/release-channel/);
  assert.match(content['compose.yaml'],/RELEASE_REGISTRY: \$\{RELEASE_REGISTRY:-docker\.io\}/);
  assert.match(content['compose.yaml'],/ENGINE_INSTALL_CF_ACCELERATION: false/);
  assert.match(content['engine-inventory.yaml'],/docker\.io\/yyhuni\/lunafox-engine-runtime-port-scan@sha256:/);
  assert.match(content['engine-inventory.yaml'],/ghcr\.io\/yyhuni\/lunafox-engine-runtime-port-scan@sha256:/);

  // The packaged lifecycle scripts are the authored bytes with the executable
  // bit the documented `./install.sh` entry point requires.
  const modeScript = `import json,sys,zipfile\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(json.dumps({n:(z.getinfo(n).external_attr>>16)&0o7777 for n in z.namelist()}))`;
  const entryModes = JSON.parse(execFileSync('python3',['-c',modeScript,path.join(options.output,artifact.name)],{encoding:'utf8'}));
  for (const [name, source] of lifecycleScripts) {
   assert.equal(entryModes[name], 0o755, `${name} must be delivered as an executable regular file`);
   assert.equal(content[name], fs.readFileSync(path.join(root, source), 'utf8'), `${name} must match its authored source`);
   assert.equal(fs.readFileSync(path.join(snapshot, name), 'utf8'), content[name], `${name} snapshot must match the ZIP`);
   assert.equal(fs.statSync(path.join(snapshot, name)).mode & 0o777, 0o755);
  }
  assert.equal(entryModes['compose.yaml'], 0o644);
  assert.equal(entryModes['.env'], 0o644);
  assert.equal(entryModes['release.manifest.yaml'], 0o644);
  assert.equal(entryModes['runtime-composition.json'], 0o644);
  assert.equal(entryModes['preheat-manifest.json'], 0o644);

  for(const registry of ['docker.io','ghcr.io']){
   const renderDir = path.join(options.dir, registry);
   fs.mkdirSync(renderDir, { recursive: true });
   const embedded = renderCompose(renderDir, content['compose.yaml'], content['.env'], 'embedded', {RELEASE_REGISTRY:registry});
   assert.ok(embedded.services.postgres);
   assert.equal(embedded.services.postgres.image, 'docker.io/library/postgres@sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675');
   assert.ok(embedded.services.postgres.volumes.some(volume => volume.source === 'postgres_data' && volume.target === '/var/lib/postgresql/data'));
   assert.equal(embedded.services.upgrader.network_mode, 'none');
   assert.equal(embedded.services.upgrader.ports, undefined);
   assert.equal(embedded.services.upgrader.command.at(-1),registry);
   assert.equal(embedded.services.server.environment.RELEASE_REGISTRY,registry);
   assert.equal(embedded.services.server.environment.ENGINE_INSTALL_REGISTRY,registry);
   assert.equal(embedded.services.server.environment.ENGINE_INSTALL_CF_ACCELERATION,'false');
   assert.equal(embedded.services.bootstrap.environment.ENGINE_INSTALL_CF_ACCELERATION,'false');
   assert.ok(embedded.services.upgrader.volumes.some(volume => volume.source === '/var/run/docker.sock' && volume.target === '/var/run/docker.sock' && volume.read_only !== true));
   assert.ok(embedded.services.upgrader.volumes.some(volume => volume.source === 'lunafox_upgrade_state' && volume.target === '/deployment/.lunafox/upgrade'));
   assert.ok(embedded.services.server.volumes.some(volume => volume.source === 'lunafox_upgrade_state' && volume.target === '/opt/lunafox/.lunafox/upgrade'));
   assert.equal(embedded.services.server.depends_on.postgres.condition, 'service_healthy');
   assert.equal(embedded.services.server.depends_on.postgres.required, false);
   assert.equal(embedded.services.migrate.depends_on.postgres.required, false);
   assert.equal(embedded.services['config-init'].environment.COMPOSE_PROFILES, 'embedded');
   const preheater = embedded.services['engine-preheater'];
   assert.equal(preheater.image, embedded.services.agent.image);
   assert.equal(preheater.restart, 'no');
   assert.equal(preheater.network_mode, 'none');
   assert.equal(preheater.environment.LUNAFOX_PREHEAT_TIMEOUT_SECONDS, '900');
   assert.deepEqual(preheater.entrypoint, ['/usr/local/bin/lunafox-engine-preheater']);
   assert.deepEqual(preheater.command, [
    '--manifest', '/deployment/preheat-manifest.json',
    '--release-manifest', '/deployment/release.manifest.yaml',
    '--runtime-composition', '/deployment/runtime-composition.json',
    '--compose', '/deployment/compose.yaml',
    '--third-party-policy', '/deployment/third-party-image-policy.json',
    '--manifest-digest', preheatManifest.manifestDigest,
    '--profile', 'embedded',
    '--cloudflare-acceleration', 'false',
   ]);
   assert.equal(preheater.labels['lunafox.preheat.manifest-digest'], preheatManifest.manifestDigest);
   assert.equal(preheater.labels['lunafox.preheat.profile'], 'embedded');
   const preheaterVolumes = new Map(preheater.volumes.map(volume => [volume.target, volume]));
   assert.deepEqual([...preheaterVolumes.keys()].sort(), [
    '/deployment/compose.yaml',
    '/deployment/preheat-manifest.json',
    '/deployment/release.manifest.yaml',
    '/deployment/runtime-composition.json',
    '/deployment/third-party-image-policy.json',
    '/var/run/docker.sock',
   ]);
   assert.equal(preheaterVolumes.get('/var/run/docker.sock').source, '/var/run/docker.sock');
   for (const [target, source] of [
    ['/deployment/compose.yaml', 'compose.yaml'],
    ['/deployment/preheat-manifest.json', 'preheat-manifest.json'],
    ['/deployment/release.manifest.yaml', 'release.manifest.yaml'],
    ['/deployment/runtime-composition.json', 'runtime-composition.json'],
    ['/deployment/third-party-image-policy.json', 'third-party-image-policy.json'],
   ]) {
    assert.equal(path.basename(preheaterVolumes.get(target).source), source);
   }
   assert.equal([...preheaterVolumes.values()].every(volume => volume.read_only === true), true);
   for (const service of ['config-init', 'redis', 'loki', 'agent-preflight', 'cert-init']) {
    assert.equal(embedded.services[service].depends_on['engine-preheater'].condition, 'service_completed_successfully');
   }
   assert.ok(embedded.services.agent.stop_grace_period, 'the resident Agent must declare a stop grace period');
   for(const service of ['server','frontend','nginx','agent','bootstrap','config-init','migrate','cert-init','upgrader','agent-preflight','engine-preheater']){
    assert.match(embedded.services[service].image,new RegExp(`^${registry.replace('.','\\.')}\\/yyhuni\\/lunafox-`));
   }
   const other=registry==='docker.io'?'ghcr.io':'docker.io';
   assert.doesNotMatch(JSON.stringify(embedded.services),new RegExp(`${other.replace('.','\\.')}\\/yyhuni\\/lunafox-`));

   const external = renderCompose(renderDir, content['compose.yaml'], content['.env'], 'external', {
    RELEASE_REGISTRY: registry, DB_HOST: '2001:db8::1', DB_PORT: '6543', DB_USER: 'remote user', DB_NAME: 'remote database',
    DB_SSLMODE: 'require', DB_PASSWORD: 'remote-password',
   });
   assert.equal(external.services.postgres, undefined);
   for (const service of ['server', 'bootstrap', 'migrate']) {
    assert.equal(external.services[service].environment.DB_HOST, '2001:db8::1');
    assert.equal(external.services[service].environment.DB_PORT, '6543');
    assert.equal(external.services[service].environment.DB_USER, 'remote user');
    assert.equal(external.services[service].environment.DB_NAME, 'remote database');
    assert.equal(external.services[service].environment.DB_SSLMODE, 'require');
   }
   assert.equal(external.services['config-init'].environment.DATABASE_MODE, 'external');
   // The freshly rendered package must carry the derived profile and the Agent
   // grace period, which a lagging public snapshot cannot prove.
   assert.equal(external.services['config-init'].environment.COMPOSE_PROFILES, 'external');
   assert.equal(external.services['engine-preheater'].environment.LUNAFOX_PREHEAT_TIMEOUT_SECONDS, '900');
   const externalProfileIndex = external.services['engine-preheater'].command.indexOf('--profile');
   assert.equal(external.services['engine-preheater'].command[externalProfileIndex + 1], 'external');
   assert.equal(external.services['engine-preheater'].image, external.services.agent.image);
   assert.ok(external.services.agent.stop_grace_period, 'the resident Agent must declare a stop grace period');

   const extracted = path.join(options.dir, `extracted-${registry}`);
   execFileSync('python3', ['-c', 'import sys,zipfile; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])', path.join(options.output, artifact.name), extracted]);
   // This is the exact shell-neutral command a native Windows PowerShell
   // operator runs.  Calling execFileSync without a shell keeps the contract
   // independent of Bash, cmd.exe, and PowerShell quoting rules.
   const directCompose = JSON.parse(execFileSync('docker', [
    'compose', '--project-directory', extracted, '--env-file', path.join(extracted, '.env'),
    '-f', path.join(extracted, 'compose.yaml'), 'config', '--format', 'json',
   ], { encoding: 'utf8', env: { ...process.env, RELEASE_REGISTRY: registry, DATABASE_MODE: 'embedded', COMPOSE_PROFILES: 'embedded' } }));
   assert.equal(directCompose.services['engine-preheater'].image, directCompose.services.agent.image);
   assert.equal(directCompose.services['engine-preheater'].environment.LUNAFOX_PREHEAT_TIMEOUT_SECONDS, '900');
   assert.equal(directCompose.services['engine-preheater'].depends_on, undefined);
   assert.equal(directCompose.services.redis.depends_on['engine-preheater'].condition, 'service_completed_successfully');
   const persistedDigest = 'f'.repeat(64);
   fs.writeFileSync(path.join(extracted, 'compose.override.yaml'), `services:\n  server:\n    image: ${registry}/yyhuni/lunafox-server@sha256:${persistedDigest}\n    environment:\n      RELEASE_VERSION: 1.2.4\n`);
   const restarted = JSON.parse(execFileSync('docker', ['compose', 'config', '--format', 'json'], {
    cwd: extracted,
    encoding: 'utf8',
    env: { ...process.env, RELEASE_REGISTRY:registry, DATABASE_MODE: 'embedded', COMPOSE_PROFILES: 'embedded' },
   }));
   assert.equal(restarted.services.server.image, `${registry}/yyhuni/lunafox-server@sha256:${persistedDigest}`);
   assert.equal(restarted.services.server.environment.RELEASE_VERSION, '1.2.4');
  }
});

test('development Compose and CF lifecycle use the same preheat gate from shell-neutral entry points', t => {
 // Public projections intentionally omit the private development Compose file.
 // The projection self-test passes the authored source root explicitly so this
 // contract remains covered without widening the public export allowlist.
 const sourceRoot = path.resolve(process.env.LUNAFOX_SOURCE_ROOT || root);
 const developmentComposePath = path.join(sourceRoot, 'docker/docker-compose.dev.yml');
 if (!fs.existsSync(developmentComposePath)) {
  t.skip('private development Compose is intentionally absent from the public projection');
  return;
 }
 const developmentCompose = fs.readFileSync(developmentComposePath, 'utf8');
 assert.match(developmentCompose, /^  engine-preheater:\n/m);
 assert.match(developmentCompose, /--development-build-results/);
 assert.match(developmentCompose, /--cloudflare-acceleration[\s\S]*- "false"/);
 assert.match(developmentCompose, /LUNAFOX_PREHEAT_TIMEOUT_SECONDS=\$\{LUNAFOX_PREHEAT_TIMEOUT_SECONDS:-900\}/);
 const serviceBlock = (service) => {
  const lines = developmentCompose.split('\n');
  const block = [];
  let active = false;
  for (const line of lines) {
   if (line === `  ${service}:`) { active = true; block.push(line); continue; }
   if (active && /^  [a-zA-Z0-9_-]+:\s*$/.test(line)) break;
   if (active) block.push(line);
  }
  return block.join('\n');
 };
 for (const service of ['postgres', 'redis', 'loki', 'engine-bootstrap']) {
  assert.match(serviceBlock(service), /engine-preheater:/, `${service} must wait for the development preheater`);
 }

 const lifecycle = fs.readFileSync(path.join(sourceRoot, 'deploy/lifecycle/lunafox-lifecycle.sh'), 'utf8');
 assert.match(lifecycle, /image pulls are owned by engine-preheater/);
 assert.match(lifecycle, /--cloudflare-acceleration\n    - "true"/);
 assert.doesNotMatch(lifecycle, /preheat_capable_deployment[\s\S]{0,120}engine-inventory\.yaml/);

 const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'lunafox-dev-compose-contract-'));
 t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
 const composePath = path.join(directory, 'docker-compose.dev.yml');
 const envPath = path.join(directory, '.env');
 fs.copyFileSync(path.join(sourceRoot, 'docker/docker-compose.dev.yml'), composePath);
 fs.writeFileSync(envPath, [
  'AGENT_IMAGE_REF=yyhuni/lunafox-agent:dev',
  'LOKI_PUSH_URL=http://localhost:3100/loki/api/v1/push',
  'PUBLIC_PORT=8443',
 ].join('\n') + '\n');
 // Native PowerShell invokes this same argv sequence; no Bash wrapper or
 // host-side image pull is part of the development entry contract.
 const composeConfigArgs = [
  'compose', '--project-directory', directory, '--env-file', envPath,
  '-f', composePath, 'config', '--format', 'json',
 ];
 const composeConfigEnv = { ...process.env, DATABASE_MODE: '', COMPOSE_PROFILES: '' };
 const renderComposeConfig = () => JSON.parse(execFileSync('docker', composeConfigArgs, {
  encoding: 'utf8',
  env: composeConfigEnv,
  stdio: ['ignore', 'pipe', 'pipe'],
 }));
 let rendered;
 try {
  rendered = renderComposeConfig();
 } catch (error) {
  const diagnostic = String(error?.stderr ?? '');
  if (!/initial_sync.*not allowed/i.test(diagnostic)) throw error;
  // `initial_sync` is a valid Compose Watch field, but older clients that
  // satisfy the repository's advertised minimum reject it during config.
  // Keep the source assertion above and validate the remaining graph through a
  // temporary compatibility projection instead of weakening the product file.
  fs.writeFileSync(composePath, developmentCompose.replace(/^\s+initial_sync:\s+true\s*$/gm, ''));
  rendered = renderComposeConfig();
 }
 const preheater = rendered.services['engine-preheater'];
 assert.equal(preheater.image, 'yyhuni/lunafox-agent:dev');
 assert.equal(preheater.pull_policy, 'never');
 assert.equal(preheater.network_mode, 'none');
 assert.deepEqual(preheater.entrypoint, ['/usr/local/bin/lunafox-engine-preheater']);
 assert.deepEqual(preheater.command, [
  '--development-build-results', '/bootstrap/image-build-results.json',
  '--cloudflare-acceleration', 'false',
 ]);
 assert.equal(preheater.environment.LUNAFOX_PREHEAT_TIMEOUT_SECONDS, '900');
});

test('prerelease packages pin the canary channel', t => {
 const options = fixture(t, '1.2.3-alpha.1');
 const [artifact] = generate(options);
 const script = `import zipfile,sys\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(z.read('compose.yaml').decode())`;
 const compose = execFileSync('python3', ['-c', script, path.join(options.output, artifact.name)], { encoding: 'utf8' });
 assert.match(compose, /RELEASE_CHANNEL: canary/);
});

test('the frozen alpha.57 package keeps the historical Compose lifecycle', t => {
 const options = fixture(t, '0.0.1-alpha.57');
 const [artifact] = generate(options);
 const script = `import zipfile,sys,json\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(json.dumps({n:z.read(n).decode() for n in ['compose.yaml','.env','.env.example','release.manifest.yaml']}))`;
 const content = JSON.parse(execFileSync('python3', ['-c', script, path.join(options.output, artifact.name)], { encoding: 'utf8' }));
 assert.doesNotMatch(content['compose.yaml'], /^  engine-preheater:$/m);
 assert.doesNotMatch(content['compose.yaml'], /engine-preheater/);
 assert.doesNotMatch(content['.env'], /^LUNAFOX_PREHEAT_/m);
 assert.equal(fs.existsSync(path.join(options.output, 'preheat-manifest.json')), false);
});

test('the registered alpha.164 bridge packages independent composition evidence', t => {
 const options = fixture(t, '0.0.1-alpha.183');
 const raw = fs.readFileSync(options.manifest, 'utf8')
  .replace(/^releaseNotes:\n[\s\S]*?(?=^runtimeImages:)/m, '')
  .replace(/^runtimeComposition:\n[\s\S]*$/m, '');
 fs.writeFileSync(options.manifest, raw);
 const composition = JSON.parse(fs.readFileSync(options.runtimeComposition, 'utf8'));
 fs.writeFileSync(options.runtimeComposition, `${JSON.stringify(bindCompositionToManifest(composition, sha256Digest(fs.readFileSync(options.manifest))), null, 2)}\n`);

 const [artifact] = generate({ ...options, releaseProfile: 'alpha164-bridge' });
 assert.equal(artifact.name, 'lunafox-v0.0.1-alpha.183.zip');
 const script = `import zipfile,sys\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(z.read('compose.yaml').decode())`;
 const compose = execFileSync('python3', ['-c', script, path.join(options.output, artifact.name)], { encoding: 'utf8' });
 assert.doesNotMatch(compose, /^  engine-preheater:$/m);
 assert.doesNotMatch(compose, /engine-preheater/);
 assert.equal(fs.existsSync(path.join(options.output, 'preheat-manifest.json')), false);
 assert.throws(
  () => generate({ ...options, releaseProfile: 'modern', output: path.join(options.dir, 'wrong-profile') }),
  /does not match registered profile/,
 );
});

for(const failure of ['missing-image','bad-digest','development','missing-manifest','unknown-image']){
 test(`reject ${failure} before publication`,t=>{
  const options=fixture(t);
  let raw=fs.readFileSync(options.manifest,'utf8');
  if(failure==='missing-image')raw=raw.replace(/  - name: agent\n[\s\S]*?(?=  - name: bootstrap)/,'');
  if(failure==='bad-digest')raw=raw.replace(/sha256:[a-f0-9]{64}/,'latest');
  if(failure==='development')raw=raw.replace('1.2.3','0.0.0-dev');
  if(failure==='unknown-image')raw=raw.replace('name: agent','name: surprise');
  fs.writeFileSync(options.manifest,raw);
  if(failure==='missing-manifest')fs.unlinkSync(options.manifest);
  assert.throws(()=>generate(options));
  assert.equal(fs.existsSync(options.output),false);
 });
}

test('rejects a composition asset whose digest is not bound by the manifest', t => {
 const options = fixture(t);
 const manifest = fs.readFileSync(options.manifest, 'utf8');
 fs.writeFileSync(options.manifest, manifest.replace(/sha256: "sha256:[a-f0-9]{64}"/, `sha256: "sha256:${'d'.repeat(64)}"`));
 assert.throws(() => generate(options), /runtime composition canonical digest does not match manifest/);
 assert.equal(fs.existsSync(options.output), false);
});
