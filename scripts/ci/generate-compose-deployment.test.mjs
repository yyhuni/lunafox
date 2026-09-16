import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { generate } from './generate-compose-deployment.mjs';

const root = path.resolve(import.meta.dirname, '../..');
function fixture(t, version = '1.2.3') {
 const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'compose-package-test-'));
 t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
 const manifest = path.join(dir, 'release.yaml');
 const runtime = ['server','frontend','nginx','agent','bootstrap'].map(name => `  - name: ${name}\n    refs:\n      - docker.io/yyhuni/lunafox-${name}@sha256:${'a'.repeat(64)}\n      - ghcr.io/yyhuni/lunafox-${name}@sha256:${'a'.repeat(64)}\n`).join('');
 fs.writeFileSync(manifest, `releaseVersion: "${version}"\nruntimeImages:\n${runtime}enginePackages:\n  - refs:\n      - docker.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'b'.repeat(64)}\n      - ghcr.io/yyhuni/lunafox-engine-runtime-port-scan@sha256:${'b'.repeat(64)}\n`);
 return { root, manifest, tag:`v${version}`, output:path.join(dir,'output'), dir };
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
 const script=`import zipfile,sys,json\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(json.dumps({n:z.read(n).decode() for n in ['compose.yaml','.env','.env.example','README.md','engine-inventory.yaml']}))`;
 const artifact=first[0];
 const content=JSON.parse(execFileSync('python3',['-c',script,path.join(options.output,artifact.name)],{encoding:'utf8'}));
 for(const [name,value] of Object.entries(content)) assert.equal(fs.readFileSync(path.join(snapshot,name),'utf8'),value);
  assert.match(content['.env'],/^DB_PASSWORD=$/m);
  assert.match(content['.env'],/^JWT_SECRET=$/m);
  assert.match(content['.env'],/^DB_USER=postgres$/m);
  assert.match(content['.env'],/^DB_NAME=lunafox$/m);
  assert.match(content['.env'],/^DATABASE_MODE=embedded$/m);
  assert.match(content['.env'],/^COMPOSE_PROFILES=\$\{DATABASE_MODE:-embedded\}$/m);
  assert.match(content['.env'],/^DB_HOST=$/m);
  assert.match(content['.env'],/^DB_PORT=5432$/m);
  assert.match(content['.env'],/^DB_SSLMODE=$/m);
  assert.match(content['.env'],/^RELEASE_REGISTRY=docker\.io$/m);
  assert.equal(content['.env.example'], content['.env']);
  assert.doesNotMatch(content['.env'],/^PUBLIC_URL=/m);
  assert.match(content['README.md'],/Docker Compose 2\.24\.0/);
  assert.match(content['README.md'],/DATABASE_MODE=external/);
  assert.match(content['compose.yaml'],/^  config-init:$/m);
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
  assert.match(content['engine-inventory.yaml'],/docker\.io\/yyhuni\/lunafox-engine-runtime-port-scan@sha256:/);
  assert.match(content['engine-inventory.yaml'],/ghcr\.io\/yyhuni\/lunafox-engine-runtime-port-scan@sha256:/);

  for(const registry of ['docker.io','ghcr.io']){
   const renderDir = path.join(options.dir, registry);
   fs.mkdirSync(renderDir, { recursive: true });
   const embedded = renderCompose(renderDir, content['compose.yaml'], content['.env'], 'embedded', {RELEASE_REGISTRY:registry});
   assert.ok(embedded.services.postgres);
   assert.equal(embedded.services.upgrader.network_mode, 'none');
   assert.equal(embedded.services.upgrader.ports, undefined);
   assert.equal(embedded.services.upgrader.command.at(-1),registry);
   assert.equal(embedded.services.server.environment.RELEASE_REGISTRY,registry);
   assert.equal(embedded.services.server.environment.ENGINE_INSTALL_REGISTRY,registry);
   assert.ok(embedded.services.upgrader.volumes.some(volume => volume.source === '/var/run/docker.sock' && volume.target === '/var/run/docker.sock' && volume.read_only !== true));
   assert.ok(embedded.services.upgrader.volumes.some(volume => volume.source === 'lunafox_upgrade_state' && volume.target === '/deployment/.lunafox/upgrade'));
   assert.ok(embedded.services.server.volumes.some(volume => volume.source === 'lunafox_upgrade_state' && volume.target === '/opt/lunafox/.lunafox/upgrade'));
   assert.equal(embedded.services.server.depends_on.postgres.condition, 'service_healthy');
   assert.equal(embedded.services.server.depends_on.postgres.required, false);
   assert.equal(embedded.services.migrate.depends_on.postgres.required, false);
   for(const service of ['server','frontend','nginx','agent','bootstrap','config-init','migrate','cert-init','upgrader','agent-preflight']){
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

   const extracted = path.join(options.dir, `extracted-${registry}`);
   execFileSync('python3', ['-c', 'import sys,zipfile; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2])', path.join(options.output, artifact.name), extracted]);
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

test('prerelease packages pin the canary channel', t => {
 const options = fixture(t, '1.2.3-alpha.1');
 const [artifact] = generate(options);
 const script = `import zipfile,sys\nwith zipfile.ZipFile(sys.argv[1]) as z:\n print(z.read('compose.yaml').decode())`;
 const compose = execFileSync('python3', ['-c', script, path.join(options.output, artifact.name)], { encoding: 'utf8' });
 assert.match(compose, /RELEASE_CHANNEL: canary/);
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
