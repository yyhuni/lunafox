"""Offline fixture for both HTTPX probing and Chromium header delivery."""

import json
from http.server import BaseHTTPRequestHandler, HTTPServer

EXPECTED = {
    "Cookie": "a=1; b=two",
    "Authorization": "Bearer local-test-token",
    "X-Fields": 'a,b:c; "quoted"',
}


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.end_headers()
            return
        matched = all(self.headers.get(key) == value for key, value in EXPECTED.items())
        print(json.dumps({
            "browser": self.headers.get("Sec-Fetch-Dest") == "document",
            "matched": matched,
        }), flush=True)
        self.send_response(200 if matched else 401)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(b"<!doctype html><title>LunaFox Screenshot</title><main>offline conformance</main>")

    def log_message(self, *_args):
        pass


HTTPServer(("127.0.0.1", 18080), Handler).serve_forever()
