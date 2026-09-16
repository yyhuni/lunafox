#!/usr/bin/env node
import fs from 'node:fs';
import { execFileSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';

const SNAPSHOT = new Set(['.env', '.env.example', 'compose.yaml', 'engine-inventory.yaml', 'release.manifest.yaml']);
const DEPLOYMENT = /^chore\(deploy\): finalize deployment snapshot v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?(?: \(#\d+\))?$/;

export function selectScope({ cwd = process.cwd(), event = {}, eventName = '', head = 'HEAD' } = {}) {
  const git = (...args) => execFileSync('git', args, { cwd, encoding: 'utf8' }).trim();
  const sha = git('rev-parse', '--verify', `${head}^{commit}`);
  const subject = git('show', '-s', '--format=%s', sha);
  if (!DEPLOYMENT.test(subject)) return 'full';
  // Compare the whole PR, not just its last commit: an earlier source change
  // must never hide behind a final generated-snapshot commit.
  let base;
  if (eventName === 'pull_request') {
    if (event.pull_request?.head?.sha !== sha || !/^[a-f0-9]{40}$/.test(event.pull_request?.base?.sha ?? '')) return 'full';
    base = event.pull_request.base.sha;
    git('merge-base', '--is-ancestor', base, sha);
  } else {
    base = git('rev-parse', '--verify', `${sha}^`);
  }
  const files = git('diff', '--name-only', '-z', base, sha).split('\0').filter(Boolean);
  return files.length > 0 && files.every(file => SNAPSHOT.has(file)) ? 'deployment' : 'full';
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const event = process.env.GITHUB_EVENT_PATH ? JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, 'utf8')) : {};
  const scope = selectScope({ event, eventName: process.env.GITHUB_EVENT_NAME });
  if (process.env.GITHUB_OUTPUT) fs.appendFileSync(process.env.GITHUB_OUTPUT, `scope=${scope}\n`);
  process.stdout.write(`${scope}\n`);
}
