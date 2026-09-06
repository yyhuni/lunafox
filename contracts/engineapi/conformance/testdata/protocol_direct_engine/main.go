package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yyhuni/lunafox/engine-go/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	canonicalSubdomainResultType    = "asset.subdomain.v1"
	protocolDirectBootstrapDeadline = 5 * time.Second
)

type fixturePaths struct {
	contextPath    string
	credentialPath string
	endpointPath   string
}

func main() {
	root := flag.String("test-runtime-root", "", "test-only root projected before fixed Engine ABI paths")
	flag.Parse()
	if flag.NArg() != 0 {
		_, _ = fmt.Fprintln(os.Stderr, "protocol-direct fixture: unexpected positional arguments")
		os.Exit(1)
	}
	paths, err := newFixturePaths(*root)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "protocol-direct fixture: %v\n", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), protocolDirectBootstrapDeadline)
	defer cancel()
	if err := run(ctx, paths); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "protocol-direct fixture: %v\n", err)
		os.Exit(1)
	}
}

// newFixturePaths only exists in this conformance binary. Production Engine
// code uses protocol's absolute ABI constants directly and exposes no path
// override through environment variables or public SDK API.
func newFixturePaths(root string) (fixturePaths, error) {
	if root == "" {
		return fixturePaths{
			contextPath:    protocol.ContextFilePath,
			credentialPath: protocol.CredentialFilePath,
			endpointPath:   protocol.ExecutionEndpointPath,
		}, nil
	}
	if root != strings.TrimSpace(root) || !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return fixturePaths{}, errors.New("test runtime root must be absolute and clean")
	}
	join := func(canonical string) string {
		return filepath.Join(root, strings.TrimPrefix(canonical, "/"))
	}
	return fixturePaths{
		contextPath:    join(protocol.ContextFilePath),
		credentialPath: join(protocol.CredentialFilePath),
		endpointPath:   join(protocol.ExecutionEndpointPath),
	}, nil
}

func run(ctx context.Context, paths fixturePaths) error {
	executionContext, inputPresence, err := loadExecutionContext(paths.contextPath)
	if err != nil {
		return err
	}
	token, err := loadCredential(paths.credentialPath)
	if err != nil {
		return err
	}
	connection, err := dialTaskSocket(ctx, paths.endpointPath)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close() }()

	client := protocol.NewEngineExecutionReportingServiceClient(connection)
	authorizedContext := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+token))
	progressMessage := "protocol-direct-ready inputs=" + inputPresence
	if uint64(len(progressMessage)) > uint64(executionContext.GetLimits().GetProgressMessageMaxBytes()) {
		return errors.New("progress message exceeds Context limit")
	}
	progressResponse, err := client.ReportProgress(authorizedContext, &protocol.ReportProgressRequest{Message: progressMessage})
	if err != nil {
		return fmt.Errorf("report progress: %w", err)
	}
	if progressResponse == nil || progressResponse.ProtoReflect().Descriptor().Fields().Len() != 0 {
		return errors.New("progress acknowledgement is not status-only")
	}

	resultItems, err := canonicalResultItems(executionContext.GetTarget())
	if err != nil {
		return err
	}
	if uint64(len(resultItems)) > uint64(executionContext.GetLimits().GetResultBatchMaxItems()) {
		return errors.New("result item count exceeds Context limit")
	}
	var resultBytes uint64
	for _, item := range resultItems {
		resultBytes += uint64(len(item))
	}
	if resultBytes > uint64(executionContext.GetLimits().GetResultBatchMaxBytes()) {
		return errors.New("result bytes exceed Context limit")
	}
	resultResponse, err := client.SubmitResultBatch(authorizedContext, &protocol.SubmitResultBatchRequest{
		ResultType: canonicalSubdomainResultType,
		Items:      resultItems,
	})
	if err != nil {
		return fmt.Errorf("submit result batch: %w", err)
	}
	if resultResponse == nil || resultResponse.ProtoReflect().Descriptor().Fields().Len() != 0 {
		return errors.New("result acknowledgement is not status-only")
	}
	return nil
}

func loadExecutionContext(path string) (*protocol.EngineExecutionContext, string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read binary execution Context: %w", err)
	}
	var executionContext protocol.EngineExecutionContext
	if err := proto.Unmarshal(payload, &executionContext); err != nil {
		return nil, "", errors.New("decode binary execution Context")
	}
	if hasUnknownFields(executionContext.ProtoReflect()) {
		return nil, "", errors.New("execution Context contains unsupported fields")
	}
	inputPresence, err := validateExecutionContext(&executionContext)
	if err != nil {
		return nil, "", err
	}
	return &executionContext, inputPresence, nil
}

func validateExecutionContext(executionContext *protocol.EngineExecutionContext) (string, error) {
	if executionContext == nil || executionContext.GetTarget() == nil || executionContext.GetConfig() == nil {
		return "", errors.New("execution Context required fields are missing")
	}
	target := executionContext.GetTarget()
	if err := requireCanonicalText("Target type", target.GetType()); err != nil {
		return "", err
	}
	if err := requireCanonicalText("Target value", target.GetValue()); err != nil {
		return "", err
	}
	switch target.GetType() {
	case "domain", "ip", "cidr":
	default:
		return "", errors.New("canonical Target type is unsupported")
	}
	if err := protocol.ValidateLimits(executionContext.GetLimits()); err != nil {
		return "", err
	}

	sections := make(map[string]bool, len(executionContext.GetConfig().GetSections()))
	for _, section := range executionContext.GetConfig().GetSections() {
		if section == nil || section.Enabled == nil || requireCanonicalText("config section ID", section.GetSectionId()) != nil {
			return "", errors.New("config section is invalid")
		}
		if _, duplicate := sections[section.GetSectionId()]; duplicate {
			return "", errors.New("duplicate config section ID")
		}
		sections[section.GetSectionId()] = section.GetEnabled()
		for _, param := range section.GetParams() {
			if param == nil || param.GetValue() == nil || requireCanonicalText("config parameter key", param.GetParamKey()) != nil || !section.GetEnabled() {
				return "", errors.New("config parameter is invalid")
			}
		}
	}
	for _, resource := range executionContext.GetConfigResources() {
		if resource == nil || requireCanonicalText("config resource section ID", resource.GetSectionId()) != nil || requireCanonicalText("config resource parameter key", resource.GetParamKey()) != nil {
			return "", errors.New("config resource is invalid")
		}
		enabled, present := sections[resource.GetSectionId()]
		if !present || !enabled || validateReadableBinding("config resource", resource.GetPath(), "application/vnd.lunafox.wordlist.v1", resource.GetContentType()) != nil {
			return "", errors.New("config resource is invalid")
		}
	}
	for _, resource := range executionContext.GetPlatformResources() {
		if resource == nil || requireCanonicalText("platform resource ID", resource.GetResourceId()) != nil || validateReadableBinding("platform resource", resource.GetPath(), "application/vnd.lunafox.subfinder-provider-config.v1+yaml", resource.GetContentType()) != nil {
			return "", errors.New("platform resource is invalid")
		}
	}

	return "none", nil
}

func validateReadableBinding(_ string, path, wantContentType, gotContentType string) error {
	if gotContentType != wantContentType || path == "" || path != strings.TrimSpace(path) || !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return errors.New("binding shape is invalid")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("binding must be a regular file")
	}
	return nil
}

func requireCanonicalText(_ string, value string) error {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) {
		return errors.New("text must be non-empty canonical UTF-8")
	}
	return nil
}

func hasUnknownFields(message protoreflect.Message) bool {
	if !message.IsValid() || len(message.GetUnknown()) != 0 {
		return true
	}
	var unknown bool
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.IsList() {
			for index := 0; index < value.List().Len(); index++ {
				if field.Kind() == protoreflect.MessageKind && hasUnknownFields(value.List().Get(index).Message()) {
					unknown = true
					return false
				}
			}
			return true
		}
		if field.Kind() == protoreflect.MessageKind && hasUnknownFields(value.Message()) {
			unknown = true
			return false
		}
		return true
	})
	return unknown
}

func loadCredential(path string) (string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read execution credential: %w", err)
	}
	if len(payload) != 43 {
		return "", errors.New("execution credential is not canonical")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(string(payload))
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != string(payload) {
		return "", errors.New("execution credential is not canonical")
	}
	return string(payload), nil
}

func dialTaskSocket(ctx context.Context, endpoint string) (*grpc.ClientConn, error) {
	info, err := os.Lstat(endpoint)
	if err != nil {
		return nil, fmt.Errorf("inspect reporting socket: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.Mode()&os.ModeSocket == 0 {
		return nil, errors.New("reporting endpoint must be a filesystem UDS")
	}
	connection, err := grpc.NewClient(
		"passthrough:///lunafox-engine-execution-v2",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDisableRetry(),
		grpc.WithContextDialer(func(dialContext context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(dialContext, "unix", endpoint)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create reporting client: %w", err)
	}
	if err := waitForReady(ctx, connection); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return connection, nil
}

func waitForReady(ctx context.Context, connection *grpc.ClientConn) error {
	connection.Connect()
	for {
		state := connection.GetState()
		switch state {
		case connectivity.Ready:
			return nil
		case connectivity.Shutdown:
			return errors.New("reporting connection shut down before Ready")
		}
		if !connection.WaitForStateChange(ctx, state) {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("wait for reporting connection Ready: %w", err)
			}
			return errors.New("reporting connection did not reach Ready")
		}
	}
}

func canonicalResultItems(target *protocol.CanonicalTarget) ([][]byte, error) {
	type subdomainResult struct {
		DNSName string `json:"dnsName"`
	}
	items := make([][]byte, 0, 2)
	for _, prefix := range []string{"api.", "www."} {
		item, err := json.Marshal(subdomainResult{DNSName: prefix + target.GetValue()})
		if err != nil {
			return nil, fmt.Errorf("encode canonical result item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}
