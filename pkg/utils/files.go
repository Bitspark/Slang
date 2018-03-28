package utils

import (
	"strings"
	"os"
	"fmt"
)

func FileWithFileEnding(filename string, fileEndings []string) (string, error) {
	for _, fileEnding := range fileEndings {
		if strings.HasSuffix(filename, fileEnding) {
			if _, err := os.Stat(filename); err == nil {
				return filename, nil
			} else {
				return "", err
			}
		}
	}

	for _, fileEnding := range fileEndings {
		filenameWithEnding := filename + fileEnding
		if _, err := os.Stat(filenameWithEnding); err == nil {
			return filenameWithEnding, nil
		}
	}

	return "", fmt.Errorf("%s: no appropriate file found", filename)
}