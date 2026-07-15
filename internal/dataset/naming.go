package dataset

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var invalidName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Paths holds file and directory paths for a named dataset.
type Paths struct {
	Name         string
	Dir          string
	JSONL        string
	TrainJSONL   string
	DatasetInfo  string
	RefinedJSONL string
	Report       string
	Manifest     string
}

// SanitizeName normalizes a dataset name for use in paths.
func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = invalidName.ReplaceAllString(name, "-")
	name = strings.Trim(strings.Trim(name, "-"), ".")
	if name == "" {
		return "dataset"
	}
	return name
}

// RandomName generates a unique random dataset directory name.
func RandomName() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "dataset-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return "dataset-" + hex.EncodeToString(b)
}

// ResolveName picks dataset name: explicit --name, then --keywords, then random.
func ResolveName(explicit, keywords string) string {
	if explicit != "" {
		return SanitizeName(explicit)
	}
	if kw := strings.TrimSpace(keywords); kw != "" {
		return SanitizeName(kw)
	}
	return RandomName()
}

// NewPaths builds dataset paths under datasetsDir for the given name.
func NewPaths(datasetsDir, name string) Paths {
	name = SanitizeName(name)
	dir := filepath.Join(datasetsDir, name)
	return Paths{
		Name:         name,
		Dir:          dir,
		JSONL:        filepath.Join(dir, "dataset.jsonl"),
		TrainJSONL:   filepath.Join(dir, "train.jsonl"),
		DatasetInfo:  filepath.Join(dir, "dataset_info.json"),
		RefinedJSONL: filepath.Join(dir, "refined.jsonl"),
		Report:       filepath.Join(dir, "refined-report.json"),
		Manifest:     filepath.Join(dir, "manifest.json"),
	}
}

// EnsureDir creates the dataset directory for the resolved name.
func EnsureDir(datasetsDir, explicitName, keywords string) (Paths, error) {
	paths := NewPaths(datasetsDir, ResolveName(explicitName, keywords))
	if err := os.MkdirAll(paths.Dir, 0o755); err != nil {
		return Paths{}, err
	}
	return paths, nil
}

// OutputPath returns the primary output path for the given format.
func (p Paths) OutputPath(format string) string {
	switch format {
	case "huggingface", "hf":
		return p.Dir
	default:
		return p.JSONL
	}
}
