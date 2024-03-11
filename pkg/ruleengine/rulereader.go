package ruleengine

import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	"fmt"
	"path/filepath"
)


func readRuleConfFiles() ([]string, error) {

	rulesFolder := "rules"
	var lines []string

	// Read all .conf files in rules folder
	err := filepath.Walk(rulesFolder, func(path string, info os.FileInfo, err error) error {
        slog.Debug("DEBUG", "Reading rule file", info.Name())
		if err != nil {
            slog.Debug("Error when reading rule file", err)
			return err
		}
		if info.IsDir() {
            slog.Debug("Ignoring folder", err)
			return nil
		}
		if strings.HasSuffix(info.Name(), ".conf") {
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
                    slog.Debug("RULE line ignored", "line", line)
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
