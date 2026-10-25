package media

import (
	"os"
	"path/filepath"
)

// UsageByChat sums media bytes per chat dir under dataDir/media
// (WL-3.3 storage page).
func UsageByChat(dataDir string) (map[string]int64, error) {
	out := map[string]int64{}
	root := filepath.Join(dataDir, "media")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var sum int64
		_ = filepath.Walk(filepath.Join(root, e.Name()), func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				sum += info.Size()
			}
			return nil
		})
		out[e.Name()] = sum
	}
	return out, nil
}

// ClearCache removes the thumbnail/link-preview cache dir contents
// (WL-3.3); media itself is untouched.
func ClearCache(cacheDir string) error {
	entries, err := os.ReadDir(cacheDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(cacheDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
