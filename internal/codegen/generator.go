package codegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GeneratedFile is a file produced by the generator
type GeneratedFile struct {
	Path    string
	Content string
	Size    int
}

// GenerateResult is the output of a generation run
type GenerateResult struct {
	Intent      AppIntent
	PluginID    string
	OutputDir   string
	Files       []GeneratedFile
	FileCount   int
	TotalBytes  int
	Duration    time.Duration
	Errors      []string
}

// Generator orchestrates intent → plan → render → write
type Generator struct {
	planner *Planner
}

// NewGenerator creates a generator
func NewGenerator() *Generator {
	return &Generator{planner: NewPlanner()}
}

// Generate generates a full project from a prompt into outputDir
func (g *Generator) Generate(ctx context.Context, prompt, outputDir string) (*GenerateResult, error) {
	start := time.Now()

	// Parse intent
	intent := ParseIntent(prompt)

	// Plan files
	plan := g.planner.Plan(intent, outputDir)

	result := &GenerateResult{
		Intent:    intent,
		PluginID:  plan.PluginID,
		OutputDir: outputDir,
	}

	// Render and write each file
	for _, spec := range plan.Files {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		content, err := Render(spec.Template, spec.Data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", spec.Path, err))
			continue
		}

		fullPath := filepath.Join(outputDir, spec.Path)
		if err := writeFile(fullPath, content); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s: %v", spec.Path, err))
			continue
		}

		gf := GeneratedFile{
			Path:    spec.Path,
			Content: content,
			Size:    len(content),
		}
		result.Files = append(result.Files, gf)
		result.TotalBytes += gf.Size
	}

	result.FileCount = len(result.Files)
	result.Duration = time.Since(start)
	return result, nil
}

// GeneratePreview returns files without writing to disk
func (g *Generator) GeneratePreview(ctx context.Context, prompt string) (*GenerateResult, error) {
	start := time.Now()
	intent := ParseIntent(prompt)
	plan := g.planner.Plan(intent, "")

	result := &GenerateResult{
		Intent:   intent,
		PluginID: plan.PluginID,
	}

	for _, spec := range plan.Files {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		content, err := Render(spec.Template, spec.Data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", spec.Path, err))
			continue
		}

		result.Files = append(result.Files, GeneratedFile{
			Path:    spec.Path,
			Content: content,
			Size:    len(content),
		})
		result.TotalBytes += len(content)
	}

	result.FileCount = len(result.Files)
	result.Duration = time.Since(start)
	return result, nil
}

// Summary returns a human-readable summary of the result
func (r *GenerateResult) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Generated %d files (%d bytes) in %s\n",
		r.FileCount, r.TotalBytes, r.Duration.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("Plugin: %s | Platform: %s | Language: %s\n",
		r.PluginID, r.Intent.Platform, r.Intent.Language))
	if len(r.Intent.Features) > 0 {
		sb.WriteString(fmt.Sprintf("Features: %s\n", strings.Join(r.Intent.Features, ", ")))
	}
	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("Errors: %d\n", len(r.Errors)))
		for _, e := range r.Errors {
			sb.WriteString("  - " + e + "\n")
		}
	}
	return sb.String()
}

// FileTree returns a visual tree of generated files
func (r *GenerateResult) FileTree() string {
	var sb strings.Builder
	sb.WriteString(r.Intent.AppName + "/\n")
	for _, f := range r.Files {
		parts := strings.Split(f.Path, "/")
		indent := strings.Repeat("  ", len(parts)-1)
		sb.WriteString(fmt.Sprintf("%s├── %s\n", indent, parts[len(parts)-1]))
	}
	return sb.String()
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}
