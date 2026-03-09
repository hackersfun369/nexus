package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/hackersfun369/nexus/internal/graph/store"
	"github.com/hackersfun369/nexus/internal/rules"
)

// ── NEXUS-SEC-001: Hardcoded Secrets ─────────────────

type HardcodedSecrets struct {
	rules.BaseRule
}

func NewHardcodedSecrets() *HardcodedSecrets {
	return &HardcodedSecrets{
		BaseRule: rules.NewBaseRule(
			"NEXUS-SEC-001",
			"Hardcoded Secrets",
			rules.CategorySecurity,
			rules.SeverityCritical,
			"Hardcoded credentials or secrets detected in source code.",
			"Move secrets to environment variables or a secrets manager.",
		),
	}
}

var secretPatterns = []string{
	"password", "passwd", "secret", "api_key", "apikey",
	"token", "auth_token", "access_token", "private_key",
	"aws_secret", "db_password", "database_password",
}

func (r *HardcodedSecrets) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-SEC-001 query functions: %w", err)
	}

	var findings []rules.Finding

	for _, fn := range fns {
		matched := matchesSecretPattern(fn.Name)
		if !matched {
			matched = matchesSecretPattern(fn.QualifiedName)
		}
		if matched {
			findings = append(findings, rules.Finding{
				RuleID:   r.ID(),
				NodeID:   fn.ID,
				FilePath: fn.Language,
				Severity: r.Severity(),
				Category: r.Category(),
				Title:    fmt.Sprintf("Potential hardcoded secret in '%s'", fn.Name),
				Description: fmt.Sprintf(
					"Function '%s' name matches secret patterns. "+
						"Credentials must never be hardcoded in source code.",
					fn.Name,
				),
				StartLine:   fn.StartLine,
				Evidence:    fmt.Sprintf("function_name=%s matches_secret_pattern=true", fn.Name),
				Remediation: "Move credentials to environment variables or a dedicated secrets manager (e.g. Vault, AWS Secrets Manager).",
				CWE:         "CWE-798",
				OWASP:       "A07:2021",
				InferenceChain: []string{
					fmt.Sprintf("function.name=%s", fn.Name),
					"name matches secret pattern",
					"hardcoded secret risk → NEXUS-SEC-001",
				},
			})
		}
	}
	return findings, nil
}

func matchesSecretPattern(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range secretPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// ── NEXUS-SEC-002: SQL Injection Risk ────────────────

type SQLInjectionRisk struct {
	rules.BaseRule
}

func NewSQLInjectionRisk() *SQLInjectionRisk {
	return &SQLInjectionRisk{
		BaseRule: rules.NewBaseRule(
			"NEXUS-SEC-002",
			"SQL Injection Risk",
			rules.CategorySecurity,
			rules.SeverityCritical,
			"Function name suggests SQL query construction without parameterization.",
			"Use parameterized queries or prepared statements.",
		),
	}
}

var sqlPatterns = []string{
	"execute_query", "run_query", "exec_sql", "raw_query",
	"execute_sql", "run_sql", "build_query", "construct_query",
	"format_query", "make_query", "query_string",
}

func (r *SQLInjectionRisk) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-SEC-002 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, sqlPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("SQL injection risk in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests raw SQL construction. "+
					"String-concatenated queries are vulnerable to SQL injection.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function_name=%s matches_sql_pattern=true", fn.Name),
			Remediation: "Use parameterized queries or an ORM. Never concatenate user input into SQL strings.",
			CWE:         "CWE-89",
			OWASP:       "A03:2021",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches sql construction pattern",
				"sql injection risk → NEXUS-SEC-002",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-SEC-003: Weak Cryptography ─────────────────

type WeakCryptography struct {
	rules.BaseRule
}

func NewWeakCryptography() *WeakCryptography {
	return &WeakCryptography{
		BaseRule: rules.NewBaseRule(
			"NEXUS-SEC-003",
			"Weak Cryptography",
			rules.CategorySecurity,
			rules.SeverityHigh,
			"Function name suggests use of weak or deprecated cryptographic algorithms.",
			"Replace with modern algorithms: AES-256, SHA-256, bcrypt, or Argon2.",
		),
	}
}

var weakCryptoPatterns = []string{
	"md5", "sha1", "des", "rc4", "blowfish",
	"base64_password", "rot13", "caesar",
}

func (r *WeakCryptography) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-SEC-003 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, weakCryptoPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Weak cryptography in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests use of a weak or deprecated cryptographic algorithm. "+
					"MD5 and SHA1 are cryptographically broken for security purposes.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function_name=%s matches_weak_crypto_pattern=true", fn.Name),
			Remediation: "Use modern algorithms: AES-256-GCM for encryption, SHA-256/SHA-3 for hashing, bcrypt/Argon2 for passwords.",
			CWE:         "CWE-327",
			OWASP:       "A02:2021",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches weak crypto pattern",
				"weak cryptography risk → NEXUS-SEC-003",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-SEC-004: Overly Broad Exception Handling ───

type OverlyBroadExceptionHandling struct {
	rules.BaseRule
}

func NewOverlyBroadExceptionHandling() *OverlyBroadExceptionHandling {
	return &OverlyBroadExceptionHandling{
		BaseRule: rules.NewBaseRule(
			"NEXUS-SEC-004",
			"Overly Broad Exception Handling",
			rules.CategorySecurity,
			rules.SeverityMedium,
			"Function name suggests swallowing all exceptions, hiding errors.",
			"Catch specific exceptions and handle or log them appropriately.",
		),
	}
}

var broadExceptionPatterns = []string{
	"catch_all", "swallow_exception", "ignore_error",
	"suppress_error", "catch_everything", "handle_all_errors",
	"silent_fail", "silent_error",
}

func (r *OverlyBroadExceptionHandling) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-SEC-004 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, broadExceptionPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Overly broad exception handling in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests catching all exceptions. "+
					"Broad exception handling can hide bugs and security issues.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function_name=%s matches_broad_exception_pattern=true", fn.Name),
			Remediation: "Catch specific exception types. Log unexpected exceptions. Never silently swallow errors.",
			CWE:         "CWE-390",
			OWASP:       "A09:2021",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches broad exception pattern",
				"error suppression risk → NEXUS-SEC-004",
			},
		})
	}
	return findings, nil
}

// ── NEXUS-SEC-005: Insecure Deserialization ───────────

type InsecureDeserialization struct {
	rules.BaseRule
}

func NewInsecureDeserialization() *InsecureDeserialization {
	return &InsecureDeserialization{
		BaseRule: rules.NewBaseRule(
			"NEXUS-SEC-005",
			"Insecure Deserialization",
			rules.CategorySecurity,
			rules.SeverityHigh,
			"Function name suggests deserializing untrusted data.",
			"Validate and sanitize all data before deserialization.",
		),
	}
}

var deserializationPatterns = []string{
	"deserialize", "unpickle", "unmarshal_user",
	"load_user_data", "parse_user_input", "eval_input",
	"exec_input", "load_pickle",
}

func (r *InsecureDeserialization) Analyze(ctx context.Context, projectID string, s store.GraphStore) ([]rules.Finding, error) {
	fns, err := s.QueryFunctions(ctx, store.FunctionFilter{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("NEXUS-SEC-005 query: %w", err)
	}

	var findings []rules.Finding
	for _, fn := range fns {
		if !matchesPatterns(fn.Name, deserializationPatterns) {
			continue
		}
		findings = append(findings, rules.Finding{
			RuleID:   r.ID(),
			NodeID:   fn.ID,
			FilePath: fn.Language,
			Severity: r.Severity(),
			Category: r.Category(),
			Title:    fmt.Sprintf("Insecure deserialization in '%s'", fn.Name),
			Description: fmt.Sprintf(
				"Function '%s' suggests deserializing data that may be untrusted. "+
					"Insecure deserialization can lead to remote code execution.",
				fn.Name,
			),
			StartLine:   fn.StartLine,
			Evidence:    fmt.Sprintf("function_name=%s matches_deserialization_pattern=true", fn.Name),
			Remediation: "Validate data integrity before deserialization. Avoid pickle/eval on untrusted input. Use safe formats like JSON with schema validation.",
			CWE:         "CWE-502",
			OWASP:       "A08:2021",
			InferenceChain: []string{
				fmt.Sprintf("function.name=%s", fn.Name),
				"name matches deserialization pattern",
				"insecure deserialization risk → NEXUS-SEC-005",
			},
		})
	}
	return findings, nil
}

// ── HELPERS ───────────────────────────────────────────

func matchesPatterns(name string, patterns []string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// DefaultSecurityRules returns all security rules
func DefaultSecurityRules() []rules.Rule {
	return []rules.Rule{
		NewHardcodedSecrets(),
		NewSQLInjectionRisk(),
		NewWeakCryptography(),
		NewOverlyBroadExceptionHandling(),
		NewInsecureDeserialization(),
	}
}
