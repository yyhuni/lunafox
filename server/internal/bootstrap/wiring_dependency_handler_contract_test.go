package bootstrap

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestBuildDependenciesWiresEveryRouteHandler(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve bootstrap test file path")
	}

	dir := filepath.Dir(file)
	wiringSource, err := os.ReadFile(filepath.Join(dir, "wiring.go"))
	if err != nil {
		t.Fatalf("read bootstrap wiring source: %v", err)
	}
	routesSource, err := os.ReadFile(filepath.Join(dir, "routes.go"))
	if err != nil {
		t.Fatalf("read bootstrap routes source: %v", err)
	}

	dependenciesStart := strings.Index(string(wiringSource), "return &deps{")
	if dependenciesStart < 0 {
		t.Fatal("buildDependencies must return root dependencies")
	}
	dependencies := string(wiringSource[dependenciesStart:])
	routeHandlerPattern := regexp.MustCompile(`d\.([A-Za-z0-9_]+Handler)`)
	dependencyAssignmentPattern := func(handler string) *regexp.Regexp {
		return regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(handler) + `:\s+`)
	}

	for _, match := range routeHandlerPattern.FindAllStringSubmatch(string(routesSource), -1) {
		handler := match[1]
		if !dependencyAssignmentPattern(handler).MatchString(dependencies) {
			t.Fatalf("buildDependencies must assign route handler %q", handler)
		}
	}
}
