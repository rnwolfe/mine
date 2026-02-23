package agents

import (
	"io"
	"os"
	"path/filepath"
)

// fileExists reports whether path exists and is a regular file (not a directory or symlink).
func fileExists(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

// dirNonEmpty reports whether path is a directory with at least one entry.
func dirNonEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}

// upsertManifestLink adds or updates a link entry in the manifest.
func upsertManifestLink(m *Manifest, source, target, agentName, mode string) {
	for i, l := range m.Links {
		if l.Target == target {
			m.Links[i] = LinkEntry{
				Source: source,
				Target: target,
				Agent:  agentName,
				Mode:   mode,
			}
			return
		}
	}
	m.Links = append(m.Links, LinkEntry{
		Source: source,
		Target: target,
		Agent:  agentName,
		Mode:   mode,
	})
}

// copyFile copies src to dst, preserving file permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir recursively copies the src directory to dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return copyFileMode(path, target, info.Mode())
	})
}

// copyFileMode copies src to dst with the given permissions.
func copyFileMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
