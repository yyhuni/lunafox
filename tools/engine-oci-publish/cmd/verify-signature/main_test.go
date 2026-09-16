package main

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/contracts/ocisignature"
	"testing"
)

func TestRejectMutableOrMissingReference(t *testing.T) {
	for _, args := range [][]string{nil, {"docker.io/yyhuni/example:latest"}, {"one", "two"}} {
		if err := run(args); err == nil {
			t.Fatalf("run(%v) accepted an invalid immutable reference", args)
		}
	}
}

func TestPublicationWaitsOnlyForMissingReferrers(t *testing.T) {
	cases := []struct {
		name      string
		failure   error
		succeeds  bool
		wantCalls int
	}{
		{"eventual visibility", &ocisignature.MissingBundleError{Reference: "test"}, true, 3},
		{"bounded absence", &ocisignature.MissingBundleError{Reference: "test"}, false, 30},
		{"integrity failure", errors.New("signature invalid"), false, 1},
		{"transport failure", ocisignature.NewTransportError(errors.New("registry denied")), false, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := verifyPublishedSignature(context.Background(), func() error {
				calls++
				if tc.succeeds && calls == 3 {
					return nil
				}
				return tc.failure
			}, func(context.Context) error { return nil })
			if (err == nil) != tc.succeeds || calls != tc.wantCalls {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		})
	}
}
func TestPublicationWaitRespectsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := verifyPublishedSignature(ctx, func() error { t.Fatal("must not verify after cancellation"); return nil }, func(context.Context) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
