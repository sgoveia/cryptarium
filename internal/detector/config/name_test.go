package config_test

import (
	"testing"

	"github.com/sgoveia/cryptarium/internal/detector/config"
)

// Re-export suite parsers via Detect fixtures; unit-test through exported Detect
// is enough for integration. This file covers Handles-adjacent naming only when
// needed — suite logic is exercised by nginx/sshd/jwt tests.
func TestDetectorName(t *testing.T) {
	if got := config.New().Name(); got != "config" {
		t.Fatalf("Name=%q", got)
	}
}
