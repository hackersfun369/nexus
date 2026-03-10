package api

import (
	"archive/zip"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hackersfun369/nexus/internal/codegen"
)

type GenerateRequest struct {
	Prompt    string `json:"prompt"`
	Platform  string `json:"platform"`
	Languages []string `json:"languages"`
	OutputDir string `json:"output_dir"`
	Preview   bool   `json:"preview"`
}

type GenerateResponse struct {
	AppName    string          `json:"app_name"`
	Platform   string          `json:"platform"`
	Language   string          `json:"language"`
	Framework  string          `json:"framework"`
	PluginID   string          `json:"plugin_id"`
	Features   []string        `json:"features"`
	Entities   []string        `json:"entities"`
	Files      []FileEntry     `json:"files"`
	FileCount  int             `json:"file_count"`
	TotalBytes int             `json:"total_bytes"`
	DurationMS int64           `json:"duration_ms"`
	OutputDir  string          `json:"output_dir"`
	Errors     []string        `json:"errors"`
	Quality    *QualityReport  `json:"quality,omitempty"`
}

type QualityReport struct {
	Issues     []QualityIssue `json:"issues"`
	IssueCount int            `json:"issue_count"`
	Score      int            `json:"score"`
	Passed     bool           `json:"passed"`
}

type QualityIssue struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
}

type FileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int    `json:"size"`
	Lang    string `json:"lang"`
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	prompt := req.Prompt
	if req.Platform != "" && !strings.Contains(strings.ToLower(prompt), req.Platform) {
		prompt += " for " + req.Platform
	}
	if len(req.Languages) > 0 {
		lang := req.Languages[0]
		if !strings.Contains(strings.ToLower(prompt), lang) {
			prompt += " using " + lang
		}
	}

	g := codegen.NewGenerator()
	ctx := r.Context()

	if req.Preview {
		result, err := g.GeneratePreview(ctx, prompt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, toResponse(result, ""))
		return
	}

	// Write to disk
	outputDir := req.OutputDir
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".nexus", "projects",
			sanitizeName(req.Prompt))
	}

	result, err := g.Generate(ctx, prompt, outputDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toResponse(result, outputDir))
}

func toResponse(result *codegen.GenerateResult, outputDir string) GenerateResponse {
	files := make([]FileEntry, 0, len(result.Files))
	for _, f := range result.Files {
		files = append(files, FileEntry{
			Path:    f.Path,
			Content: f.Content,
			Size:    f.Size,
			Lang:    langFromPath(f.Path),
		})
	}

	// Run quality analysis on generated files
	report := codegen.Analyze(context.Background(), result.Files)
	issues := make([]QualityIssue, 0, len(report.Issues))
	for _, i := range report.Issues {
		issues = append(issues, QualityIssue{
			RuleID:   i.RuleID,
			Severity: i.Severity,
			Category: i.Category,
			File:     i.File,
			Line:     i.Line,
			Message:  i.Message,
			Fix:      i.Fix,
		})
	}
	quality := &QualityReport{
		Issues:     issues,
		IssueCount: report.IssueCount,
		Score:      report.Score,
		Passed:     report.Passed,
	}

	return GenerateResponse{
		AppName:    result.Intent.AppName,
		Platform:   result.Intent.Platform,
		Language:   result.Intent.Language,
		Framework:  result.Intent.Framework,
		PluginID:   result.PluginID,
		Features:   result.Intent.Features,
		Entities:   result.Intent.Entities,
		Files:      files,
		FileCount:  result.FileCount,
		TotalBytes: result.TotalBytes,
		DurationMS: result.Duration.Milliseconds(),
		OutputDir:  outputDir,
		Errors:     result.Errors,
		Quality:    quality,
	}
}

func langFromPath(path string) string {
	switch {
	case strings.HasSuffix(path, ".py"):    return "python"
	case strings.HasSuffix(path, ".go"):    return "go"
	case strings.HasSuffix(path, ".ts"):    return "typescript"
	case strings.HasSuffix(path, ".tsx"):   return "typescript"
	case strings.HasSuffix(path, ".js"):    return "javascript"
	case strings.HasSuffix(path, ".kt"):    return "kotlin"
	case strings.HasSuffix(path, ".dart"):  return "dart"
	case strings.HasSuffix(path, ".swift"): return "swift"
	case strings.HasSuffix(path, ".rs"):    return "rust"
	case strings.HasSuffix(path, ".sql"):   return "sql"
	case strings.HasSuffix(path, ".md"):    return "markdown"
	case strings.HasSuffix(path, ".json"):  return "json"
	case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"): return "yaml"
	case strings.HasSuffix(path, ".toml"):  return "toml"
	case strings.HasSuffix(path, ".xml"):   return "xml"
	case strings.HasSuffix(path, ".html"):  return "html"
	case strings.HasSuffix(path, ".css"):   return "css"
	case strings.HasSuffix(path, "Dockerfile"): return "dockerfile"
	case strings.HasSuffix(path, "Makefile"):   return "makefile"
	default: return "text"
	}
}

func sanitizeName(prompt string) string {
	words := strings.Fields(strings.ToLower(prompt))
	if len(words) > 4 {
		words = words[:4]
	}
	name := strings.Join(words, "-")
	var sb strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			sb.WriteRune(c)
		}
	}
	result := sb.String()
	if result == "" {
		return "nexus-project"
	}
	return result
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) handleDownloadZip(w http.ResponseWriter, r *http.Request) {
	prompt := r.URL.Query().Get("prompt")
	platform := r.URL.Query().Get("platform")
	language := r.URL.Query().Get("language")
	if prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	full := prompt
	if platform != "" && !strings.Contains(strings.ToLower(prompt), platform) {
		full += " for " + platform
	}
	if language != "" && !strings.Contains(strings.ToLower(prompt), language) {
		full += " using " + language
	}

	g := codegen.NewGenerator()
	result, err := g.GeneratePreview(r.Context(), full)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		"attachment; filename=\""+sanitizeName(result.Intent.AppName)+".zip\"")

	zw := newZipWriter(w)
	for _, f := range result.Files {
		if err := zw.addFile(f.Path, f.Content); err != nil {
			continue
		}
	}
	zw.close()
}

type zipWriter struct {
	zw *zip.Writer
}

func newZipWriter(w http.ResponseWriter) *zipWriter {
	return &zipWriter{zw: zip.NewWriter(w)}
}

func (z *zipWriter) addFile(path, content string) error {
	f, err := z.zw.Create(path)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func (z *zipWriter) close() { z.zw.Close() }
