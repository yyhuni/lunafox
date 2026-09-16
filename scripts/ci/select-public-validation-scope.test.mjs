import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { selectScope } from './select-public-validation-scope.mjs';

test('complete PR diff prevents source changes hidden behind a snapshot commit', async t => {
  const cwd = fs.mkdtempSync(path.join(os.tmpdir(), 'public-scope-'));
  t.after(() => fs.rmSync(cwd, { recursive: true, force: true }));
  const git = (...args) => execFileSync('git', args, { cwd, encoding: 'utf8' }).trim();
  git('init', '-q'); git('config', 'user.name', 'Test'); git('config', 'user.email', 'test@example.org');
  const commit = (file, text, subject) => { fs.writeFileSync(path.join(cwd, file), text); git('add', '.'); git('commit', '-qm', subject); return git('rev-parse', 'HEAD'); };
  const base = commit('source.txt', 'original', 'source');
  const subject = 'chore(deploy): finalize deployment snapshot v1.2.3-alpha.1';
  let head = commit('compose.yaml', 'first', subject);
  const event = () => ({ pull_request: { base: { sha: base }, head: { sha: head } } });
  assert.equal(await selectScope({ cwd, event: event(), eventName: 'pull_request' }), 'deployment');
  assert.equal(await selectScope({ cwd }), 'deployment');
  head = commit('source.txt', 'changed', 'source');
  assert.equal(await selectScope({ cwd }), 'full');
  head = commit('compose.yaml', 'second', subject);
  assert.equal(await selectScope({ cwd, event: event(), eventName: 'pull_request' }), 'full');
  assert.equal(await selectScope({ cwd, event: {}, eventName: 'pull_request' }), 'full');
  head = commit('unknown-file', 'unexpected', subject);
  assert.equal(await selectScope({ cwd }), 'full');
  head = commit('.env', 'PUBLIC_HOST=test', 'unrecognized identity');
  assert.equal(await selectScope({ cwd }), 'full');
});

test('source squash reuses only the latest exact-head canonical PR validation', async t => {
  const cwd = fs.mkdtempSync(path.join(os.tmpdir(), 'public-source-scope-'));
  t.after(() => fs.rmSync(cwd, { recursive: true, force: true }));
  const git = (...args) => execFileSync('git', args, { cwd, encoding: 'utf8' }).trim();
  git('init', '-q'); git('config', 'user.name', 'Test'); git('config', 'user.email', 'test@example.org');
  fs.mkdirSync(path.join(cwd, '.github/workflows'), { recursive: true });
  fs.writeFileSync(path.join(cwd, '.github/workflows/public-validate.yml'), 'name: test\n');
  fs.writeFileSync(path.join(cwd, 'source.txt'), 'base'); git('add', '.'); git('commit', '-qm', 'base');
  const base = git('rev-parse', 'HEAD');
  fs.writeFileSync(path.join(cwd, 'source.txt'), 'reviewed'); git('add', '.'); git('commit', '-qm', 'reviewed');
  const reviewed = git('rev-parse', 'HEAD');
  const tree = git('rev-parse', 'HEAD^{tree}');
  const workflowBlob = git('rev-parse', 'HEAD:.github/workflows/public-validate.yml');
  git('reset', '--hard', base); fs.writeFileSync(path.join(cwd, 'source.txt'), 'reviewed'); git('add', '.');
  git('commit', '-qm', 'chore(export): generated deployment projection v1.2.3-alpha.4 (#78)');
  const merged = git('rev-parse', 'HEAD');
  const runsPath = `/actions/workflows/public-validate.yml/runs?event=pull_request&head_sha=${reviewed}&per_page=100`;
  const values = {
    '/pulls/78': { state: 'closed', merged_at: '2026-09-16T00:00:00Z', merge_commit_sha: merged, base: { ref: 'main', sha: base }, head: { sha: reviewed, ref: 'export/v1.2.3-alpha.4', repo: { full_name: 'yyhuni/lunafox' } } },
    [`/commits/${reviewed}`]: { commit: { tree: { sha: tree } } },
    [`/contents/.github/workflows/public-validate.yml?ref=${reviewed}`]: { sha: workflowBlob },
    [runsPath]: { workflow_runs: [
      { id: 10, head_sha: reviewed, head_branch: 'export/v1.2.3-alpha.4', event: 'pull_request', path: '.github/workflows/public-validate.yml', head_repository: { full_name: 'yyhuni/lunafox' }, status: 'completed', conclusion: 'success' },
    ] },
  };
  const api = async request => structuredClone(values[request]);
  assert.equal(await selectScope({ cwd, eventName: 'push', api, repository: 'yyhuni/lunafox' }), 'validated-source');
  values[runsPath].workflow_runs.push({ ...values[runsPath].workflow_runs[0], id: 11, conclusion: 'failure' });
  assert.equal(await selectScope({ cwd, eventName: 'push', api, repository: 'yyhuni/lunafox' }), 'full');
  values['/pulls/78'].head.repo.full_name = 'attacker/fork';
  assert.equal(await selectScope({ cwd, eventName: 'push', api, repository: 'yyhuni/lunafox' }), 'full');
  assert.equal(await selectScope({ cwd, eventName: 'push', api: async () => { throw new Error('unavailable'); }, repository: 'yyhuni/lunafox' }), 'full');
});
