package migrate

import (
	"fmt"
	"io/ioutil"
	"os"
)

// File exists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)

	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

// Get files retrieves all files based on the file path param
func getFiles(path string) ([]os.FileInfo, error) {
	files, err := ioutil.ReadDir(path)

	if err != nil {
		return nil, err
	}

	return files, nil
}

// Get the migration contents, this will read the file of the
// path provided and return the sql in string format.
func getFileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)

	if err != nil {
		return "", fmt.Errorf("cannot get file contents - %w", err)
	}

	return string(contents), nil
}

// Check if the directory exists based on the path argument, if the
// directory ends with a "/", the returning string will be
// stripped.
func processDirectory(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("migration path not found - %v", path)
	}

	lastChar := path[len(path)-1:]
	if lastChar == "/" {
		r := []rune(path)
		path = string(r[:len(r)-1])
	}

	return path, nil
}
