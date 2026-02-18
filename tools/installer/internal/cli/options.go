package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ModeProd = "prod"
	ModeDev  = "dev"
)

type Options struct {
	Mode            string
	Version         string
	UseGoProxyCN    bool
	PublicURL       string
	PublicPort      string
	ImageRegistry   string
	ImageNamespace  string
	AgentImageRef   string
	AgentImageRefs  []string
	WorkerImageRef  string
	WorkerImageRefs []string
	AgentServerURL  string
	AgentNetwork    string
	SharedDataBind  string

	RootDir     string
	DockerDir   string
	EnvFile     string
	ComposeFile string
	ComposeDev  string
	ComposeProd string
	Go111Module string
	GoProxy     string
	ListenAddr  string
}

func Parse(args []string) (Options, error) {
	dev := false
	version := ""
	useGoProxyCN := false
	imageRegistry := firstNonEmpty(os.Getenv("LUNAFOX_IMAGE_REGISTRY"), DefaultImageRegistry)
	imageNamespace := firstNonEmpty(os.Getenv("LUNAFOX_IMAGE_NAMESPACE"), DefaultImageNamespace)
	agentImageRefsRaw := strings.TrimSpace(os.Getenv("AGENT_IMAGE_REFS"))
	workerImageRefsRaw := strings.TrimSpace(os.Getenv("WORKER_IMAGE_REFS"))
	publicURL := DefaultPublicURL
	publicPort := DefaultPublicPort
	agentServerURL := firstNonEmpty(os.Getenv("LUNAFOX_AGENT_SERVER_URL"), DefaultAgentServerURL)
	agentNetwork := firstNonEmpty(os.Getenv("LUNAFOX_AGENT_DOCKER_NETWORK"), DefaultAgentNetwork)
	sharedDataBind := firstNonEmpty(os.Getenv("LUNAFOX_SHARED_DATA_VOLUME_BIND"), DefaultSharedDataBind)
	rootDirInput := ""
	listenAddr := DefaultListenAddr

	fs := flag.NewFlagSet("lunafox-installer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&dev, "dev", false, "开发模式")
	fs.StringVar(&version, "version", strings.TrimSpace(os.Getenv("LUNAFOX_INSTALLER_VERSION")), "安装版本，例如 v1.5.13")
	fs.BoolVar(&useGoProxyCN, "goproxy", false, "使用 goproxy.cn")
	fs.StringVar(&imageRegistry, "image-registry", imageRegistry, "镜像仓库域名，例如 docker.io 或 ghcr.io")
	fs.StringVar(&imageNamespace, "image-namespace", imageNamespace, "镜像命名空间，例如 yyhuni")
	fs.StringVar(&agentImageRefsRaw, "agent-image-refs", agentImageRefsRaw, "Agent 镜像候选列表（逗号分隔，按优先级）")
	fs.StringVar(&workerImageRefsRaw, "worker-image-refs", workerImageRefsRaw, "Worker 镜像候选列表（逗号分隔，按优先级）")
	fs.StringVar(&rootDirInput, "root-dir", "", "项目根目录（必填）")
	fs.StringVar(&listenAddr, "listen", listenAddr, "Web 页面监听地址，例如 127.0.0.1:18083")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	mode := ModeProd
	if dev {
		mode = ModeDev
	}

	version = strings.TrimSpace(version)
	imageRegistry = strings.Trim(strings.TrimSpace(imageRegistry), "/")
	if imageRegistry == "" {
		return Options{}, fmt.Errorf("--image-registry 不能为空")
	}
	imageNamespace = strings.Trim(strings.TrimSpace(imageNamespace), "/")
	if imageNamespace == "" {
		return Options{}, fmt.Errorf("--image-namespace 不能为空")
	}
	sharedDataBind = strings.TrimSpace(sharedDataBind)
	if sharedDataBind == "" {
		return Options{}, fmt.Errorf("LUNAFOX_SHARED_DATA_VOLUME_BIND 不能为空")
	}
	// Single-value refs were intentionally removed; parse candidates only.
	agentImageRefs := parseImageRefs(agentImageRefsRaw)
	workerImageRefs := parseImageRefs(workerImageRefsRaw)
	agentImageRef := ""
	workerImageRef := ""
	if len(agentImageRefs) > 0 {
		agentImageRef = agentImageRefs[0]
	}
	if len(workerImageRefs) > 0 {
		workerImageRef = workerImageRefs[0]
	}

	if mode == ModeProd {
		// Production is digest-only to guarantee immutable pull targets.
		if len(agentImageRefs) == 0 {
			return Options{}, fmt.Errorf("生产模式必须提供 --agent-image-refs")
		}
		if len(workerImageRefs) == 0 {
			return Options{}, fmt.Errorf("生产模式必须提供 --worker-image-refs")
		}
		for _, ref := range agentImageRefs {
			if !isDigestImageRef(ref) {
				return Options{}, fmt.Errorf("生产模式要求 --agent-image-refs 使用 digest（@sha256:...）")
			}
		}
		for _, ref := range workerImageRefs {
			if !isDigestImageRef(ref) {
				return Options{}, fmt.Errorf("生产模式要求 --worker-image-refs 使用 digest（@sha256:...）")
			}
		}
	} else {
		for _, ref := range agentImageRefs {
			if !hasImageTagOrDigest(ref) {
				return Options{}, fmt.Errorf("--agent-image-refs 必须包含 tag 或 digest")
			}
		}
		for _, ref := range workerImageRefs {
			if !hasImageTagOrDigest(ref) {
				return Options{}, fmt.Errorf("--worker-image-refs 必须包含 tag 或 digest")
			}
		}
	}

	goProxy := "https://proxy.golang.org,direct"
	if useGoProxyCN {
		goProxy = "https://goproxy.cn,direct"
	}

	rootDir, err := resolveRootDir(rootDirInput)
	if err != nil {
		return Options{}, err
	}

	dockerDir := filepath.Join(rootDir, "docker")
	composeDev := filepath.Join(dockerDir, "docker-compose.dev.yml")
	composeProd := filepath.Join(dockerDir, "docker-compose.yml")
	composeFile := composeProd
	if mode == ModeDev {
		composeFile = composeDev
	}

	return Options{
		Mode:            mode,
		Version:         version,
		UseGoProxyCN:    useGoProxyCN,
		PublicURL:       publicURL,
		PublicPort:      publicPort,
		ImageRegistry:   imageRegistry,
		ImageNamespace:  imageNamespace,
		AgentImageRef:   agentImageRef,
		AgentImageRefs:  agentImageRefs,
		WorkerImageRef:  workerImageRef,
		WorkerImageRefs: workerImageRefs,
		AgentServerURL:  agentServerURL,
		AgentNetwork:    agentNetwork,
		SharedDataBind:  sharedDataBind,
		RootDir:         rootDir,
		DockerDir:       dockerDir,
		EnvFile:         filepath.Join(dockerDir, ".env"),
		ComposeFile:     composeFile,
		ComposeDev:      composeDev,
		ComposeProd:     composeProd,
		Go111Module:     "on",
		GoProxy:         goProxy,
		ListenAddr:      strings.TrimSpace(listenAddr),
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func resolveRootDir(preferred string) (string, error) {
	trimmed := strings.TrimSpace(preferred)
	if trimmed == "" {
		return "", fmt.Errorf("--root-dir 不能为空")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("解析 --root-dir 失败: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("规范化 --root-dir 失败: %w", err)
	}
	if !isProjectRoot(canonical) {
		return "", fmt.Errorf("--root-dir 不是有效的 LunaFox 项目目录: %s", canonical)
	}
	return canonical, nil
}

func isProjectRoot(root string) bool {
	dockerDir := filepath.Join(root, "docker")
	composeFile := filepath.Join(dockerDir, "docker-compose.yml")
	composeDevFile := filepath.Join(dockerDir, "docker-compose.dev.yml")
	installerEntry := filepath.Join(root, "tools", "installer", "cmd", "lunafox-installer", "main.go")
	if stat, err := os.Stat(dockerDir); err != nil || !stat.IsDir() {
		return false
	}
	if _, err := os.Stat(composeFile); err != nil {
		return false
	}
	if _, err := os.Stat(composeDevFile); err != nil {
		return false
	}
	if _, err := os.Stat(installerEntry); err != nil {
		return false
	}
	return true
}

func hasImageTagOrDigest(imageRef string) bool {
	if strings.Contains(imageRef, "@") {
		return true
	}
	return strings.LastIndex(imageRef, ":") > strings.LastIndex(imageRef, "/")
}

func isDigestImageRef(imageRef string) bool {
	ref := strings.TrimSpace(imageRef)
	at := strings.LastIndex(ref, "@")
	if at <= 0 || at == len(ref)-1 {
		return false
	}
	return strings.HasPrefix(strings.ToLower(ref[at+1:]), "sha256:")
}

func parseImageRefs(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		ref := strings.TrimSpace(part)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		result = append(result, ref)
	}
	return result
}
