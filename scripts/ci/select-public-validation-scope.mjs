#!/usr/bin/env node
import fs from "node:fs";
import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

const SNAPSHOT = new Set([
  ".env", ".env.example", "compose.yaml", "engine-inventory.yaml", "release.manifest.yaml",
  // The delivered lifecycle entry points are part of the deployment snapshot: a
  // change to any of them must run the full public validation.
  "install.sh", "start.sh", "restart.sh", "stop.sh", "status.sh", "logs.sh", "uninstall.sh", "lunafox-lifecycle.sh",
]);
const DEPLOYMENT = /^chore\(deploy\): finalize deployment snapshot v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)?(?: \(#\d+\))?$/;
const EXPORTED_SOURCE = /^chore\(export\): generated deployment projection v\d+\.\d+\.\d+(?:-[A-Za-z0-9.-]+)? \(#(\d+)\)$/;
const SHA = /^[a-f0-9]{40}$/;
const WORKFLOW_PATH = ".github/workflows/public-validate.yml";

async function githubApi(path, { apiUrl, repository, token, fetchImpl = fetch }) {
  if (!apiUrl || !repository || !token) throw new Error("GitHub API identity is incomplete");
  const response = await fetchImpl(`${apiUrl}/repos/${repository}${path}`, {
    signal: AbortSignal.timeout(15000),
    headers: { Accept: "application/vnd.github+json", Authorization: `Bearer ${token}`, "X-GitHub-Api-Version": "2022-11-28" },
  });
  if (!response.ok) throw new Error(`GitHub API ${response.status}: ${path}`);
  return response.json();
}

async function selectValidatedSource({ git, sha, subject, eventName, api, repository }) {
  const match = subject.match(EXPORTED_SOURCE);
  if (eventName !== "push" || !match || !api || !repository) return false;
  try {
    const parent = git("rev-parse", "--verify", `${sha}^`);
    if (!SHA.test(parent) || git("show", "-s", "--format=%P", sha).split(" ").length !== 1) return false;
    const pr = await api(`/pulls/${match[1]}`);
    if (pr.state !== "closed" || !pr.merged_at || pr.merge_commit_sha !== sha ||
        pr.base?.ref !== "main" || pr.base?.sha !== parent || pr.head?.repo?.full_name !== repository ||
        !SHA.test(pr.head?.sha ?? "")) return false;
    const reviewedCommit = await api(`/commits/${pr.head.sha}`);
    const reviewedWorkflow = await api(`/contents/${WORKFLOW_PATH}?ref=${pr.head.sha}`);
    if (reviewedCommit.commit?.tree?.sha !== git("rev-parse", `${sha}^{tree}`) ||
        reviewedWorkflow.sha !== git("rev-parse", `${sha}:${WORKFLOW_PATH}`)) return false;
    const runs = await api(`/actions/workflows/public-validate.yml/runs?event=pull_request&head_sha=${pr.head.sha}&per_page=100`);
    const canonical = (runs.workflow_runs ?? []).filter(run =>
      run.head_sha === pr.head.sha && run.head_branch === pr.head.ref && run.event === "pull_request" &&
      run.path === WORKFLOW_PATH && run.head_repository?.full_name === repository);
    const latest = canonical.sort((a, b) => b.id - a.id)[0];
    return latest?.status === "completed" && latest.conclusion === "success";
  } catch {
    return false;
  }
}

export async function selectScope({ cwd = process.cwd(), event = {}, eventName = "", head = "HEAD", api, repository = process.env.GITHUB_REPOSITORY } = {}) {
  const git = (...args) => execFileSync("git", args, { cwd, encoding: "utf8" }).trim();
  const sha = git("rev-parse", "--verify", `${head}^{commit}`);
  const subject = git("show", "-s", "--format=%s", sha);
  if (await selectValidatedSource({ git, sha, subject, eventName, api, repository })) return "validated-source";
  if (!DEPLOYMENT.test(subject)) return "full";
  // Compare the whole PR, not just its last commit: an earlier source change
  // must never hide behind a final generated-snapshot commit.
  let base;
  if (eventName === "pull_request") {
    if (event.pull_request?.head?.sha !== sha || !SHA.test(event.pull_request?.base?.sha ?? "")) return "full";
    base = event.pull_request.base.sha;
    git("merge-base", "--is-ancestor", base, sha);
  } else {
    base = git("rev-parse", "--verify", `${sha}^`);
  }
  const files = git("diff", "--name-only", "-z", base, sha).split("\0").filter(Boolean);
  return files.length > 0 && files.every(file => SNAPSHOT.has(file)) ? "deployment" : "full";
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const event = process.env.GITHUB_EVENT_PATH ? JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, "utf8")) : {};
  const api = process.env.GITHUB_TOKEN ? path => githubApi(path, {
    apiUrl: process.env.GITHUB_API_URL,
    repository: process.env.GITHUB_REPOSITORY,
    token: process.env.GITHUB_TOKEN,
  }) : undefined;
  const scope = await selectScope({ event, eventName: process.env.GITHUB_EVENT_NAME, api });
  if (process.env.GITHUB_OUTPUT) fs.appendFileSync(process.env.GITHUB_OUTPUT, `scope=${scope}\n`);
  process.stdout.write(`${scope}\n`);
}
