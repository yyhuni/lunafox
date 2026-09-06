package installedengines

import (
	"errors"
	"testing"
)

type installedEngineQueryErrorStub struct{ getErr error }

func (stub installedEngineQueryErrorStub) ListInstalledEnginePackages() ([]ResolvedInstalledEnginePackage, error) {
	return nil, nil
}

func (stub installedEngineQueryErrorStub) GetInstalledEnginePackage(string) (ResolvedInstalledEnginePackage, error) {
	return ResolvedInstalledEnginePackage{}, stub.getErr
}

func TestGetInstalledEnginePackageMapsOnlyDomainNotFound(t *testing.T) {
	tests := []struct {
		name         string
		queryErr     error
		wantNotFound bool
	}{
		{name: "repository not found", queryErr: ErrEnginePackageNotFound, wantNotFound: true},
		{name: "cache corrupt", queryErr: errors.New("cache corrupt")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureInstalledEngineQueryForErrorTest(t, installedEngineQueryErrorStub{getErr: test.queryErr})
			_, err := GetInstalledEnginePackage("engine.lunafox.port_scan")
			if errors.Is(err, ErrEnginePackageNotFound) != test.wantNotFound {
				t.Fatalf("GetInstalledEnginePackage() error = %v, wantNotFound=%v", err, test.wantNotFound)
			}
			if !test.wantNotFound && !errors.Is(err, test.queryErr) {
				t.Fatalf("technical error was not preserved: %v", err)
			}
		})
	}
}

func configureInstalledEngineQueryForErrorTest(t *testing.T, configured Query) {
	t.Helper()
	installedEngineQueryMu.Lock()
	previous := installedEngineQuery
	installedEngineQuery = configured
	installedEngineQueryMu.Unlock()
	t.Cleanup(func() {
		installedEngineQueryMu.Lock()
		installedEngineQuery = previous
		installedEngineQueryMu.Unlock()
	})
}
