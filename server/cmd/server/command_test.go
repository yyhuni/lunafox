package main

import (
	"strings"
	"testing"
)

func TestParseServerCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantKind       serverCommandKind
		wantCorpusPath string
		wantErr        bool
	}{
		{
			name:     "server runs normally without a subcommand",
			args:     []string{"server"},
			wantKind: serverCommandRun,
		},
		{
			name:     "engine bootstrap remains supported",
			args:     []string{"server", "engine-bootstrap"},
			wantKind: serverCommandEngineBootstrap,
		},
		{
			name:           "fingerprint bootstrap accepts one absolute staged corpus path",
			args:           []string{"server", "fingerprint-bootstrap", "/tmp/lunafox/fingerprint-bootstrap/web_fingerprint_v4.json"},
			wantKind:       serverCommandFingerprintBootstrap,
			wantCorpusPath: "/tmp/lunafox/fingerprint-bootstrap/web_fingerprint_v4.json",
		},
		{
			name:     "wordlist bootstrap accepts absolute manifest and source directory",
			args:     []string{"server", "wordlist-bootstrap", "/tmp/lunafox/wordlists/manifest.json", "/tmp/lunafox/wordlists"},
			wantKind: serverCommandWordlistBootstrap,
		},
		{
			name:     "agent bootstrap accepts an absolute credentials path and explicit identity",
			args:     []string{"server", "agent-bootstrap", "/run/lunafox-bootstrap/agent.json", "lunafox-agent", "0.0.1-alpha.47"},
			wantKind: serverCommandAgentBootstrap,
		},
		{
			name:     "server resetadmin uses bounded reset command",
			args:     []string{"server", "resetadmin"},
			wantKind: serverCommandResetAdmin,
		},
		{
			name:     "wordlist resource migration uses bounded database command",
			args:     []string{"server", "wordlist-resource-migrate"},
			wantKind: serverCommandWordlistResourceMigration,
		},
		{
			name:     "image path executable resetadmin uses bounded reset command",
			args:     []string{"resetadmin"},
			wantKind: serverCommandResetAdmin,
		},
		{
			name:    "removed reset alias is rejected",
			args:    []string{"server", "reset-admin-password"},
			wantErr: true,
		},
		{
			name:    "unknown command is rejected",
			args:    []string{"server", "unknown-command"},
			wantErr: true,
		},
		{
			name:    "fingerprint bootstrap requires an absolute path",
			args:    []string{"server", "fingerprint-bootstrap", "web_fingerprint_v4.json"},
			wantErr: true,
		},
		{
			name:    "fingerprint bootstrap rejects missing path",
			args:    []string{"server", "fingerprint-bootstrap"},
			wantErr: true,
		},
		{
			name:    "fingerprint bootstrap rejects extra arguments",
			args:    []string{"server", "fingerprint-bootstrap", "/tmp/corpus.json", "unexpected"},
			wantErr: true,
		},
		{
			name:    "wordlist bootstrap requires absolute paths",
			args:    []string{"server", "wordlist-bootstrap", "manifest.json", "/tmp/wordlists"},
			wantErr: true,
		},
		{
			name:    "agent bootstrap requires an absolute credentials path",
			args:    []string{"server", "agent-bootstrap", "agent.json", "lunafox-agent", "0.0.1-alpha.47"},
			wantErr: true,
		},
		{
			name:    "reset executable rejects positional arguments",
			args:    []string{"resetadmin", "unexpected"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseServerCommand(test.args)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected command parsing to fail")
				}
				if !strings.Contains(err.Error(), "Usage:") {
					t.Fatalf("error = %q, want usage", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse command: %v", err)
			}
			if got.kind != test.wantKind {
				t.Fatalf("command kind = %d, want %d", got.kind, test.wantKind)
			}
			if got.fingerprintBootstrapPath != test.wantCorpusPath {
				t.Fatalf("fingerprint bootstrap path = %q, want %q", got.fingerprintBootstrapPath, test.wantCorpusPath)
			}
		})
	}
}
