package fileutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

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
