package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"bitovi.com/code-analyzer/src/utils"
)

type CloneRepositoryOutput struct {
	Files           []string
	TotalFiles      int
	ZipFileLocation string
}

func CloneRepository(repository string) (CloneRepositoryOutput, error) {
	temporaryDirectory := filepath.Join(os.TempDir(), utils.CleanRepository(repository))
	if err := os.MkdirAll(temporaryDirectory, os.ModePerm); err != nil {
		return CloneRepositoryOutput{}, err
	}
	if err := os.RemoveAll(temporaryDirectory); err != nil {
		return CloneRepositoryOutput{}, err
	}

	// Clone the git repository
	cmd := exec.Command("git", "clone", "--depth", "1", repository, temporaryDirectory)
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
		fileList = append(fileList, path)

		return nil
	})
	if err != nil {
		return CloneRepositoryOutput{}, err
	}

	if err := os.RemoveAll(temporaryDirectory); err != nil {
		fmt.Printf("error cleaning up temporary directory: %s\n", err)
	}

	return CloneRepositoryOutput{
		Files:           fileList,
		TotalFiles:      len(fileList),
		ZipFileLocation: "",
	}, nil
}

// func CollectDocuments(ctx context.Context, input CollectDocumentsInput) (CollectDocumentsOutput, error) {
// 	temporaryDirectory := input.WorkflowID
// 	if err := os.MkdirAll(temporaryDirectory, os.ModePerm); err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}

// 	parts := strings.Split(input.GitRepoURL, "/")
// 	organization := parts[3]
// 	repository := strings.TrimSuffix(parts[4], ".git")
// 	repoPath := fmt.Sprintf("%s/%s", organization, repository)

// 	temporaryGitHubDirectory := filepath.Join(temporaryDirectory, repoPath)
// 	if err := os.RemoveAll(temporaryGitHubDirectory); err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}

// 	// Clone the git repository
// 	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", input.GitRepoBranch, fmt.Sprintf("https://github.com/%s.git", repoPath), temporaryGitHubDirectory)
// 	if err := cmd.Run(); err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}

// 	var filteredFileList []string
// 	err := filepath.Walk(temporaryGitHubDirectory, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		if info.IsDir() {
// 			return nil
// 		}

// 		fileExtension := strings.TrimPrefix(filepath.Ext(info.Name()), ".")

// 		if slices.Contains(input.FileExtensions, fileExtension) {
// 			filteredFileList = append(filteredFileList, path)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}

// 	//Create zip
// 	zipFileName := "files.zip"
// 	zipFileLocation := filepath.Join(temporaryDirectory, zipFileName)
// 	zipFile, err := os.Create(zipFileLocation)
// 	if err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}
// 	defer zipFile.Close()

// 	archive := zip.NewWriter(zipFile)
// 	defer archive.Close()

// 	// Add files to the zip
// 	for _, filePath := range filteredFileList {
// 		sourceFile, err := os.Open(filePath)
// 		if err != nil {
// 			return CollectDocumentsOutput{}, err
// 		}
// 		defer sourceFile.Close()

// 		fileName := filepath.Base(filePath)
// 		writer, err := archive.Create(fileName)
// 		if err != nil {
// 			return CollectDocumentsOutput{}, err
// 		}

// 		_, err = io.Copy(writer, sourceFile)
// 		if err != nil {
// 			return CollectDocumentsOutput{}, err
// 		}
// 	}
// 	archive.Close()

// 	fileContent, err := os.ReadFile(zipFileLocation)
// 	if err != nil {
// 		return CollectDocumentsOutput{}, err
// 	}

// 	putS3Object(ctx, PutS3ObjectInput{Body: fileContent, Bucket: input.S3Bucket, Key: zipFileName})

// 	return CollectDocumentsOutput{ZipFileName: zipFileName}, nil
// }
