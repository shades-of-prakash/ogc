package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func IsGitRepo(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	cmd := exec.Command(
		"git",
		"-C",
		absPath,
		"rev-parse",
		"--is-inside-work-tree",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("'%s' is not a git repository", absPath)
	}

	if string(output) != "true\n" {
		return fmt.Errorf("'%s' is not a git repository", absPath)
	}

	return nil
}

func GetGitDiff(path string) (string, error) {
	cmd := exec.Command(
		"git",
		"-C",
		path,
		"diff",
		"--cached",
	)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w\n%s", err, output)
	}

	if len(output) == 0 {
		return "", fmt.Errorf("no staged changes found. Please run 'git add' first")
	}

	return string(output), nil
}
