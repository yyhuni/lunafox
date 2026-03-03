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

type PublicAddressSource string

const (
	PublicAddressSourceDefault  PublicAddressSource = "default"
	PublicAddressSourceURL      PublicAddressSource = "public-url"
	PublicAddressSourceHostPort PublicAddressSource = "public-host-port"
)

type Options struct {
	Mode                string
	ReleaseVersion      string
	UseGoProxyCN        bool
	PublicURL           string
	PublicPort          string
	PublicAddressSource PublicAddressSource
	NonInteractive      bool
	ImageRegistry       string
	ImageNamespace      string
	AgentImageRef       string
	AgentImageRefs      []string
	WorkerImageRef      string
	WorkerImageRefs     []string
	AgentNetwork        string
	SharedDataBind      string
	ReleaseManifest     string

	RootDir     string
	DockerDir   string
	EnvFile     string
	ComposeFile string
	ComposeDev  string
	ComposeProd string
	Go111Module string
	GoProxy     string
}

func (options Options) HasExplicitPublicAddress() bool {
	return options.PublicAddressSource != PublicAddressSourceDefault
}

func Parse(args []string) (Options, error) {
	if err := checkLegacyFlags(args); err != nil {
		return Options{}, err
	}

	dev := false
	nonInteractive := false
	releaseVersion := ""
	useGoProxyCN := false
	publicURLRaw := ""
	publicHostRaw := ""
	publicPortRaw := ""
	imageRegistry := firstNonEmpty(os.Getenv("LUNAFOX_IMAGE_REGISTRY"), DefaultImageRegistry)
	imageNamespace := firstNonEmpty(os.Getenv("LUNAFOX_IMAGE_NAMESPACE"), DefaultImageNamespace)
	agentImageRefsRaw := strings.TrimSpace(os.Getenv("AGENT_IMAGE_REFS"))
	workerImageRefsRaw := strings.TrimSpace(os.Getenv("WORKER_IMAGE_REFS"))
	agentNetwork := firstNonEmpty(os.Getenv("LUNAFOX_AGENT_DOCKER_NETWORK"), DefaultAgentNetwork)
	sharedDataBind := firstNonEmpty(os.Getenv("LUNAFOX_SHARED_DATA_VOLUME_BIND"), DefaultSharedDataBind)
	releaseManifestInput := strings.TrimSpace(os.Getenv("LUNAFOX_RELEASE_MANIFEST"))
	rootDirInput := ""

	fs := flag.NewFlagSet("lunafox-installer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.BoolVar(&dev, "dev", false, "开发模式")
	fs.StringVar(&releaseVersion, "version", strings.TrimSpace(os.Getenv("LUNAFOX_INSTALLER_VERSION")), "发布版本，例如 v1.5.13")
	fs.BoolVar(&useGoProxyCN, "goproxy", false, "使用 goproxy.cn")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "禁用交互向导，缺少必填参数时直接失败")
	fs.StringVar(&publicURLRaw, "public-url", "", "公网访问地址（与 --public-host/--public-port 二选一），例如 https://10.0.0.8:18443")
	fs.StringVar(&publicHostRaw, "public-host", "", "公网主机（仅支持 localhost 或 IPv4，需配合 --public-port），例如 10.0.0.8")
	fs.StringVar(&publicPortRaw, "public-port", "", "公网端口（1-65535，需配合 --public-host）")
	fs.StringVar(&imageRegistry, "image-registry", imageRegistry, "镜像仓库域名，例如 docker.io 或 ghcr.io")
	fs.StringVar(&imageNamespace, "image-namespace", imageNamespace, "镜像命名空间，例如 yyhuni")
	fs.StringVar(&agentImageRefsRaw, "agent-image-refs", agentImageRefsRaw, "Agent 镜像候选列表（逗号分隔，按优先级）")
	fs.StringVar(&workerImageRefsRaw, "worker-image-refs", workerImageRefsRaw, "Worker 镜像候选列表（逗号分隔，按优先级）")
	fs.StringVar(&releaseManifestInput, "release-manifest", releaseManifestInput, "release manifest 文件路径（默认 <root-dir>/release.manifest.yaml）")
	fs.StringVar(&rootDirInput, "root-dir", "", "项目根目录（必填）")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	if fs.NArg() > 0 {
		return Options{}, fmt.Errorf("不支持的位置参数: %s", strings.Join(fs.Args(), " "))
	}

	publicURLProvided := hasFlag(args, "--public-url")
	publicHostProvided := hasFlag(args, "--public-host")
	publicPortProvided := hasFlag(args, "--public-port")

	if publicURLProvided && publicHostProvided {
		return Options{}, fmt.Errorf("--public-url 与 --public-host 不能同时使用")
	}
	if publicURLProvided && publicPortProvided {
		return Options{}, fmt.Errorf("--public-url 与 --public-port 不能同时使用")
	}
	if publicHostProvided && !publicPortProvided {
		return Options{}, fmt.Errorf("--public-host 需要配合 --public-port 使用")
	}
	if publicPortProvided && !publicHostProvided {
		return Options{}, fmt.Errorf("--public-port 需要配合 --public-host 使用")
	}

	mode := ModeProd
	if dev {
		mode = ModeDev
	}

	publicURL := ""
	publicPort := ""
	addressSource := PublicAddressSourceDefault
	var err error

	switch {
	case publicURLProvided:
		publicURL, publicPort, err = NormalizePublicURL(publicURLRaw, "")
		if err != nil {
			return Options{}, err
		}
		addressSource = PublicAddressSourceURL
	case publicHostProvided:
		publicURL, publicPort, err = NormalizePublicHostPort(publicHostRaw, publicPortRaw)
		if err != nil {
			return Options{}, err
		}
		addressSource = PublicAddressSourceHostPort
	}

	if nonInteractive && addressSource == PublicAddressSourceDefault {
		return Options{}, fmt.Errorf("--non-interactive 需要配合 --public-url 或 --public-host/--public-port")
	}

	releaseVersion = strings.TrimSpace(releaseVersion)
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
	releaseManifest := strings.TrimSpace(releaseManifestInput)
	if releaseManifest == "" {
		releaseManifest = filepath.Join(rootDir, "release.manifest.yaml")
	} else if !filepath.IsAbs(releaseManifest) {
		releaseManifest = filepath.Join(rootDir, releaseManifest)
	}

	dockerDir := filepath.Join(rootDir, "docker")
	composeDev := filepath.Join(dockerDir, "docker-compose.dev.yml")
	composeProd := filepath.Join(dockerDir, "docker-compose.yml")
	composeFile := composeProd
	if mode == ModeDev {
		composeFile = composeDev
	}

	return Options{
		Mode:                mode,
		ReleaseVersion:      releaseVersion,
		UseGoProxyCN:        useGoProxyCN,
		PublicURL:           publicURL,
		PublicPort:          publicPort,
		PublicAddressSource: addressSource,
		NonInteractive:      nonInteractive,
		ImageRegistry:       imageRegistry,
		ImageNamespace:      imageNamespace,
		AgentImageRef:       agentImageRef,
		AgentImageRefs:      agentImageRefs,
		WorkerImageRef:      workerImageRef,
		WorkerImageRefs:     workerImageRefs,
		AgentNetwork:        agentNetwork,
		SharedDataBind:      sharedDataBind,
		ReleaseManifest:     releaseManifest,
		RootDir:             rootDir,
		DockerDir:           dockerDir,
		EnvFile:             filepath.Join(dockerDir, ".env"),
		ComposeFile:         composeFile,
		ComposeDev:          composeDev,
		ComposeProd:         composeProd,
		Go111Module:         "on",
		GoProxy:             goProxy,
	}, nil
}

func checkLegacyFlags(args []string) error {
	for _, arg := range args {
		switch {
		case arg == "--listen" || strings.HasPrefix(arg, "--listen="):
			return fmt.Errorf("--listen 已移除，请改用 --public-url 或 --public-host/--public-port")
		case arg == "--web" || strings.HasPrefix(arg, "--web="):
			return fmt.Errorf("--web 已移除：安装器现在仅支持终端交互")
		case arg == "--headless-install" || strings.HasPrefix(arg, "--headless-install="):
			return fmt.Errorf("--headless-install 已移除，请使用 --non-interactive + 显式公网地址")
		}
	}
	return nil
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
