package collector

// FileRef is a single file discovered in a scan target.
// Paths in Path are repo-relative with forward slashes on every platform.
type FileRef struct {
	// Path is relative to the scan root, always forward slashes.
	Path string
	// AbsPath is the absolute filesystem path used to read contents.
	AbsPath string
	// Size is the file size in bytes at discovery time.
	Size int64
}

// Target is a local directory to scan (after Resolve has cloned remotes if needed).
type Target struct {
	// Root is the absolute path of the repository or directory root.
	Root string
}
