package domain

type EngineCatalogItem struct {
	EngineID             string
	ManifestVersion      string
	Publisher            string
	PackageVersion       string
	ArtifactRef          string
	PackageDigest        string
	EngineAPIMajor       uint32
	SupportedTargetTypes []string
	ExecutionResources   []string
	ConfigSections       []EngineConfigSection
	LocaleResources      map[string]map[string]any
}

type EngineConfigSection struct {
	ID              string
	DefaultEnabled  bool
	RequiredEnabled bool
	Params          []EngineConfigParam
}

type EngineConfigParam struct {
	Key       string
	Type      string
	Default   any
	Minimum   *int
	Maximum   *int
	MinLength *int
	MaxLength *int
	MinItems  *int
	MaxItems  *int
	Pattern   string
	Enum      []string
	Resource  *EngineConfigParamResource
}

type EngineConfigParamResource struct {
	Kind string
}
