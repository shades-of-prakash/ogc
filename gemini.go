package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/genai"
)

//go:embed prompts/guidelines.txt prompts/examples.txt
var promptFS embed.FS

type Gemini struct {
	Client *genai.Client
	Model  string
}

func NewGemini(ctx context.Context, cfg *Config) (*Gemini, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Gemini{
		Client: client,
		Model:  cfg.Model,
	}, nil
}

func (g *Gemini) Generate(ctx context.Context, prompt string) (string, error) {
	stopSpinner := StartSpinner("Whispering to Gemini... (writing clean commit messages is harder than code)...")

	config := &genai.GenerateContentConfig{
		SafetySettings: []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
		},
	}

	resp, err := g.Client.Models.GenerateContent(
		ctx,
		g.Model,
		genai.Text(prompt),
		config,
	)

	close(stopSpinner)
	// Add a slight delay to ensure terminal printing clears properly and moves cursor to a clean state
	fmt.Print("\r\033[K")

	if err != nil {
		return "", err
	}

	text := resp.Text()

	// Strip out <reasoning> tag if present in the LLM response
	if idx := strings.Index(text, "</reasoning>"); idx != -1 {
		text = text[idx+len("</reasoning>"):]
	}

	return strings.TrimSpace(text), nil
}

func StartSpinner(message string) chan struct{} {
	stop := make(chan struct{})
	go func() {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				fmt.Printf("\r%s%s %s%s", BoldPurple, spinChars[i], message, Reset)
				i = (i + 1) % len(spinChars)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return stop
}
func LoadPromptFile(path string) string {
	// 1. Try local filesystem directly
	data, err := os.ReadFile(path)
	if err == nil {
		return string(data)
	}

	// 2. Try with prompts/ prefix in local filesystem
	if !strings.HasPrefix(path, "prompts/") {
		data, err = os.ReadFile("prompts/" + path)
		if err == nil {
			return string(data)
		}
	}

	// 3. Fallback to embedded filesystem
	embedPath := path
	if !strings.HasPrefix(embedPath, "prompts/") {
		embedPath = "prompts/" + embedPath
	}
	data, err = promptFS.ReadFile(embedPath)
	if err == nil {
		return string(data)
	}

	return ""
}

func BuildPrompt(moduleName, diff, taskInfo, targetTag string) string {
	guidelines := LoadPromptFile("guidelines.txt")
	examples := LoadPromptFile("examples.txt")

	tagRule := ""
	if targetTag != "" {
		tagRule = fmt.Sprintf("\nCRITICAL REQUIREMENT:\nYou MUST select [%s] as the tag. Do not auto-detect. Follow the [%s] template strictly.\n", targetTag, targetTag)
	}

	return fmt.Sprintf(`
You are an expert software engineer and commit message reviewer.
Generate a git commit message following the provided guidelines.
%s
GUIDELINES
==========
%s

EXAMPLES
========
The examples below are authoritative for STRUCTURE and FORMATTING.
If GUIDELINES and EXAMPLES ever seem to conflict on structure, follow EXAMPLES.

%s

INPUT
=====
Module:
%s

Task Information:
%s

Git Diff:
%s

IMPORTANT
=========
The task information is the authoritative source for understanding:
- Why the change was made.
- The business or functional problem.
- Reproduction steps.
- Root cause.
- Expected behavior.

Use the git diff to verify and understand the implementation.
Prioritize the task information when writing the commit message.
Do not simply summarize the diff.

STRICT RULES & TAG IDENTIFICATION
=================================
1. Decision Tree for Tag Selection (Evaluate step-by-step in your reasoning):
   - Step 1 (MERGE check): Is the commit a merge commit, forward-port, or main commit for a feature involving several separated commits?
             -> If YES, select [MERGE].
   - Step 2 (REV check): Does this commit revert a previous commit?
             -> If YES, select [REV].
   - Step 3 (CLA check): Is this commit signing the Contributor License Agreement?
             -> If YES, select [CLA].
   - Step 4 (REL check): Is this commit specifically for a new major/minor release or stable version bump?
             -> If YES, select [REL].
   - Step 5 (FIX check): Does the change correct a bug, traceback, error, constraint violation, or incorrect/broken behavior?
             -> If YES, select [FIX].
   - Step 6 (I18N check): Are the modifications strictly to translation files (e.g. .po, .pot, or translation edits)?
             -> If YES, select [I18N].
   - Step 7 (MOV check): Are files or code blocks moved from one location to another (without changing content)?
             -> If YES, select [MOV].
   - Step 8 (REM check): Does the change remove resources, dead code, views, fields, or obsolete modules?
             -> If YES, select [REM].
   - Step 9 (ADD check): Does the change introduce a brand-new Odoo model (a new Python class inheriting from 'models.Model') or a brand-new module?
             -> If YES, select [ADD].
   - Step 10 (REF/CLN/PERF/LINT check): Does the change refactor code [REF], clean up styling/imports [CLN], optimize speed/memory [PERF], or fix linter/compliance formatting [LINT] without altering functional behavior?
             -> If YES, select the most specific tag of [REF], [CLN], [PERF], or [LINT].
   - Step 11 (Fallback - IMP check): If none of the above apply, and the change incrementally improves, enhances, or adds features/views/fields/buttons to an existing model/view.
             -> Select [IMP].

2. Structure Mapping:
   You MUST match the commit message format to the selected tag's template strictly:
   - For [FIX]: Use '[FIX] Template' (Steps to Reproduce, Issue, Cause, Fix). Deduce/write steps to reproduce if not explicitly provided.
   - For [IMP]: Use '[IMP] Template' (Prior to this commit, Post this commit, Why This Approach, Challenges Faced).
   - For [ADD]: Use '[ADD] Template' (Purpose of this commit, Implementation Details, Key Configuration).
   - For [REF]: Use '[REF] Template' (Prior to this commit, Post this commit, Why This Approach).
   - For [REM]: Use '[REM] Template' (Reason for Removal, Resources Removed, Impact & Cleanup).
   - For [REV]: Use '[REV] Template' (Commit to Revert, Reason for Reversion, Restored State).
   - For [MOV]: Use '[MOV] Template' (Source & Destination, Reason for Move, Reference Updates).
   - For [REL]: Use '[REL] Template' (Version Details, Release Notes & Main Changes).
   - For [MERGE]: Use '[MERGE] Template' (Source & Target Branches, Merged Changes & Commits, Conflict Resolution).
   - For [CLA]: Use '[CLA] Template' (Signed By, Declaration).
   - For [I18N]: Use '[I18N] Template' (Languages Updated, Changes Summary).
   - For [PERF]: Use '[PERF] Template' (Identified Bottleneck, Optimization Strategy, Performance Measurements).
   - For [CLN]: Use '[CLN] Template' (Target Code/Files, Cleanup Actions, Readability Benefits).
   - For [LINT]: Use '[LINT] Template' (Linter / Tool, Violations Fixed).

REQUIRED OUTPUT FORMAT
======================
Your response MUST be divided into two parts:
1. Inner Reasoning: Write your step-by-step tag classification reasoning inside <reasoning>...</reasoning> tags. Explain which steps in the decision tree were evaluated and why you chose the selected tag.
2. Commit Message: Immediately after the closing </reasoning> tag, write ONLY the final, raw git commit message. Do NOT wrap the commit message in markdown code blocks.

Example:
<reasoning>
1. Task asks to fix a traceback error when loading tag demo data.
2. Evaluated steps 1-4: No merge, revert, CLA, or release.
3. Evaluated step 5 (FIX check): Yes, it is a bug fix. So I select [FIX].
4. Template selected: [FIX] template.
</reasoning>
[FIX] estate: fix tag constraint violation

Steps to Reproduce:
1. ...
`,
		tagRule,
		guidelines,
		examples,
		moduleName,
		taskInfo,
		diff,
	)
}


func HandleGeminiError(err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "API_KEY_INVALID"),
		strings.Contains(msg, "API key not valid"):

		fmt.Printf("%sError: Gemini authentication failed.%s\n", BoldRed, Reset)
		fmt.Println()
		fmt.Println("The API key in '~/.config/ogc/config.toml' is invalid, expired, or incorrectly copied.")
		fmt.Println()
		fmt.Println("Update your configuration:")
		fmt.Printf("  %sapi_key = \"AIzaSy...\"%s\n", BoldCyan, Reset)
		fmt.Printf("  %smodel   = \"gemini-2.5-pro\"%s\n", BoldCyan, Reset)
		fmt.Println()
		fmt.Println("Get a free API key from:")
		fmt.Printf("  %shttps://aistudio.google.com/app/apikey%s\n", BoldBlue, Reset)

	case strings.Contains(strings.ToLower(msg), "quota"):
		fmt.Printf("%sError: Gemini API quota exceeded.%s\n", BoldRed, Reset)
		fmt.Println()
		fmt.Println("Your API key is valid, but the usage limit has been reached.")
		fmt.Println("Check your quota and billing settings in Google AI Studio.")

	default:
		fmt.Printf("%sGemini Error: %v%s\n", BoldRed, err, Reset)
	}

	os.Exit(1)
}
