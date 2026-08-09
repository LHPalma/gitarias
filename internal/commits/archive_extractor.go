package commits

import (
	"os"
	"path/filepath"

	"github.com/LHPalma/gitarias/internal/git"
)

type ArchiveExtractor struct {
	runner git.Runner
}

func NewArchiveExtractor(runner git.Runner) *ArchiveExtractor {
	return &ArchiveExtractor{runner: runner}
}

func (extractor *ArchiveExtractor) Extract(sha string, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}

	archivePath := filepath.Join(filepath.Dir(destination), sha+".tar")

	if _, err := extractor.runner.Run("archive", "--format=tar", "--output="+archivePath, sha); err != nil {
		return err
	}
	defer os.Remove(archivePath)

	return untar(archivePath, destination)
}

func (extractor *ArchiveExtractor) Release(string) {}
