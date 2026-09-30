import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it, vi } from "vitest";

import { handleRequest, THIRD_PARTY_POLICY, type FetchLike, type RequestDiagnostic } from "../src/registry";

const repository = "yyhuni/lunafox-engine-runtime-port-scan";
const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const blobDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
const env = {};
const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");

function registryRequest(path: string, init?: RequestInit): Request {
  return new Request(`https://registry.example${path}`, init);
}

function tokenResponse(): Response {
  return new Response(JSON.stringify({ token: "anonymous-pull-token" }));
}

function diagnostics(): { entries: RequestDiagnostic[]; logger: (entry: RequestDiagnostic) => void } {
  const entries: RequestDiagnostic[] = [];
  return { entries, logger: (entry) => entries.push(entry) };
}

describe("LunaFox GHCR registry Worker", () => {
  it("keeps the third-party Worker allowlist aligned with the release-frozen policy", () => {
    const releasePolicy = JSON.parse(
      fs.readFileSync(path.join(repositoryRoot, "scripts/ci/third-party-image-policy.json"), "utf8"),
    ) as { entries: Array<{ repository: string; digest: string }> };

    expect(THIRD_PARTY_POLICY.map(({ repository, digest }) => ({ repository, digest }))).toEqual(
      releasePolicy.entries.map(({ repository, digest }) => ({ repository, digest })),
    );
  });

  it("routes every release-frozen third-party manifest to Docker Hub anonymously", async () => {
    for (const entry of THIRD_PARTY_POLICY) {
      const fetcher = vi
        .fn<FetchLike>()
        .mockResolvedValueOnce(tokenResponse())
        .mockResolvedValueOnce(new Response("manifest", { status: 200 }));
      const response = await handleRequest(
        registryRequest(`/v2/${entry.repository}/manifests/${entry.digest}`),
        env,
        fetcher,
      );

      expect(response.status).toBe(200);
      const tokenURL = new URL(String(fetcher.mock.calls[0]?.[0]));
      expect(tokenURL.origin + tokenURL.pathname).toBe("https://auth.docker.io/token");
      expect(tokenURL.searchParams.get("service")).toBe("registry.docker.io");
      expect(tokenURL.searchParams.get("scope")).toBe(`repository:${entry.repository}:pull`);
      const upstreamURL = new URL(String(fetcher.mock.calls[1]?.[0]));
      expect(upstreamURL.origin + upstreamURL.pathname).toBe(
        `https://registry-1.docker.io/v2/${entry.repository}/manifests/${entry.digest}`,
      );
      expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Authorization")).toBe(
        "Bearer anonymous-pull-token",
      );
    }
  });

  it("permits Docker Hub blob transport only for an allowlisted repository", async () => {
    const entry = THIRD_PARTY_POLICY[0];
    const blob = "sha256:" + "c".repeat(64);
    const fetcher = vi.fn<FetchLike>().mockResolvedValueOnce(tokenResponse()).mockResolvedValueOnce(new Response("blob"));

    const response = await handleRequest(
      registryRequest(`/v2/${entry.repository}/blobs/${blob}`, { headers: { Range: "bytes=0-3" } }),
      env,
      fetcher,
    );

    expect(response.status).toBe(200);
    const upstreamURL = new URL(String(fetcher.mock.calls[1]?.[0]));
    expect(upstreamURL.origin).toBe("https://registry-1.docker.io");
    expect(upstreamURL.pathname).toBe(`/v2/${entry.repository}/blobs/${blob}`);
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Range")).toBe("bytes=0-3");
  });

  it("strips Docker Hub credentials after a third-party redirect", async () => {
    const entry = THIRD_PARTY_POLICY[0];
    const blob = "sha256:" + "c".repeat(64);
    const fetcher = vi
      .fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockResolvedValueOnce(new Response(null, { status: 307, headers: { Location: "https://cdn.example/postgres-layer" } }))
      .mockResolvedValueOnce(new Response("layer", { status: 200 }));

    const response = await handleRequest(
      registryRequest(`/v2/${entry.repository}/blobs/${blob}`, { headers: { Range: "bytes=0-3" } }),
      env,
      fetcher,
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toBe("layer");
    expect(new URL(String(fetcher.mock.calls[2]?.[0])).origin).toBe("https://cdn.example");
    expect(new Headers(fetcher.mock.calls[2]?.[1]?.headers).get("Authorization")).toBeNull();
    expect(new Headers(fetcher.mock.calls[2]?.[1]?.headers).get("Range")).toBe("bytes=0-3");
  });

  it("rejects third-party manifest digest drift while answering third-party Referrers probes as empty", async () => {
    const fetcher = vi.fn();
    const entry = THIRD_PARTY_POLICY[0];
    const wrongDigest = "sha256:" + "d".repeat(64);
    const paths = [
      `/v2/${entry.repository}/manifests/${wrongDigest}`,
      `/v2/${entry.repository}/manifests/latest`,
      `/v2/library/unknown/manifests/${entry.digest}`,
      `/v2/docker.io/${entry.repository}/manifests/${entry.digest}`,
      `/v2/${entry.repository}/manifests/${entry.digest}?ns=docker.io`,
      `/v2/${entry.repository}/blobs/uploads/${entry.digest}`,
    ];

    for (const path of paths) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(403);
    }
    expect(fetcher).not.toHaveBeenCalled();

    // The Worker serves no Referrers for third-party repositories, so probes
    // must read as an empty referrers set: Docker aborts proxied pulls on 403.
    const referrerProbes = [
      `/v2/${entry.repository}/referrers/${entry.digest}`,
      `/v2/${entry.repository}/referrers/${entry.digest}?artifactType=${encodeURIComponent("application/vnd.dev.sigstore.bundle.v0.3+json")}`,
      `/v2/${entry.repository}/manifests/sha256-${entry.digest.slice(7)}`,
    ];
    for (const path of referrerProbes) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(404);
    }
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("maps third-party token and registry failures to a credential-safe upstream failure", async () => {
    const entry = THIRD_PARTY_POLICY[0];
    const tokenFailure = vi.fn<FetchLike>().mockResolvedValueOnce(new Response("rate limited", { status: 429 }));
    const tokenResponseFailure = await handleRequest(
      registryRequest(`/v2/${entry.repository}/manifests/${entry.digest}`),
      env,
      tokenFailure,
    );
    expect(tokenResponseFailure.status).toBe(502);
    expect(tokenFailure).toHaveBeenCalledTimes(1);

    const registryFailure = vi.fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockRejectedValueOnce(new Error("Docker Hub connection failed"));
    const registryResponseFailure = await handleRequest(
      registryRequest(`/v2/${entry.repository}/manifests/${entry.digest}`),
      env,
      registryFailure,
    );
    expect(registryResponseFailure.status).toBe(502);
    expect(registryFailure).toHaveBeenCalledTimes(2);
  });

  it("answers Registry V2 GET and HEAD probes locally", async () => {
    const fetcher = vi.fn(async () => {
      throw new Error("upstream fetch must not run");
    });

    for (const method of ["GET", "HEAD"]) {
      const response = await handleRequest(registryRequest("/v2/", { method }), env, fetcher);
      expect(response.status).toBe(200);
      expect(response.headers.get("Docker-Distribution-API-Version")).toBe("registry/2.0");
    }
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("rejects unsupported methods before contacting GHCR", async () => {
    const fetcher = vi.fn();
    const response = await handleRequest(
      registryRequest(`/v2/${repository}/blobs/uploads/`, { method: "POST" }),
      env,
      fetcher,
    );

    expect(response.status).toBe(405);
    expect(response.headers.get("Allow")).toBe("GET, HEAD");
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("rejects tags, catalog, malformed digests, and non-first-party repositories locally", async () => {
    const fetcher = vi.fn();
    const rejectedPaths = [
      `/v2/${repository}/manifests/latest`,
      "/v2/_catalog",
      `/v2/${repository}/blobs/sha512:${"a".repeat(128)}`,
      `/v2/astral-sh/uv/manifests/${digest}`,
    ];

    for (const path of rejectedPaths) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(403);
    }
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("returns a local 404 for unfiltered OCI referrers probes so Docker can finish a pull", async () => {
    const fetcher = vi.fn();
    for (const path of [`/v2/${repository}/referrers/${digest}`, `/v2/${repository}/manifests/sha256-${digest.slice(7)}`]) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(404);
    }
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("forwards only the filtered Sigstore Referrers request and preserves OCI headers", async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('{"manifests":[]}'));
        controller.close();
      },
    });
    const fetcher = vi
      .fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockResolvedValueOnce(
        new Response(stream, {
          headers: {
            "Content-Type": "application/vnd.oci.image.index.v1+json",
            "OCI-Filters-Applied": "artifactType",
            "OCI-Subject": digest,
            Link: '</v2/' + repository + '/referrers/' + digest + '?artifactType=next>; rel="next"',
          },
        }),
      );
    const artifactType = encodeURIComponent("application/vnd.dev.sigstore.bundle.v0.3+json");

    const response = await handleRequest(
      registryRequest(`/v2/${repository}/referrers/${digest}?artifactType=${artifactType}`, {
        headers: { Accept: "application/vnd.oci.image.index.v1+json" },
      }),
      env,
      fetcher,
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toBe('{"manifests":[]}');
    expect(response.headers.get("OCI-Filters-Applied")).toBe("artifactType");
    expect(response.headers.get("OCI-Subject")).toBe(digest);
    expect(response.headers.get("Link")).toContain('rel="next"');

    const upstreamURL = new URL(String(fetcher.mock.calls[1]?.[0]));
    expect(upstreamURL.origin + upstreamURL.pathname).toBe(`https://ghcr.io/v2/${repository}/referrers/${digest}`);
    expect(upstreamURL.searchParams.get("artifactType")).toBe("application/vnd.dev.sigstore.bundle.v0.3+json");
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Authorization")).toBe("Bearer anonymous-pull-token");
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Accept")).toBe(
      "application/vnd.oci.image.index.v1+json",
    );
  });

  it("forwards filtered Sigstore Referrers HEAD requests without a downstream body", async () => {
    const fetcher = vi
      .fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockResolvedValueOnce(
        new Response(null, {
          headers: {
            "Content-Type": "application/vnd.oci.image.index.v1+json",
            "OCI-Filters-Applied": "artifactType",
            "OCI-Subject": digest,
          },
        }),
      );
    const artifactType = encodeURIComponent("application/vnd.dev.sigstore.bundle.v0.3+json");

    const response = await handleRequest(
      registryRequest(`/v2/${repository}/referrers/${digest}?artifactType=${artifactType}`, { method: "HEAD" }),
      env,
      fetcher,
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toBe("");
    expect(response.headers.get("OCI-Filters-Applied")).toBe("artifactType");
    expect(response.headers.get("OCI-Subject")).toBe(digest);
    expect(fetcher.mock.calls[1]?.[1]?.method).toBe("HEAD");
  });

  it("answers unauthorized Referrers queries as empty and never forwards them", async () => {
    const fetcher = vi.fn();
    const referrerProbes = [
      `/v2/${repository}/referrers/${digest}?artifactType=application%2Fvnd.oci.image.manifest.v1%2Bjson`,
      `/v2/${repository}/referrers/${digest}?artifactType=application%2Fvnd.dev.sigstore.bundle.v0.3%2Bjson&n=1`,
      `/v2/${repository}/referrers/${digest}?artifactType=application%2Fvnd.dev.sigstore.bundle.v0.3%2Bjson&artifactType=application%2Fvnd.dev.sigstore.bundle.v0.3%2Bjson`,
      `/v2/${repository}/referrers/${digest}?artifactType=application%2Fvnd.dev.sigstore.bundle.v0.2%2Bjson`,
    ];
    for (const path of referrerProbes) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(404);
    }
    expect(fetcher).not.toHaveBeenCalled();

    // Content requests and malformed digests keep the strict local rejection.
    const rejectedPaths = [
      `/v2/${repository}/manifests/${digest}?n=1`,
      `/v2/${repository}/referrers/sha256:${"a".repeat(63)}?artifactType=application%2Fvnd.dev.sigstore.bundle.v0.3%2Bjson`,
    ];
    for (const path of rejectedPaths) {
      const response = await handleRequest(registryRequest(path), env, fetcher);
      expect(response.status).toBe(403);
    }
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("obtains an anonymous repository-scoped token and forwards caller media types", async () => {
    const fetcher = vi
      .fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockResolvedValueOnce(
        new Response("manifest", {
          headers: {
            "Content-Type": "application/vnd.oci.image.index.v1+json",
            "Docker-Content-Digest": digest,
            "WWW-Authenticate": 'Bearer realm="https://ghcr.io/token"',
          },
        }),
      );

    const response = await handleRequest(
      registryRequest(`/v2/${repository}/manifests/${digest}`, {
        headers: { Accept: "application/vnd.oci.image.index.v1+json" },
      }),
      env,
      fetcher,
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toBe("manifest");
    expect(response.headers.get("WWW-Authenticate")).toBeNull();

    const tokenURL = new URL(String(fetcher.mock.calls[0]?.[0]));
    expect(tokenURL.origin + tokenURL.pathname).toBe("https://ghcr.io/token");
    expect(tokenURL.searchParams.get("scope")).toBe(`repository:${repository}:pull`);
    expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get("Authorization")).toBeNull();
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Authorization")).toBe("Bearer anonymous-pull-token");
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Accept")).toBe(
      "application/vnd.oci.image.index.v1+json",
    );
  });

  it("follows redirects internally, strips credentials after GHCR, and streams the final body", async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("streamed-"));
        controller.enqueue(new TextEncoder().encode("blob"));
        controller.close();
      },
    });
    const fetcher = vi
      .fn<FetchLike>()
      .mockResolvedValueOnce(tokenResponse())
      .mockResolvedValueOnce(
        new Response(null, {
          status: 307,
          headers: { Location: "https://cdn.example/blob?signature=private" },
        }),
      )
      .mockResolvedValueOnce(
        new Response(stream, {
          status: 206,
          headers: {
            "Accept-Ranges": "bytes",
            "Content-Range": "bytes 10-20/100",
            "Docker-Content-Digest": blobDigest,
          },
        }),
      );

    const response = await handleRequest(
      registryRequest(`/v2/${repository}/blobs/${blobDigest}`, { headers: { Range: "bytes=10-20" } }),
      env,
      fetcher,
    );

    expect(response.status).toBe(206);
    expect(await response.text()).toBe("streamed-blob");
    expect(response.headers.get("Content-Range")).toBe("bytes 10-20/100");
    expect(response.headers.get("Location")).toBeNull();
    expect(new Headers(fetcher.mock.calls[1]?.[1]?.headers).get("Range")).toBe("bytes=10-20");
    expect(new Headers(fetcher.mock.calls[2]?.[1]?.headers).get("Authorization")).toBeNull();
  });

  it("bounds redirect chains and keeps redirect locations out of the response", async () => {
    const fetcher = vi.fn<FetchLike>().mockResolvedValueOnce(tokenResponse());
    for (let index = 0; index < 4; index += 1) {
      fetcher.mockResolvedValueOnce(
        new Response(null, { status: 302, headers: { Location: `https://cdn.example/${index}` } }),
      );
    }

    const response = await handleRequest(registryRequest(`/v2/${repository}/blobs/${blobDigest}`), env, fetcher);

    expect(response.status).toBe(502);
    expect(response.headers.get("Location")).toBeNull();
    expect(fetcher).toHaveBeenCalledTimes(5);
  });

  it("logs only credential-safe structured diagnostics", async () => {
    const { entries, logger } = diagnostics();
    const fetcher = vi.fn(async () => {
      const credentialError = ["Bearer", "credential-from-client-must-never-appear"].join(" ");
      throw new Error(credentialError);
    });
    const clock = vi.fn().mockReturnValueOnce(10).mockReturnValueOnce(17);

    const response = await handleRequest(
      registryRequest(`/v2/${repository}/manifests/${digest}`, {
        headers: { Authorization: "Bearer client-secret", Cookie: "session=secret" },
      }),
      env,
      fetcher,
      logger,
      clock,
    );

    expect(response.status).toBe(502);
    expect(entries).toEqual([
      {
        event: "oci_registry_request",
        method: "GET",
        repository,
        downstreamStatus: 502,
        upstreamStatus: null,
        elapsedMs: 7,
      },
    ]);
    expect(JSON.stringify(entries)).not.toContain("client-secret");
    expect(JSON.stringify(entries)).not.toContain("credential-from-client");
  });
});
