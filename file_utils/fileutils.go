package fileutils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type File struct {
	FileName    string `json:"fileName"`
	FileContent string `json:"fileContent"`
}

type FileList struct {
	Files []File `json:"files"`
}

func ToKebabCase(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if r == ' ' {
			builder.WriteRune('-')
		} else {
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	return builder.String()
}

func WriteToFile(fileName string, content string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func CreateTxtFilename(directory string, iteration int, name string, suffix string) string {
	return filepath.Join(
		directory,
		ToKebabCase(fmt.Sprintf("%03d %s %s.txt", iteration, name, suffix)),
	)
}

func SaveFilesFromJson(outputDir string, jsonData []byte) error {
	files, err := createFilesFromJson(jsonData)
	if err != nil {
		return err
	}

	for name, content := range files {
		fullPath := filepath.Join(outputDir, name)

		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return err
		}

		err = os.WriteFile(fullPath, []byte(content), 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func createFilesFromJson(jsonData []byte) (map[string]string, error) {
	output := map[string]string{}

	var fileList FileList
	if err := json.Unmarshal(jsonData, &fileList); err != nil {
		return map[string]string{}, err
	}

	for _, file := range fileList.Files {
		output[file.FileName] = file.FileContent
	}

	return output, nil
}
