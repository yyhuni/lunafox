package web

import (
	"bytes"
	_ "embed"
	"html/template"
	"strings"

	"github.com/yyhuni/lunafox/tools/installer/internal/cli"
)

//go:embed static/index.html
var indexHTMLTemplate string

var indexTemplate = template.Must(template.New("index").Parse(indexHTMLTemplate))

func renderIndexHTML(baseOptions cli.Options) ([]byte, error) {
	installMode := strings.TrimSpace(baseOptions.Mode)
	if installMode != cli.ModeProd && installMode != cli.ModeDev {
		installMode = cli.ModeProd
	}

	defaultHost, defaultPort := parseHostPortFromURL(baseOptions.PublicURL)
	if defaultHost == "" {
		defaultHost = "localhost"
	}
	defaultPort = pick(defaultPort, baseOptions.PublicPort, cli.DefaultPublicPort)

	data := indexTemplateData{
		InstallMode:       installMode,
		DefaultPublicHost: defaultHost,
		DefaultPublicPort: defaultPort,
		ShowGoProxy:       installMode == cli.ModeDev,
	}

	var buffer bytes.Buffer
	if err := indexTemplate.Execute(&buffer, data); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
