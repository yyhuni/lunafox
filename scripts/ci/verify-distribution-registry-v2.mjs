#!/usr/bin/env node

import http from "node:http";
import https from "node:https";
import { pathToFileURL } from "node:url";

const acceptedStatuses = new Set([200, 401]);
const distributionVersion = "registry/2.0";

function usage() {
  return `Usage: node scripts/ci/verify-distribution-registry-v2.mjs --endpoint URL [--timeout-ms N]\n`;
}

function parseArgs(argv) {
  const args = { endpoint: "", timeoutMs: 5000 };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === "--endpoint") args.endpoint = argv[++index] ?? "";
    else if (arg === "--timeout-ms") args.timeoutMs = Number(argv[++index] ?? "");
    else if (arg === "--help" || arg === "-h") {
      process.stdout.write(usage());
      process.exit(0);
    } else throw new Error(`unknown argument: ${arg}`);
  }
  if (!args.endpoint) throw new Error("--endpoint is required");
  if (!Number.isSafeInteger(args.timeoutMs) || args.timeoutMs <= 0) {
    throw new Error("--timeout-ms must be a positive integer");
  }
  return args;
}

function parseEndpoint(rawEndpoint) {
  let endpoint;
  try {
    endpoint = new URL(rawEndpoint);
  } catch {
    throw new Error(`invalid Registry endpoint: ${rawEndpoint}`);
  }
  if (endpoint.protocol !== "http:" && endpoint.protocol !== "https:") {
    throw new Error(`Registry endpoint must explicitly use http or https: ${rawEndpoint}`);
  }
  if (endpoint.username || endpoint.password) {
    throw new Error("Registry endpoint must not contain credentials");
  }
  if (endpoint.pathname !== "/v2/" || endpoint.search || endpoint.hash) {
    throw new Error(`Registry endpoint must use the exact /v2/ path: ${rawEndpoint}`);
  }
  return endpoint;
}

function validateRegistryResponse(statusCode, rawVersionHeader) {
  if (!acceptedStatuses.has(statusCode)) {
    throw new Error(`Registry /v2/ returned HTTP ${statusCode}; expected 200 or 401`);
  }
  const headerValues = (Array.isArray(rawVersionHeader) ? rawVersionHeader : [rawVersionHeader])
    .filter((value) => typeof value === "string")
    .flatMap((value) => value.split(","))
    .map((value) => value.trim().toLowerCase());
  if (!headerValues.includes(distributionVersion)) {
    throw new Error("Registry /v2/ response is missing Docker-Distribution-Api-Version: registry/2.0");
  }
}

async function verifyDistributionRegistry(rawEndpoint, { timeoutMs = 5000 } = {}) {
  if (!Number.isSafeInteger(timeoutMs) || timeoutMs <= 0) {
    throw new Error("Registry verification timeout must be a positive integer");
  }
  const endpoint = parseEndpoint(rawEndpoint);
  const transport = endpoint.protocol === "http:" ? http : https;

  await new Promise((resolve, reject) => {
    let settled = false;
    const finish = (error) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      if (error) reject(error);
      else resolve();
    };
    const request = transport.request(endpoint, {
      method: "GET",
      headers: { accept: "application/json" },
    });
    const timer = setTimeout(() => {
      request.destroy(new Error(`Registry /v2/ request timed out after ${timeoutMs}ms`));
    }, timeoutMs);

    request.once("response", (response) => {
      try {
        validateRegistryResponse(
          response.statusCode ?? 0,
          response.headers["docker-distribution-api-version"],
        );
        finish();
      } catch (error) {
        finish(error);
      } finally {
        response.destroy();
      }
    });
    request.once("error", (error) => {
      finish(new Error(`cannot reach Registry /v2/ endpoint ${endpoint.origin}: ${error.message}`));
    });
    request.end();
  });
}

async function main() {
  try {
    const args = parseArgs(process.argv.slice(2));
    await verifyDistributionRegistry(args.endpoint, { timeoutMs: args.timeoutMs });
    process.stdout.write(`verified Distribution Registry v2 endpoint: ${args.endpoint}\n`);
  } catch (error) {
    process.stderr.write(`Distribution Registry v2 preflight failed: ${error.message}\n`);
    process.exitCode = 1;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) await main();

export { validateRegistryResponse, verifyDistributionRegistry };
