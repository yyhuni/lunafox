package docker

import "testing"

func TestJoinPrefix(t *testing.T) {
	joined := joinPrefix([]string{"sudo"}, "docker", "info")
	if len(joined) != 3 {
		t.Fatalf("unexpected length: %d", len(joined))
	}
	if joined[0] != "sudo" || joined[1] != "docker" || joined[2] != "info" {
		t.Fatalf("unexpected joined args: %#v", joined)
	}
}
