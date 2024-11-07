package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bitovi.com/code-analyzer/src/utils"
)

type CloneRepositoryInput struct {
	Repository       string
	StorageDirectory string
}
type CloneRepositoryOutput struct {
	Files []string
}

func CloneRepository(input CloneRepositoryInput) (CloneRepositoryOutput, error) {
	temporaryDirectory := filepath.Join(input.StorageDirectory, utils.CleanRepository(input.Repository))
	if err := os.MkdirAll(temporaryDirectory, os.ModePerm); err != nil {
		return CloneRepositoryOutput{}, err
	}
	if err := os.RemoveAll(temporaryDirectory); err != nil {
		return CloneRepositoryOutput{}, err
	}

	cmd := exec.Command("git", "clone", "--depth", "1", input.Repository, temporaryDirectory)
	if err := cmd.Run(); err != nil {
		return CloneRepositoryOutput{}, err
	}

	var fileList []string
	err := filepath.Walk(temporaryDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if isHidden(path) {
			return nil
		}
		fileList = append(fileList, path)

		return nil
	})
	if err != nil {
		return CloneRepositoryOutput{}, err
	}

	return CloneRepositoryOutput{
		Files: fileList,
	}, nil
}

func isHidden(path string) bool {
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
