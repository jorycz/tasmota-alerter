package utils

import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	"fmt"
	"path/filepath"
)


func ReadFilesWithSuffix(fileFolder string, fileMask string) ([]string, error) {
	var lines []string

	// Read all files with suffix in a specified folder
	err := filepath.Walk(fileFolder, func(path string, info os.FileInfo, err error) error {
        slog.Debug("Reading", "file", info.Name())
		if err != nil {
            slog.Debug("Error when reading file", err)
			return err
		}
		if info.IsDir() {
            slog.Debug("Ignoring folder", err)
			return nil
		}

		if strings.HasSuffix(info.Name(), fileMask) {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Ignore lines starting with # and empty lines
				if !strings.HasPrefix(line, "#") && len(strings.TrimSpace(line)) > 0 {
					lines = append(lines, line)
				} else {
                    slog.Debug("File line ignored", "line", line)
                }
			}
			// Check for errors during scanning
			if err := scanner.Err(); err != nil {
                slog.Error("Error when scanning file", err)
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error reading rule configs %v", err)
	}

	return lines, nil
}
