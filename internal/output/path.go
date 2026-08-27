package output

import "path/filepath"

// relPath renders p relative to root, falling back to p itself when the
// relative form cannot be computed (e.g. different drives or a p outside
// any common ancestor). Shared by every renderer so path presentation
// stays consistent across text, JSON, and history output.
func relPath(root, p string) string {
	if rel, err := filepath.Rel(root, p); err == nil {
		return rel
	}
	return p
}
