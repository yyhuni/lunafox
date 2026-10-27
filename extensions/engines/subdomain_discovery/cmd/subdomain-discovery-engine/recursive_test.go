package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	enginecontract "github.com/yyhuni/lunafox/engines/subdomain_discovery/contract"
)

func TestRunSubdomainDiscoveryReportsOnlyResolveAfterRecursiveSuccess(t *testing.T) {
	for _, failRecursive := range []string{"0", "1"} {
		t.Run("recursive-failure="+failRecursive, func(t *testing.T) {
			binDir := t.TempDir()
			callsPath := filepath.Join(t.TempDir(), "puredns.calls")
			writeExecutable(t, filepath.Join(binDir, "subfinder"), `#!/bin/sh
set -eu
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf 'passive.example.com\n' > "$out"
`)
			writeExecutable(t, filepath.Join(binDir, "puredns"), `#!/bin/sh
set -eu
mode="$1"
parent=""
case "$mode" in
  bruteforce) parent="$3"; shift 3 ;;
  resolve) shift 2 ;;
  *) exit 64 ;;
esac
while [ "$#" -gt 0 ]; do
  case "$1" in
    --write) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s %s\n' "$mode" "$parent" >> "$PUREDNS_CALLS_FILE"
case "$mode:$parent" in
  bruteforce:example.com) printf 'api.example.com\n' > "$out" ;;
  bruteforce:api.example.com)
    if [ "$FAIL_RECURSIVE" = 1 ]; then exit 13; fi
    printf 'deep.api.example.com\n' > "$out" ;;
  resolve:) printf 'deep.api.example.com\n' > "$out" ;;
  *) exit 65 ;;
esac
`)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PUREDNS_CALLS_FILE", callsPath)
			t.Setenv("FAIL_RECURSIVE", failRecursive)
			results := &captureSubdomains{}
			execution := subdomainExecution(t.TempDir(), "/wordlist.txt", "/resolvers.txt", "/provider.yaml", results)
			execution.Config.Bruteforce.Enabled = true
			execution.Config.Bruteforce.Recursive = true
			execution.Config.Bruteforce.RecursiveDepth = 1

			err := runSubdomainDiscovery(context.Background(), execution)

			if failRecursive == "1" {
				require.Error(t, err)
				require.Zero(t, results.calls)
			} else {
				require.NoError(t, err)
				require.Equal(t, []enginecontract.Subdomain{{DNSName: "deep.api.example.com"}}, results.items)
			}
			calls, err := os.ReadFile(callsPath)
			require.NoError(t, err)
			require.Equal(t, "bruteforce example.com\nbruteforce api.example.com\nresolve \n", string(calls))
		})
	}
}
