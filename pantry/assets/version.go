package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
)

// ContentHash computes a 10-character hex SHA-256 fingerprint of the
// concatenated content of the named files inside fsys.
// Files that cannot be read are silently skipped.
func ContentHash(fsys fs.FS, paths ...string) string {
	h := sha256.New()
	for _, name := range paths {
		if data, err := fs.ReadFile(fsys, name); err == nil {
			h.Write(data)
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:10]
}
