package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type serverCommandKind uint8

const (
	serverCommandRun serverCommandKind = iota
	serverCommandEngineBootstrap
	serverCommandFingerprintBootstrap
	serverCommandWordlistBootstrap
	serverCommandAgentBootstrap
	serverCommandResetAdmin
	serverCommandWordlistResourceMigration
)

type serverCommand struct {
	kind                     serverCommandKind
	program                  string
	fingerprintBootstrapPath string
	wordlistManifestPath     string
	wordlistSourcePath       string
	agentCredentialsPath     string
	agentHostname            string
	agentVersion             string
}

func parseServerCommand(args []string) (serverCommand, error) {
	if len(args) == 0 {
		return serverCommand{}, fmt.Errorf("server command arguments are required")
	}

	program := filepath.Base(args[0])
	if program == "resetadmin" {
		if len(args) != 1 {
			return serverCommand{program: program}, fmt.Errorf("%s", serverUsage(program))
		}
		return serverCommand{kind: serverCommandResetAdmin, program: program}, nil
	}

	if len(args) == 1 {
		return serverCommand{kind: serverCommandRun, program: program}, nil
	}
	if len(args) == 3 && args[1] == "fingerprint-bootstrap" {
		corpusPath := strings.TrimSpace(args[2])
		if corpusPath == "" || !filepath.IsAbs(corpusPath) {
			return serverCommand{program: program}, fmt.Errorf("%s", serverUsage(program))
		}
		return serverCommand{kind: serverCommandFingerprintBootstrap, program: program, fingerprintBootstrapPath: corpusPath}, nil
	}
	if len(args) == 4 && args[1] == "wordlist-bootstrap" {
		manifestPath := strings.TrimSpace(args[2])
		sourcePath := strings.TrimSpace(args[3])
		if manifestPath == "" || sourcePath == "" || !filepath.IsAbs(manifestPath) || !filepath.IsAbs(sourcePath) {
			return serverCommand{program: program}, fmt.Errorf("%s", serverUsage(program))
		}
		return serverCommand{
			kind:                 serverCommandWordlistBootstrap,
			program:              program,
			wordlistManifestPath: manifestPath,
			wordlistSourcePath:   sourcePath,
		}, nil
	}
	if len(args) == 5 && args[1] == "agent-bootstrap" {
		credentialsPath := strings.TrimSpace(args[2])
		hostname := strings.TrimSpace(args[3])
		version := strings.TrimSpace(args[4])
		if credentialsPath == "" || hostname == "" || version == "" || !filepath.IsAbs(credentialsPath) {
			return serverCommand{program: program}, fmt.Errorf("%s", serverUsage(program))
		}
		return serverCommand{
			kind:                 serverCommandAgentBootstrap,
			program:              program,
			agentCredentialsPath: credentialsPath,
			agentHostname:        hostname,
			agentVersion:         version,
		}, nil
	}
	if len(args) == 2 {
		switch args[1] {
		case "engine-bootstrap":
			return serverCommand{kind: serverCommandEngineBootstrap, program: program}, nil
		case "resetadmin":
			return serverCommand{kind: serverCommandResetAdmin, program: program}, nil
		case "wordlist-resource-migrate":
			return serverCommand{kind: serverCommandWordlistResourceMigration, program: program}, nil
		}
	}

	return serverCommand{program: program}, fmt.Errorf("%s", serverUsage(program))
}

func serverUsage(program string) string {
	if program == "" || program == "." {
		program = "server"
	}
	return fmt.Sprintf("Usage: %s [engine-bootstrap|fingerprint-bootstrap <staged-corpus-path>|wordlist-bootstrap <manifest-path> <source-directory>|agent-bootstrap <credentials-path> <hostname> <agent-version>|resetadmin|wordlist-resource-migrate]\n", program)
}
