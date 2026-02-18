package subdomain_discovery

import (
	"embed"

	"github.com/yyhuni/lunafox/worker/internal/activity"
)

//go:generate go run ../../../cmd/const-gen/main.go -input templates.yaml -output constants_generated.go -package subdomain_discovery
//go:generate go run ../../../cmd/schema-gen/main.go -input templates.yaml -output schema_generated.json
//go:generate go run ../../../cmd/schema-gen/main.go -input templates.yaml -output ../../../../server/internal/engineschema/subdomain_discovery.schema.json
//go:generate go run ../../../cmd/doc-gen/main.go -input templates.yaml -output-dir ../../../../docs/config-reference

//go:embed templates.yaml schema_generated.json
var templatesFS embed.FS

// loader is the template loader for subdomain discovery workflow
var loader = activity.NewTemplateLoader(templatesFS, "templates.yaml")

// getTemplate returns the command template for a given tool
func getTemplate(toolName string) (activity.CommandTemplate, error) {
	return loader.Get(toolName)
}
