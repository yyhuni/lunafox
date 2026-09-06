export const WEBSITE_ASSET_SCOPE = {
  url: "https://api.acme.com",
  host: "api.acme.com",
  readOnly: true,
} as const

export const WEBSITE_URL_SCOPE_CASES = [
  {
    name: "root includes descendants and ignores query fragments",
    scope: "https://api.acme.com/?source=scan#section",
    candidate: "https://api.acme.com/v1/products?query=1#fragment",
    matches: true,
  },
  {
    name: "default port matches explicit default port",
    scope: "https://api.acme.com",
    candidate: "https://api.acme.com:443/v1/products",
    matches: true,
  },
  {
    name: "explicit non-default port matches only itself",
    scope: "https://api.acme.com:8443",
    candidate: "https://api.acme.com:8443/v1/products",
    matches: true,
  },
  {
    name: "nested scope includes itself",
    scope: "https://api.acme.com/a",
    candidate: "https://api.acme.com/a",
    matches: true,
  },
  {
    name: "nested scope includes a trailing slash",
    scope: "https://api.acme.com/a",
    candidate: "https://api.acme.com/a/",
    matches: true,
  },
  {
    name: "nested scope includes descendants",
    scope: "https://api.acme.com/a/",
    candidate: "https://api.acme.com/a/child",
    matches: true,
  },
  {
    name: "nested scope rejects a sibling prefix",
    scope: "https://api.acme.com/a",
    candidate: "https://api.acme.com/ab",
    matches: false,
  },
  {
    name: "scope rejects a lookalike host",
    scope: "https://api.acme.com",
    candidate: "https://api.acme.com.evil/a",
    matches: false,
  },
  {
    name: "scope rejects a scheme mismatch",
    scope: "https://api.acme.com",
    candidate: "http://api.acme.com/a",
    matches: false,
  },
  {
    name: "scope rejects a port mismatch",
    scope: "https://api.acme.com",
    candidate: "https://api.acme.com:8443/a",
    matches: false,
  },
  {
    name: "escaped path remains a path boundary",
    scope: "https://api.acme.com/a%2Fb",
    candidate: "https://api.acme.com/a%2Fb/c",
    matches: true,
  },
  {
    name: "escaped path does not decode into another hierarchy",
    scope: "https://api.acme.com/a%2Fb",
    candidate: "https://api.acme.com/a/b/c",
    matches: false,
  },
] as const
