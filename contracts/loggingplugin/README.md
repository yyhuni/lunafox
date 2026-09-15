# Loki installer helper

`policy.sh` is the single source for the supported official plugin version and
daemon architectures. `manager.sh` is shared executable installer behavior;
it is not a protocol contract. Go callers embed these files through `Script()`;
shell callers source them. The caller defines `lunafox_plugin_docker` and must
preserve the Docker command exit status and daemon selection.

`lunafox_plugin_prepare install <reference>` may update a known, unreferenced
plugin. `start` only installs a missing plugin or enables the exact deployed
version. `lunafox_plugin_remove` never forces removal and returns an error when
identity, ownership or container references cannot be verified.

Ownership uses a separate empty daemon volume named
`lunafox_loki_plugin_<full-plugin-ID>`, labeled with `io.lunafox.logging-plugin`
equal to the ID. This metadata is outside business volumes and receipts. Missing
records are not adopted automatically. A failed later business installation must
retain this volume and the plugin. An administrator can alter these records;
they provide management provenance, not a security boundary against daemon admins.

All containers, including stopped and unrelated containers, participate in
reference checks. Enumeration/inspection failure is never interpreted as no
references. Docker non-force guards remain the final defense against races.
If upgrade changes the plugin ID, preparation stops instead of silently adopting
the replacement. The generic `loki` plugin is never managed by these helpers.

Docker plugin inspect exposes `Id` (not the container-style `ID`) and may
normalize Hub references to `docker.io/grafana/...`. Interface types are rendered
with a Go-template range because the CLI exposes them as an untyped array.
Explicit version changes use `--skip-remote-check` after validating the official
repository and ownership, suppressing only the changed-reference prompt, not
Docker's non-force reference guards.

Run `go test ./loggingplugin` from `contracts` for shell state-machine tests.
