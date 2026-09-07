package resourcenames

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	workflowresource "github.com/yyhuni/lunafox/contracts/scanworkflow"
)

const (
	agentsCollection        = "agents"
	sessionsCollection      = "sessions"
	scansCollection         = "scans"
	targetsCollection       = "targets"
	tasksCollection         = "tasks"
	wordlistsCollection     = "wordlists"
	scanWorkflowsCollection = "scanWorkflows"
	executionsCollection    = "executions"
	commandsCollection      = "commands"
	enginesCollection       = "engines"
	packagesCollection      = "packages"
	blacklistPolicyResource = "blacklistPolicy"
)

func Agent(agentID string) string {
	return formatStringResource(agentsCollection, agentID)
}

func ParseAgent(name string) (string, error) {
	return parseStringResource(name, agentsCollection)
}

func AgentSession(agentID, sessionID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", agentsCollection, strings.TrimSpace(agentID), sessionsCollection, strings.TrimSpace(sessionID))
}

func ParseAgentSession(name string) (agentID string, sessionID string, err error) {
	parts := splitResourceName(name)
	if len(parts) != 4 || parts[0] != agentsCollection || parts[2] != sessionsCollection {
		return "", "", fmt.Errorf("agent session name must use %s/{agent}/%s/{session}", agentsCollection, sessionsCollection)
	}
	agentID = strings.TrimSpace(parts[1])
	if agentID == "" {
		return "", "", fmt.Errorf("agent resource id is required")
	}
	sessionID = strings.TrimSpace(parts[3])
	if sessionID == "" {
		return "", "", fmt.Errorf("session resource id is required")
	}
	return agentID, sessionID, nil
}

// Scan returns the canonical scan resource name.
func Scan(scanID int) string {
	return formatIntResource(scansCollection, scanID)
}

// ParseScan parses a canonical scan resource name.
func ParseScan(name string) (int, error) {
	return parseIntResource(name, scansCollection)
}

func Target(targetID int) string {
	return formatIntResource(targetsCollection, targetID)
}

func ParseTarget(name string) (int, error) {
	return parseIntResource(name, targetsCollection)
}

// BlacklistPolicy returns the canonical global blacklist singleton name.
func BlacklistPolicy() string {
	return blacklistPolicyResource
}

// ParseBlacklistPolicy validates the canonical global blacklist singleton name.
func ParseBlacklistPolicy(name string) error {
	if strings.TrimSpace(name) != blacklistPolicyResource {
		return fmt.Errorf("blacklist policy name must be %s", blacklistPolicyResource)
	}
	return nil
}

// TargetBlacklistPolicy returns the canonical Target-local blacklist singleton name.
func TargetBlacklistPolicy(targetID int) string {
	return fmt.Sprintf("%s/%d/%s", targetsCollection, targetID, blacklistPolicyResource)
}

// ParseTargetBlacklistPolicy parses a canonical Target-local blacklist singleton name.
func ParseTargetBlacklistPolicy(name string) (int, error) {
	parts := splitResourceName(name)
	if len(parts) != 3 || parts[0] != targetsCollection || parts[2] != blacklistPolicyResource {
		return 0, fmt.Errorf("target blacklist policy name must use %s/{target}/%s", targetsCollection, blacklistPolicyResource)
	}
	return parsePositiveInt(parts[1], "target")
}

func Task(scanID, taskID int) string {
	return fmt.Sprintf("%s/%d/%s/%d", scansCollection, scanID, tasksCollection, taskID)
}

func ParseTask(name string) (scanID int, taskID int, err error) {
	parts := splitResourceName(name)
	if len(parts) != 4 || parts[0] != scansCollection || parts[2] != tasksCollection {
		return 0, 0, fmt.Errorf("task name must use %s/{scan}/%s/{task}", scansCollection, tasksCollection)
	}
	scanID, err = parsePositiveInt(parts[1], "scan")
	if err != nil {
		return 0, 0, err
	}
	taskID, err = parsePositiveInt(parts[3], "task")
	if err != nil {
		return 0, 0, err
	}
	return scanID, taskID, nil
}

// Wordlist returns the canonical Wordlist resource name.
func Wordlist(wordlistID int) string {
	return formatIntResource(wordlistsCollection, wordlistID)
}

// ParseWordlist parses a canonical Wordlist resource name.
func ParseWordlist(name string) (int, error) {
	return parseIntResource(name, wordlistsCollection)
}

func ScanWorkflow(workflowID string) string {
	return workflowresource.ScanWorkflowName(workflowID)
}

func ParseScanWorkflow(name string) (string, error) {
	return workflowresource.ParseScanWorkflowName(name)
}

func Engine(engineID string) string {
	return formatStringResource(enginesCollection, engineID)
}

func ParseEngine(name string) (string, error) {
	parts := splitResourceName(name)
	if len(parts) != 2 || parts[0] != enginesCollection {
		return "", fmt.Errorf("engine name must use %s/{engine}", enginesCollection)
	}
	value := strings.TrimSpace(parts[1])
	if value == "" {
		return "", fmt.Errorf("engine resource id is required")
	}
	return value, nil
}

func EnginePackage(engineID, packageID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", enginesCollection, strings.TrimSpace(engineID), packagesCollection, strings.TrimSpace(packageID))
}

func ParseEnginePackage(name string) (engineID string, packageID string, err error) {
	parts := splitResourceName(name)
	if len(parts) != 4 || parts[0] != enginesCollection || parts[2] != packagesCollection {
		return "", "", fmt.Errorf("engine package name must use %s/{engine}/%s/{package}", enginesCollection, packagesCollection)
	}
	engineID = strings.TrimSpace(parts[1])
	if engineID == "" {
		return "", "", fmt.Errorf("engine resource id is required")
	}
	packageID = strings.TrimSpace(parts[3])
	if packageID == "" {
		return "", "", fmt.Errorf("package resource id is required")
	}
	if !isEnginePackageResourceID(packageID) {
		return "", "", fmt.Errorf("package resource id must use lower-case letters, digits, and hyphens")
	}
	return engineID, packageID, nil
}

func Execution(executionID string) string {
	return formatStringResource(executionsCollection, executionID)
}

func ParseExecution(name string) (string, error) {
	return parseStringResource(name, executionsCollection)
}

func ExecutionCommand(executionID, commandID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", executionsCollection, strings.TrimSpace(executionID), commandsCollection, strings.TrimSpace(commandID))
}

func ParseExecutionCommand(name string) (executionID string, commandID string, err error) {
	parts := splitResourceName(name)
	if len(parts) != 4 || parts[0] != executionsCollection || parts[2] != commandsCollection {
		return "", "", fmt.Errorf("execution command name must use %s/{execution}/%s/{command}", executionsCollection, commandsCollection)
	}
	executionID = strings.TrimSpace(parts[1])
	if executionID == "" {
		return "", "", fmt.Errorf("execution resource id is required")
	}
	commandID = strings.TrimSpace(parts[3])
	if commandID == "" {
		return "", "", fmt.Errorf("command resource id is required")
	}
	return executionID, commandID, nil
}

func formatIntResource(collection string, id int) string {
	return fmt.Sprintf("%s/%d", collection, id)
}

func parseIntResource(name, collection string) (int, error) {
	value, err := parseStringResource(name, collection)
	if err != nil {
		return 0, err
	}
	return parsePositiveInt(value, strings.TrimSuffix(collection, "s"))
}

func formatStringResource(collection, value string) string {
	return collection + "/" + strings.TrimSpace(value)
}

func parseStringResource(name, collection string) (string, error) {
	parts := splitResourceName(name)
	if len(parts) != 2 || parts[0] != collection {
		return "", fmt.Errorf("resource name must use %s/{resource}", collection)
	}
	value := strings.TrimSpace(parts[1])
	if value == "" {
		return "", fmt.Errorf("%s resource id is required", strings.TrimSuffix(collection, "s"))
	}
	return value, nil
}

func splitResourceName(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	parts := strings.Split(name, "/")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func parsePositiveInt(value, label string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s resource id must be a positive integer", label)
	}
	return parsed, nil
}

func isEnginePackageResourceID(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if unicode.IsLower(r) || unicode.IsDigit(r) {
			continue
		}
		if r == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}
