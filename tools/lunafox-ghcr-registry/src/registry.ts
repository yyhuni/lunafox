export interface ProxyEnv {}

export type FetchLike = (input: Request | string | URL, init?: RequestInit) => Promise<Response>;

export interface RequestDiagnostic {
  event: "oci_registry_request";
  method: string;
  repository: string | null;
  downstreamStatus: number;
  upstreamStatus: number | null;
  elapsedMs: number;
}

export type DiagnosticLogger = (diagnostic: RequestDiagnostic) => void;

type RegistryResource = "manifests" | "blobs" | "referrers";

interface AuthorizedRequest {
  repository: string;
  resource: RegistryResource;
  digest: string;
  artifactType?: string;
  upstream: UpstreamRegistry;
}

interface UpstreamRegistry {
  origin: string;
  tokenOrigin: string;
  service: string;
}

interface ThirdPartyPolicyEntry extends UpstreamRegistry {
  repository: string;
  digest: string;
}

class UpstreamError extends Error {
  constructor(readonly upstreamStatus: number | null) {
    super("upstream registry request failed");
  }
}

const GHCR_ORIGIN = "https://ghcr.io";
const GHCR_REGISTRY: UpstreamRegistry = {
  origin: GHCR_ORIGIN,
  tokenOrigin: `${GHCR_ORIGIN}/token`,
  service: "ghcr.io",
};
const DOCKER_HUB_REGISTRY: UpstreamRegistry = {
  origin: "https://registry-1.docker.io",
  tokenOrigin: "https://auth.docker.io/token",
  service: "registry.docker.io",
};
const MAX_REDIRECTS = 3;
const sha256DigestPattern = /^sha256:[a-f0-9]{64}$/;
const repositoryPattern = /^yyhuni\/lunafox-[a-z0-9][a-z0-9._-]*$/;
const registryPathPattern = /^\/v2\/(.+)\/(manifests|blobs)\/(.+)$/;
const referrersPathPattern = /^\/v2\/(.+)\/referrers\/(sha256:[a-f0-9]{64})$/;
const legacyReferrersTagPattern = /^\/v2\/(.+)\/manifests\/sha256-[a-f0-9]{64}$/;
const sigstoreBundleArtifactType = "application/vnd.dev.sigstore.bundle.v0.3+json";

// These entries are release-frozen. The Worker deliberately does not accept a
// repository and digest supplied by the caller as an upstream selector.
export const THIRD_PARTY_POLICY: readonly ThirdPartyPolicyEntry[] = Object.freeze([
  Object.freeze({
    repository: "library/postgres",
    digest: "sha256:67f41722b7a8cbdb868a44a4995c846eddfdc2973bccb291ce937dce88ad5675",
    ...DOCKER_HUB_REGISTRY,
  }),
  Object.freeze({
    repository: "library/redis",
    digest: "sha256:2afba59292f25f5d1af200496db41bea2c6c816b059f57ae74703a50a03a27d0",
    ...DOCKER_HUB_REGISTRY,
  }),
  Object.freeze({
    repository: "grafana/loki",
    digest: "sha256:3c8fd3570dd9219951a60d3f919c7f31923d10baee578b77bc26c4a0b32d092d",
    ...DOCKER_HUB_REGISTRY,
  }),
  Object.freeze({
    repository: "grafana/alloy",
    digest: "sha256:b8ec653c44235fbe910879145dac3597d66b0aaecf60bcbbe82580767771a839",
    ...DOCKER_HUB_REGISTRY,
  }),
]);

const passthroughResponseHeaders = [
  "Accept-Ranges",
  "Cache-Control",
  "Content-Disposition",
  "Content-Encoding",
  "Content-Length",
  "Content-Range",
  "Content-Type",
  "Docker-Content-Digest",
  "Docker-Distribution-API-Version",
  "Etag",
  "Last-Modified",
  "Link",
  "OCI-Filters-Applied",
  "OCI-Subject",
  "Retry-After",
  "Vary",
];

function defaultLogger(diagnostic: RequestDiagnostic): void {
  console.log(diagnostic);
}

function parseAuthorizedRequest(url: URL): AuthorizedRequest | null {
  const match = registryPathPattern.exec(url.pathname);
  if (match) {
    if (url.search !== "") {
      return null;
    }

    const [, repository, resource, digest] = match;
    if (!sha256DigestPattern.test(digest)) return null;

    if (repositoryPattern.test(repository)) {
      return {
        repository,
        resource: resource as RegistryResource,
        digest,
        upstream: GHCR_REGISTRY,
      };
    }

    const policy = THIRD_PARTY_POLICY.find((entry) => entry.repository === repository);
    if (!policy) return null;
    // The image manifest is the release identity and must be exact. Blob
    // digests are discovered from that manifest by the OCI client and are
    // independently checked by Docker/ORAS against the streamed bytes.
    if (resource === "manifests" && policy.digest !== digest) return null;

    return {
      repository,
      resource: resource as RegistryResource,
      digest,
      upstream: policy,
    };
  }

  const referrersMatch = referrersPathPattern.exec(url.pathname);
  if (!referrersMatch || !repositoryPattern.test(referrersMatch[1])) {
    return null;
  }

  const artifactTypes = url.searchParams.getAll("artifactType");
  if (url.searchParams.size !== 1 || artifactTypes.length !== 1 || artifactTypes[0] !== sigstoreBundleArtifactType) {
    return null;
  }

  return {
    repository: referrersMatch[1],
    resource: "referrers",
    digest: referrersMatch[2],
    artifactType: sigstoreBundleArtifactType,
    upstream: GHCR_REGISTRY,
  };
}

function parsedRepository(pathname: string): string | null {
  return (
    registryPathPattern.exec(pathname)?.[1] ??
    referrersPathPattern.exec(pathname)?.[1] ??
    legacyReferrersTagPattern.exec(pathname)?.[1] ??
    null
  );
}

function isAllowedCompatibilityProbe(url: URL): boolean {
  if (url.search !== "") {
    return false;
  }

  const match = referrersPathPattern.exec(url.pathname);
  if (match !== null) return repositoryPattern.test(match[1]);

  const legacyMatch = legacyReferrersTagPattern.exec(url.pathname);
  return legacyMatch !== null && repositoryPattern.test(legacyMatch[1]);
}

function upstreamRequestHeaders(request: Request, token: string | null): Headers {
  const headers = new Headers();
  const accept = request.headers.get("Accept");
  const range = request.headers.get("Range");

  if (accept) {
    headers.set("Accept", accept);
  }
  if (range) {
    headers.set("Range", range);
  }
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  return headers;
}

async function getAnonymousPullToken(
  target: AuthorizedRequest,
  fetcher: FetchLike,
): Promise<string> {
  const tokenURL = new URL(target.upstream.tokenOrigin);
  tokenURL.searchParams.set("service", target.upstream.service);
  tokenURL.searchParams.set("scope", `repository:${target.repository}:pull`);

  let response: Response;
  try {
    response = await fetcher(tokenURL, { redirect: "manual" });
  } catch {
    throw new UpstreamError(null);
  }

  if (!response.ok) {
    throw new UpstreamError(response.status);
  }

  let payload: { token?: unknown; access_token?: unknown };
  try {
    payload = (await response.json()) as { token?: unknown; access_token?: unknown };
  } catch {
    throw new UpstreamError(response.status);
  }

  const token = payload.token ?? payload.access_token;
  if (typeof token !== "string" || token.length === 0) {
    throw new UpstreamError(response.status);
  }

  return token;
}

function isRedirect(response: Response): boolean {
  return response.status >= 300 && response.status < 400;
}

async function fetchUpstreamResource(
  request: Request,
  target: AuthorizedRequest,
  token: string,
  fetcher: FetchLike,
): Promise<Response> {
  const initialURL = new URL(`${target.upstream.origin}/v2/${target.repository}/${target.resource}/${target.digest}`);
  if (target.resource === "referrers") {
    initialURL.searchParams.set("artifactType", target.artifactType ?? sigstoreBundleArtifactType);
  }
  let nextURL = initialURL.toString();
  let includeAuthorization = true;

  for (let redirects = 0; redirects <= MAX_REDIRECTS; redirects += 1) {
    let response: Response;
    try {
      response = await fetcher(nextURL, {
        method: request.method,
        headers: upstreamRequestHeaders(request, includeAuthorization ? token : null),
        redirect: "manual",
      });
    } catch {
      throw new UpstreamError(null);
    }

    if (!isRedirect(response)) {
      return response;
    }

    const location = response.headers.get("Location");
    if (!location || redirects === MAX_REDIRECTS) {
      throw new UpstreamError(response.status);
    }

    try {
      nextURL = new URL(location, nextURL).toString();
    } catch {
      throw new UpstreamError(response.status);
    }
    includeAuthorization = false;
  }

  throw new UpstreamError(null);
}

function downstreamResponse(upstream: Response, request: Request): Response {
  const headers = new Headers();
  for (const name of passthroughResponseHeaders) {
    const value = upstream.headers.get(name);
    if (value) {
      headers.set(name, value);
    }
  }

  return new Response(request.method === "HEAD" ? null : upstream.body, {
    status: upstream.status,
    headers,
  });
}

function localRegistryProbe(): Response {
  return new Response(null, {
    status: 200,
    headers: { "Docker-Distribution-API-Version": "registry/2.0" },
  });
}

function methodNotAllowed(): Response {
  return new Response("method not allowed", {
    status: 405,
    headers: { Allow: "GET, HEAD" },
  });
}

function policyRejected(): Response {
  return new Response("forbidden", { status: 403 });
}

function notFound(): Response {
  return new Response("not found", { status: 404 });
}

function upstreamFailure(): Response {
  return new Response("upstream registry unavailable", { status: 502 });
}

export async function handleRequest(
  request: Request,
  _env: ProxyEnv,
  fetcher: FetchLike = fetch,
  logger: DiagnosticLogger = defaultLogger,
  now: () => number = Date.now,
): Promise<Response> {
  const startedAt = now();
  const url = new URL(request.url);
  let repository: string | null = null;
  let upstreamStatus: number | null = null;
  let response: Response;

  try {
    if (request.method !== "GET" && request.method !== "HEAD") {
      response = methodNotAllowed();
    } else if (url.pathname === "/v2/" && url.search === "") {
      response = localRegistryProbe();
    } else if (!url.pathname.startsWith("/v2/")) {
      response = notFound();
    } else {
      repository = parsedRepository(url.pathname);
      if (isAllowedCompatibilityProbe(url)) {
        // Docker probes both the OCI Referrers API and its legacy digest-tag
        // fallback after pulling images with attestations. GHCR returns 404 for
        // these probes when no referrers index exists; keep the Worker read-only
        // and avoid exposing a broader resource API.
        response = notFound();
      } else {
        const target = parseAuthorizedRequest(url);
        if (!target) {
          response = policyRejected();
        } else {
          repository = target.repository;
          const token = await getAnonymousPullToken(target, fetcher);
          const upstream = await fetchUpstreamResource(request, target, token, fetcher);
          upstreamStatus = upstream.status;
          response = downstreamResponse(upstream, request);
        }
      }
    }
  } catch (error) {
    if (error instanceof UpstreamError) {
      upstreamStatus = error.upstreamStatus;
    }
    response = upstreamFailure();
  }

  logger({
    event: "oci_registry_request",
    method: request.method,
    repository,
    downstreamStatus: response.status,
    upstreamStatus,
    elapsedMs: Math.max(0, now() - startedAt),
  });
  return response;
}
