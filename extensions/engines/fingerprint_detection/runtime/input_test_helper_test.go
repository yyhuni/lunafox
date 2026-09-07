package fingerprintdetectionruntime

import "context"

type testInputPath string

func (path testInputPath) Path(context.Context) (string, error) { return string(path), nil }
