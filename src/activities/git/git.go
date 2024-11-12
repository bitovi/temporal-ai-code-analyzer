package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bitovi.com/code-analyzer/src/activities/s3"
	"bitovi.com/code-analyzer/src/utils"
)

type ArchiveRepositoryInput struct {
	Repository string
	Bucket     string
}
type ArchiveRepositoryOutput struct {
	Keys []string
}

func ArchiveRepository(input ArchiveRepositoryInput) (ArchiveRepositoryOutput, error) {
	temporaryDirectory := filepath.Join(os.TempDir(), utils.CleanRepository(input.Repository))
	if err := os.MkdirAll(temporaryDirectory, os.ModePerm); err != nil {
		return ArchiveRepositoryOutput{}, err
	}
	if err := os.RemoveAll(temporaryDirectory); err != nil {
		return ArchiveRepositoryOutput{}, err
	}

	if utils.ChaosExists("github") {
		return ArchiveRepositoryOutput{}, fmt.Errorf("error cloning repository -- are you sure you want to use GitHub?")
	}
	cmd := exec.Command("git", "clone", "--depth", "1", input.Repository, temporaryDirectory)
	if err := cmd.Run(); err != nil {
		return ArchiveRepositoryOutput{}, err
	}

	var fileList []string
	err := filepath.Walk(temporaryDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if utils.IsHiddenFile(path) || utils.IsConfigFile(path) || utils.IsImageFile(path) {
			return nil
		}
		fileList = append(fileList, path)

		return nil
	})
	if err != nil {
		return ArchiveRepositoryOutput{}, fmt.Errorf("error walking temporary directory: %w", err)
	}

	var keys []string
	for _, filePath := range fileList {
		file, err := os.Open(filePath)
		if err != nil {
			return ArchiveRepositoryOutput{}, fmt.Errorf("error opening %s: %w", filePath, err)
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return ArchiveRepositoryOutput{}, fmt.Errorf("error reading %s: %w", filePath, err)
		}

		if utils.ChaosExists("aws") {
			return ArchiveRepositoryOutput{}, fmt.Errorf("error with S3.Put -- AWS is totally down")
		}
		key := strings.ReplaceAll(filePath, temporaryDirectory+"/", "")
		err = s3.PutObject(
			input.Bucket,
			key,
			data,
		)
		if err != nil {
			return ArchiveRepositoryOutput{}, fmt.Errorf("error putting object in S3 for %s: %w", key, err)
		}
		keys = append(keys, key)
	}

	return ArchiveRepositoryOutput{
		Keys: keys,
	}, nil
}
