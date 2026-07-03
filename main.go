package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/shades-of-prakash/ogc/utils"
	"github.com/spf13/cobra"
)



const (
	Reset      = "\033[0m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
	BoldPurple = "\033[1;35m"
	BoldCyan   = "\033[1;36m"
	Gray       = "\033[37m"
)

var stdinReader = bufio.NewReader(os.Stdin)


var validOdooTags = map[string]bool{
	"FIX":   true,
	"REF":   true,
	"ADD":   true,
	"REM":   true,
	"REV":   true,
	"MOV":   true,
	"REL":   true,
	"IMP":   true,
	"MERGE": true,
	"CLA":   true,
	"I18N":  true,
	"PERF":  true,
	"CLN":   true,
	"LINT":  true,
}

func main() {
	var config commitConfig

	rootCmd := &cobra.Command{
		Use:   "ogc [flags] <path>",
		Short: "Generate commit messages from clipboard or editor input",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("exactly one path argument is required")
			}

			if _, err := os.Stat(args[0]); os.IsNotExist(err) {
				return fmt.Errorf("path '%s' does not exist", args[0])
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			config.path = args[0]

			// Validate Task ID (required, cannot be empty, cannot start with '-' due to parser swallowing)
			taskID := strings.TrimSpace(config.taskID)
			if taskID == "" {
				fmt.Printf("%sError: task ID (-i / --task) is required. Even AI cannot guess which task you are working on!%s\n", BoldRed, Reset)
				os.Exit(1)
			}
			if strings.HasPrefix(taskID, "-") {
				fmt.Printf("%sError: invalid task ID '%s'. Task ID cannot start with '-' (did you forget to provide a value for -i / --task?)%s\n", BoldRed, taskID, Reset)
				os.Exit(1)
			}

			// Validate Module Name (required, cannot be empty, cannot start with '-' due to parser swallowing)
			moduleName := strings.TrimSpace(config.moduleName)
			if moduleName == "" {
				fmt.Printf("%sError: module name (-m / --module) is required. Please tell us which Odoo module you are modifying!%s\n", BoldRed, Reset)
				os.Exit(1)
			}
			if strings.HasPrefix(moduleName, "-") {
				fmt.Printf("%sError: invalid module name '%s'. Module name cannot start with '-' (did you forget to provide a value for -m / --module?)%s\n", BoldRed, moduleName, Reset)
				os.Exit(1)
			}

			if err := EnsureConfigFile(); err != nil {
				fmt.Printf("%sConfig Initialization Error: %v%s\n", BoldRed, err, Reset)
				os.Exit(1)
			}

			cfg, err := LoadConfig()
			if err != nil {
				fmt.Printf("%sConfig Error: %v%s\n", BoldRed, err, Reset)
				os.Exit(1)
			}

			if err := IsGitRepo(config.path); err != nil {
				fmt.Printf("%sError: '%s' is not a git repository. Are you sure you are in the right folder?%s\n", BoldRed, config.path, Reset)
				os.Exit(1)
			}

			diff, err := GetGitDiff(config.path)
			if err != nil {
				fmt.Printf("%sError retrieving git diff: %v%s\n", BoldRed, err, Reset)
				os.Exit(1)
			}

			var taskInfo string

			if config.useClipboard {
				content, err := utils.GetClipboard()
				if err != nil {
					fmt.Printf("%sError retrieving clipboard content: %v%s\n", BoldRed, err, Reset)
					os.Exit(1)
				}

				fmt.Printf("%s📋 Sniffed some task details from your clipboard:%s\n", BoldCyan, Reset)
				fmt.Printf("%s--------------------------------------------------------------------------------%s\n", Gray, Reset)
				fmt.Println(strings.TrimSpace(content))
				fmt.Printf("%s--------------------------------------------------------------------------------%s\n", Gray, Reset)
				fmt.Printf("%s🤔 Do you want to generate a commit message using this clipboard content? (y/N): %s", BoldYellow, Reset)

				input, err := stdinReader.ReadString('\n')
				if err != nil {
					fmt.Printf("%sError reading input: %v%s\n", BoldRed, err, Reset)
					os.Exit(1)
				}

				input = strings.TrimSpace(strings.ToLower(input))
				if input != "y" && input != "yes" {
					fmt.Printf("%s🚫 Mission aborted! Gemini went back to sleep.%s\n", BoldRed, Reset)
					os.Exit(0)
				}

				taskInfo = content
			}

			if config.useEditor {
				content, err := getEditorContent()
				if err != nil {
					fmt.Printf("%sError reading editor content: %v%s\n", BoldRed, err, Reset)
					os.Exit(1)
				}
				taskInfo = content
			}

			ctx := context.Background()

			gemini, err := NewGemini(ctx, cfg)
			if err != nil {
				fmt.Printf("%sGemini Initialization Error: %v%s\n", BoldRed, err, Reset)
				os.Exit(1)
			}

			var selectedTag string
			if config.targetTag != "" {
				upperTag := strings.ToUpper(config.targetTag)
				if validOdooTags[upperTag] {
					selectedTag = upperTag
				} else {
					fmt.Printf("%s⚠️ Warning: '%s' is not a valid Odoo git tag. Defaulting to AUTO detection.%s\n", BoldYellow, config.targetTag, Reset)
					fmt.Printf("%sValid tags are: FIX, REF, ADD, REM, REV, MOV, REL, IMP, MERGE, CLA, I18N, PERF, CLN, LINT%s\n\n", BoldCyan, Reset)
				}
			}

			prompt := BuildPrompt(
				config.moduleName,
				diff,
				taskInfo,
				selectedTag,
			)

			commitMessage, err := gemini.Generate(ctx, prompt)

			if err != nil {
				HandleGeminiError(err)
			}

			commitMessage = WrapText(commitMessage, cfg.LineLength)

			if config.taskID != "" {
				cleanID := strings.TrimPrefix(config.taskID, "#")
				commitMessage = commitMessage + "\n\n#" + cleanID
			}

			fmt.Println()
			fmt.Printf("%s✨ BOOM! Your Odoo-approved Git masterpiece is ready:%s\n", BoldGreen, Reset)
			fmt.Printf("%s================================================================================%s\n", Gray, Reset)
			fmt.Println(commitMessage)
			fmt.Printf("%s================================================================================%s\n", Gray, Reset)

			// Prompt to copy to clipboard in a funny way
			fmt.Println()
			fmt.Printf("%s📋 Teleport this masterpiece directly to your clipboard? (y/N): %s", BoldYellow, Reset)
			input, err := stdinReader.ReadString('\n')
			if err == nil {
				input = strings.TrimSpace(strings.ToLower(input))
				if input == "y" || input == "yes" {
					err = utils.SetClipboard(commitMessage)
					if err != nil {
						fmt.Printf("%s❌ Ah snap! Clipboard teleportation failed: %v%s\n", BoldRed, err, Reset)
					} else {
						fmt.Printf("%s🚀 Copied! Go flex it on git!%s\n", BoldGreen, Reset)
					}
				} else {
					fmt.Printf("%s👋 Fine, keeping it local. Happy committing!%s\n", BoldCyan, Reset)
				}
			}
		},
	}

	rootCmd.Flags().BoolVarP(
		&config.useClipboard,
		"clipboard",
		"c",
		false,
		"Use content from system clipboard",
	)

	rootCmd.Flags().BoolVarP(
		&config.useEditor,
		"editor",
		"e",
		false,
		"Use editor mode",
	)

	rootCmd.Flags().StringVarP(
		&config.moduleName,
		"module",
		"m",
		"",
		"Module name",
	)

	rootCmd.Flags().StringVarP(
		&config.targetTag,
		"tag",
		"t",
		"",
		"Force a specific Odoo git tag (case-insensitive). Valid options: FIX, REF, ADD, REM, etc.",
	)

	rootCmd.Flags().StringVarP(
		&config.taskID,
		"task",
		"i",
		"",
		"Task ID or Issue reference to append at the end of the commit message",
	)

	rootCmd.MarkFlagsOneRequired("clipboard", "editor")
	rootCmd.MarkFlagsMutuallyExclusive("clipboard", "editor")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func getEditorContent() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default fallback editor
	}

	tmpFile, err := os.CreateTemp("", "ogc-task-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Editor '%s' failed to start. Please type or paste your task description below directly, then press Ctrl+D when finished:\n", editor)
		fmt.Println("--------------------------------------------------------------------------------")
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WrapText(text string, limit int) string {
	mergedText := CleanAndMergeLines(text)
	lines := strings.Split(mergedText, "\n")
	var result []string

	for i, line := range lines {
		// Do not wrap the first line (header)
		if i == 0 {
			result = append(result, line)
			continue
		}

		if len(line) <= limit {
			result = append(result, line)
			continue
		}

		// Detect if list item and preserve prefix indentation
		prefix := ""
		trimmed := strings.TrimSpace(line)
		isList := false
		if strings.HasPrefix(trimmed, "- ") {
			idx := strings.Index(line, "- ")
			prefix = line[:idx+2]
			isList = true
		} else if len(trimmed) > 2 && trimmed[1] == '.' && trimmed[2] == ' ' {
			idx := strings.Index(line, ". ")
			prefix = line[:idx+2]
			isList = true
		}

		// Split line into words
		words := strings.Fields(line)
		if len(words) == 0 {
			result = append(result, line)
			continue
		}

		// If it's a list item, remove the marker word from words list
		if isList && len(words) > 1 {
			words = words[1:]
		}

		var currentLine string
		firstWord := true

		for _, word := range words {
			if firstWord {
				currentLine = prefix + word
				firstWord = false
				continue
			}

			if len(currentLine)+1+len(word) > limit {
				result = append(result, currentLine)
				if prefix != "" {
					currentLine = strings.Repeat(" ", len(prefix)) + word
				} else {
					currentLine = word
				}
			} else {
				currentLine += " " + word
			}
		}
		if currentLine != "" {
			result = append(result, currentLine)
		}
	}

	return strings.Join(result, "\n")
}

func CleanAndMergeLines(text string) string {
	lines := strings.Split(text, "\n")
	var mergedLines []string
	var currentParagraph strings.Builder

	for i, line := range lines {
		// Keep the first line (header) untouched
		if i == 0 {
			mergedLines = append(mergedLines, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		// If line is empty, it separates paragraphs
		if trimmed == "" {
			if currentParagraph.Len() > 0 {
				mergedLines = append(mergedLines, currentParagraph.String())
				currentParagraph.Reset()
			}
			mergedLines = append(mergedLines, "")
			continue
		}

		// Detect if this is a new section/bullet point/header
		isNewSection := strings.HasPrefix(trimmed, "- ") ||
			(len(trimmed) > 2 && trimmed[1] == '.' && trimmed[2] == ' ') ||
			strings.HasSuffix(trimmed, ":")

		if isNewSection {
			if currentParagraph.Len() > 0 {
				mergedLines = append(mergedLines, currentParagraph.String())
				currentParagraph.Reset()
			}
			currentParagraph.WriteString(line)
		} else {
			// Merge continuation line with current paragraph
			if currentParagraph.Len() > 0 {
				currentParagraph.WriteString(" " + trimmed)
			} else {
				currentParagraph.WriteString(line)
			}
		}
	}

	if currentParagraph.Len() > 0 {
		mergedLines = append(mergedLines, currentParagraph.String())
	}

	return strings.Join(mergedLines, "\n")
}
