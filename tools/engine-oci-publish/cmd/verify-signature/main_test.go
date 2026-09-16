package main

import "testing"

func TestRejectMutableOrMissingReference(t *testing.T) {
	for _, args := range [][]string{nil, {"docker.io/yyhuni/example:latest"}, {"one", "two"}} {
		if err := run(args); err == nil {
			t.Fatalf("run(%v) accepted an invalid immutable reference", args)
		}
	}
}
