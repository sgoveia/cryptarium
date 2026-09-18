package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Resolved is a scan root ready for Walk, plus optional cleanup for clones.
type Resolved struct {
	// Root is the absolute filesystem path to walk.
	Root string
	// Display is the original target (path or URL) for report metadata.
	Display string
	// Cleanup removes a temporary clone directory. No-op for local paths.
	Cleanup func()
}

// IsRemoteURL reports whether target looks like an HTTP(S) or file:// git remote.
// file:// is supported so tests (and local bare repos) can clone without the network.
func IsRemoteURL(target string) bool {
	return strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "file://")
}

// IsSSHURL reports whether target looks like an SSH git remote (unsupported).
func IsSSHURL(target string) bool {
	return strings.HasPrefix(target, "git@") || strings.HasPrefix(target, "ssh://")
}

// Resolve turns a local path or public HTTPS git URL into a walkable root.
// For remote URLs it shallow-clones into a temp directory; callers must invoke
// Cleanup when finished (defer resolved.Cleanup()).
func Resolve(ctx context.Context, target string) (Resolved, error) {
	if target == "" {
		return Resolved{}, fmt.Errorf("resolve: target is required")
	}
	if IsSSHURL(target) {
		return Resolved{}, fmt.Errorf("resolve: SSH git URLs are not supported; use an HTTPS URL for public repositories")
	}
	if IsRemoteURL(target) {
		return cloneRemote(ctx, target)
	}

	root, err := filepath.Abs(target)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve path %s: %w", target, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Resolved{}, fmt.Errorf("resolve path %s: %w", target, err)
	}
	if !info.IsDir() {
		return Resolved{}, fmt.Errorf("resolve path %s: not a directory", target)
	}
	return Resolved{
		Root:    root,
		Display: target,
		Cleanup: func() {},
	}, nil
}

func cloneRemote(ctx context.Context, url string) (Resolved, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return Resolved{}, fmt.Errorf("clone %s: git is required on PATH to scan remote URLs", url)
	}

	dir, err := os.MkdirTemp("", "cryptarium-clone-*")
	if err != nil {
		return Resolved{}, fmt.Errorf("clone %s: create temp dir: %w", url, err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	args := []string{"clone", "--depth", "1", "--single-branch", "--", url, dir}
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // G204: url is an explicit user scan target; binary is fixed "git"; "--" precedes url.
	var stderr bytes.Buffer
	cmd.Stdout = ioDiscard{}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		cleanup()
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Resolved{}, fmt.Errorf("clone %s: %w", url, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return Resolved{}, fmt.Errorf("clone %s: %s", url, truncate(msg, 512))
	}

	return Resolved{
		Root:    dir,
		Display: url,
		Cleanup: cleanup,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ioDiscard silences git clone stdout.
type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
