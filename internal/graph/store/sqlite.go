package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hackersfun369/nexus/internal/graph/schema"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := schema.OpenDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	store := &SQLiteStore{db: db}
	if err := store.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	return store, nil
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	return schema.NewMigrator(s.db).Migrate()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ── PROJECT ───────────────────────────────────────────

func (s *SQLiteStore) CreateProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO project (id, name, root_path, platform, primary_language,
			created_at, updated_at, version, description)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.RootPath, p.Platform, p.PrimaryLanguage,
		p.CreatedAt, p.UpdatedAt, p.Version, p.Description,
	)
	if err != nil {
		return fmt.Errorf("CreateProject: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (Project, error) {
	var p Project
	var lastAnalyzed sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, root_path, platform, primary_language,
			created_at, updated_at, last_analyzed, version, description
		FROM project WHERE id = ?`, id,
	).Scan(
		&p.ID, &p.Name, &p.RootPath, &p.Platform, &p.PrimaryLanguage,
		&p.CreatedAt, &p.UpdatedAt, &lastAnalyzed, &p.Version, &p.Description,
	)
	if err == sql.ErrNoRows {
		return Project{}, &NotFoundError{Kind: "project", ID: id}
	}
	if err != nil {
		return Project{}, fmt.Errorf("GetProject: %w", err)
	}
	if lastAnalyzed.Valid {
		p.LastAnalyzed = &lastAnalyzed.Time
	}
	return p, nil
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, p Project) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE project SET name=?, platform=?, primary_language=?,
			updated_at=?, version=?, description=?
		WHERE id=?`,
		p.Name, p.Platform, p.PrimaryLanguage,
		time.Now(), p.Version, p.Description, p.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM project WHERE id=?`, id)
	return err
}

// ── MODULES ───────────────────────────────────────────

func (s *SQLiteStore) WriteModule(ctx context.Context, m Module) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO nodes (id, node_type, project_id, checksum, updated_at)
		VALUES (?, 'MODULE', ?, ?, ?)`,
		m.ID, m.ProjectID, m.Checksum, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("WriteModule node: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO modules
			(id, file_path, qualified_name, language, lines_of_code,
			 parse_status, parse_errors, cycle_risk)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.FilePath, m.QualifiedName, m.Language,
		m.LinesOfCode, m.ParseStatus, m.ParseErrors, m.CycleRisk,
	)
	if err != nil {
		return fmt.Errorf("WriteModule: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetModule(ctx context.Context, id string) (Module, error) {
	var m Module
	err := s.db.QueryRowContext(ctx, `
		SELECT m.id, n.project_id, m.file_path, m.qualified_name,
			m.language, m.lines_of_code, m.parse_status,
			m.parse_errors, m.cycle_risk, n.checksum,
			n.created_at, n.updated_at
		FROM modules m JOIN nodes n ON m.id = n.id
		WHERE m.id = ?`, id,
	).Scan(
		&m.ID, &m.ProjectID, &m.FilePath, &m.QualifiedName,
		&m.Language, &m.LinesOfCode, &m.ParseStatus,
		&m.ParseErrors, &m.CycleRisk, &m.Checksum,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Module{}, &NotFoundError{Kind: "module", ID: id}
	}
	if err != nil {
		return Module{}, fmt.Errorf("GetModule: %w", err)
	}
	return m, nil
}

func (s *SQLiteStore) GetModuleByPath(ctx context.Context, projectID, filePath string) (Module, error) {
	var m Module
	err := s.db.QueryRowContext(ctx, `
		SELECT m.id, n.project_id, m.file_path, m.qualified_name,
			m.language, m.lines_of_code, m.parse_status,
			m.parse_errors, m.cycle_risk, n.checksum,
			n.created_at, n.updated_at
		FROM modules m JOIN nodes n ON m.id = n.id
		WHERE m.file_path = ? AND n.project_id = ?`,
		filePath, projectID,
	).Scan(
		&m.ID, &m.ProjectID, &m.FilePath, &m.QualifiedName,
		&m.Language, &m.LinesOfCode, &m.ParseStatus,
		&m.ParseErrors, &m.CycleRisk, &m.Checksum,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Module{}, &NotFoundError{Kind: "module", ID: filePath}
	}
	if err != nil {
		return Module{}, fmt.Errorf("GetModuleByPath: %w", err)
	}
	return m, nil
}

func (s *SQLiteStore) QueryModules(ctx context.Context, f ModuleFilter) ([]Module, error) {
	query := `
		SELECT m.id, n.project_id, m.file_path, m.qualified_name,
			m.language, m.lines_of_code, m.parse_status,
			m.parse_errors, m.cycle_risk, n.checksum,
			n.created_at, n.updated_at
		FROM modules m JOIN nodes n ON m.id = n.id
		WHERE n.project_id = ? AND n.is_deleted = FALSE`
	args := []interface{}{f.ProjectID}
	if f.Language != "" {
		query += " AND m.language = ?"
		args = append(args, f.Language)
	}
	query += " ORDER BY m.file_path"
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryModules: %w", err)
	}
	defer rows.Close()
	modules := []Module{}
	for rows.Next() {
		var m Module
		if err := rows.Scan(
			&m.ID, &m.ProjectID, &m.FilePath, &m.QualifiedName,
			&m.Language, &m.LinesOfCode, &m.ParseStatus,
			&m.ParseErrors, &m.CycleRisk, &m.Checksum,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, rows.Err()
}

func (s *SQLiteStore) DeleteModule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET is_deleted=TRUE, updated_at=? WHERE id=?`,
		time.Now(), id,
	)
	return err
}

// ── FUNCTIONS ─────────────────────────────────────────

func (s *SQLiteStore) WriteFunction(ctx context.Context, f Function) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO nodes (id, node_type, project_id, checksum, updated_at)
		VALUES (?, 'FUNCTION', ?, ?, ?)`,
		f.ID, f.ProjectID, f.Checksum, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("WriteFunction node: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO functions
			(id, name, qualified_name, module_id, language,
			 start_line, start_col, end_line, end_col,
			 visibility, parameters, return_type,
			 is_async, is_static, is_abstract, is_constructor,
			 cyclomatic_complexity, lines_of_code, parameter_count,
			 nesting_depth, fan_in, fan_out, test_coverage, doc_comment, annotations)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		f.ID, f.Name, f.QualifiedName, f.ModuleID, f.Language,
		f.StartLine, f.StartCol, f.EndLine, f.EndCol,
		f.Visibility, f.Parameters, f.ReturnType,
		f.IsAsync, f.IsStatic, f.IsAbstract, f.IsConstructor,
		f.CyclomaticComplexity, f.LinesOfCode, f.ParameterCount,
		f.NestingDepth, f.FanIn, f.FanOut, f.TestCoverage, f.DocComment, f.Annotations,
	)
	if err != nil {
		return fmt.Errorf("WriteFunction: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetFunction(ctx context.Context, id string) (Function, error) {
	var f Function
	err := s.db.QueryRowContext(ctx, `
		SELECT f.id, n.project_id, f.module_id, f.name, f.qualified_name,
			f.language, f.start_line, f.start_col, f.end_line, f.end_col,
			f.visibility, f.parameters, f.return_type,
			f.is_async, f.is_static, f.is_abstract, f.is_constructor,
			f.cyclomatic_complexity, f.lines_of_code, f.parameter_count,
			f.nesting_depth, f.fan_in, f.fan_out, f.test_coverage, f.doc_comment, f.annotations,
			n.checksum, n.created_at, n.updated_at
		FROM functions f JOIN nodes n ON f.id = n.id
		WHERE f.id = ?`, id,
	).Scan(
		&f.ID, &f.ProjectID, &f.ModuleID, &f.Name, &f.QualifiedName,
		&f.Language, &f.StartLine, &f.StartCol, &f.EndLine, &f.EndCol,
		&f.Visibility, &f.Parameters, &f.ReturnType,
		&f.IsAsync, &f.IsStatic, &f.IsAbstract, &f.IsConstructor,
		&f.CyclomaticComplexity, &f.LinesOfCode, &f.ParameterCount,
		&f.NestingDepth, &f.FanIn, &f.FanOut, &f.TestCoverage, &f.DocComment, &f.Annotations,
		&f.Checksum, &f.CreatedAt, &f.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Function{}, &NotFoundError{Kind: "function", ID: id}
	}
	if err != nil {
		return Function{}, fmt.Errorf("GetFunction: %w", err)
	}
	return f, nil
}

func (s *SQLiteStore) QueryFunctions(ctx context.Context, f FunctionFilter) ([]Function, error) {
	query := `
		SELECT f.id, n.project_id, f.module_id, f.name, f.qualified_name,
			f.language, f.start_line, f.start_col, f.end_line, f.end_col,
			f.visibility, f.parameters, f.return_type,
			f.is_async, f.is_static, f.is_abstract, f.is_constructor,
			f.cyclomatic_complexity, f.lines_of_code, f.parameter_count,
			f.nesting_depth, f.fan_in, f.fan_out, f.test_coverage, f.doc_comment, f.annotations,
			n.checksum, n.created_at, n.updated_at
		FROM functions f JOIN nodes n ON f.id = n.id
		WHERE n.project_id = ? AND n.is_deleted = FALSE`
	args := []interface{}{f.ProjectID}
	if f.ModuleID != "" {
		query += " AND f.module_id = ?"
		args = append(args, f.ModuleID)
	}
	if f.MinComplexity > 0 {
		query += " AND f.cyclomatic_complexity >= ?"
		args = append(args, f.MinComplexity)
	}
	query += " ORDER BY f.cyclomatic_complexity DESC"
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryFunctions: %w", err)
	}
	defer rows.Close()
	functions := []Function{}
	for rows.Next() {
		var fn Function
		if err := rows.Scan(
			&fn.ID, &fn.ProjectID, &fn.ModuleID, &fn.Name, &fn.QualifiedName,
			&fn.Language, &fn.StartLine, &fn.StartCol, &fn.EndLine, &fn.EndCol,
			&fn.Visibility, &fn.Parameters, &fn.ReturnType,
			&fn.IsAsync, &fn.IsStatic, &fn.IsAbstract, &fn.IsConstructor,
			&fn.CyclomaticComplexity, &fn.LinesOfCode, &fn.ParameterCount,
			&fn.NestingDepth, &fn.FanIn, &fn.FanOut, &fn.TestCoverage, &fn.DocComment, &fn.Annotations,
			&fn.Checksum, &fn.CreatedAt, &fn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		functions = append(functions, fn)
	}
	return functions, rows.Err()
}

func (s *SQLiteStore) DeleteFunction(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET is_deleted=TRUE, updated_at=? WHERE id=?`,
		time.Now(), id,
	)
	return err
}

// ── CLASSES ───────────────────────────────────────────

func (s *SQLiteStore) WriteClass(ctx context.Context, c Class) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO nodes (id, node_type, project_id, checksum, updated_at)
		VALUES (?, 'CLASS', ?, ?, ?)`,
		c.ID, c.ProjectID, c.Checksum, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("WriteClass node: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO classes
			(id, name, qualified_name, module_id, language, kind,
			 start_line, start_col, end_line, end_col,
			 visibility, method_count, field_count, lines_of_code,
			 lack_of_cohesion, coupling_between_objects,
			 doc_comment, annotations, is_abstract)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.QualifiedName, c.ModuleID, c.Language, c.Kind,
		c.StartLine, c.StartCol, c.EndLine, c.EndCol,
		c.Visibility, c.MethodCount, c.FieldCount, c.LinesOfCode,
		c.LackOfCohesion, c.CouplingBetweenObjects,
		c.DocComment, c.Annotations, c.IsAbstract,
	)
	if err != nil {
		return fmt.Errorf("WriteClass: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetClass(ctx context.Context, id string) (Class, error) {
	var c Class
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, n.project_id, c.module_id, c.name, c.qualified_name,
			c.language, c.kind, c.start_line, c.start_col,
			c.end_line, c.end_col, c.visibility,
			c.method_count, c.field_count, c.lines_of_code,
			c.lack_of_cohesion, c.coupling_between_objects,
			c.doc_comment, c.annotations, c.is_abstract,
			n.checksum, n.created_at, n.updated_at
		FROM classes c JOIN nodes n ON c.id = n.id
		WHERE c.id = ?`, id,
	).Scan(
		&c.ID, &c.ProjectID, &c.ModuleID, &c.Name, &c.QualifiedName,
		&c.Language, &c.Kind, &c.StartLine, &c.StartCol,
		&c.EndLine, &c.EndCol, &c.Visibility,
		&c.MethodCount, &c.FieldCount, &c.LinesOfCode,
		&c.LackOfCohesion, &c.CouplingBetweenObjects,
		&c.DocComment, &c.Annotations, &c.IsAbstract,
		&c.Checksum, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return Class{}, &NotFoundError{Kind: "class", ID: id}
	}
	if err != nil {
		return Class{}, fmt.Errorf("GetClass: %w", err)
	}
	return c, nil
}

func (s *SQLiteStore) QueryClasses(ctx context.Context, f ClassFilter) ([]Class, error) {
	query := `
		SELECT c.id, n.project_id, c.module_id, c.name, c.qualified_name,
			c.language, c.kind, c.start_line, c.start_col,
			c.end_line, c.end_col, c.visibility,
			c.method_count, c.field_count, c.lines_of_code,
			c.lack_of_cohesion, c.coupling_between_objects,
			c.doc_comment, c.annotations, c.is_abstract,
			n.checksum, n.created_at, n.updated_at
		FROM classes c JOIN nodes n ON c.id = n.id
		WHERE n.project_id = ? AND n.is_deleted = FALSE`
	args := []interface{}{f.ProjectID}
	if f.ModuleID != "" {
		query += " AND c.module_id = ?"
		args = append(args, f.ModuleID)
	}
	query += " ORDER BY c.name"
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryClasses: %w", err)
	}
	defer rows.Close()
	classes := []Class{}
	for rows.Next() {
		var c Class
		if err := rows.Scan(
			&c.ID, &c.ProjectID, &c.ModuleID, &c.Name, &c.QualifiedName,
			&c.Language, &c.Kind, &c.StartLine, &c.StartCol,
			&c.EndLine, &c.EndCol, &c.Visibility,
			&c.MethodCount, &c.FieldCount, &c.LinesOfCode,
			&c.LackOfCohesion, &c.CouplingBetweenObjects,
			&c.DocComment, &c.Annotations, &c.IsAbstract,
			&c.Checksum, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}
	return classes, rows.Err()
}

func (s *SQLiteStore) DeleteClass(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE nodes SET is_deleted=TRUE, updated_at=? WHERE id=?`,
		time.Now(), id,
	)
	return err
}

// ── ISSUES ────────────────────────────────────────────

func (s *SQLiteStore) WriteIssue(ctx context.Context, i Issue) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO issues
			(id, node_id, project_id, rule_id, severity, category,
			 title, description, file_path, start_line, start_col,
			 evidence, remediation, inference_chain, cwe, owasp, status)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		i.ID, i.NodeID, i.ProjectID, i.RuleID, i.Severity, i.Category,
		i.Title, i.Description, i.FilePath, i.StartLine, i.StartCol,
		i.Evidence, i.Remediation, i.InferenceChain, i.CWE, i.OWASP,
		string(i.Status),
	)
	if err != nil {
		return fmt.Errorf("WriteIssue: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetIssue(ctx context.Context, id string) (Issue, error) {
	var i Issue
	var status string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, project_id, rule_id, severity, category,
			title, description, file_path, start_line, start_col,
			evidence, remediation, inference_chain, cwe, owasp,
			status, detected_at
		FROM issues WHERE id = ?`, id,
	).Scan(
		&i.ID, &i.NodeID, &i.ProjectID, &i.RuleID, &i.Severity, &i.Category,
		&i.Title, &i.Description, &i.FilePath, &i.StartLine, &i.StartCol,
		&i.Evidence, &i.Remediation, &i.InferenceChain, &i.CWE, &i.OWASP,
		&status, &i.DetectedAt,
	)
	if err == sql.ErrNoRows {
		return Issue{}, &NotFoundError{Kind: "issue", ID: id}
	}
	if err != nil {
		return Issue{}, fmt.Errorf("GetIssue: %w", err)
	}
	i.Status = IssueStatus(status)
	return i, nil
}

func (s *SQLiteStore) QueryIssues(ctx context.Context, f IssueFilter) ([]Issue, error) {
	query := `
		SELECT id, node_id, project_id, rule_id, severity, category,
			title, description, file_path, start_line, start_col,
			evidence, remediation, inference_chain, cwe, owasp,
			status, detected_at
		FROM issues WHERE project_id = ?`
	args := []interface{}{f.ProjectID}
	if f.Severity != "" {
		query += " AND severity = ?"
		args = append(args, f.Severity)
	}
	if f.Status != "" {
		query += " AND status = ?"
		args = append(args, string(f.Status))
	}
	if f.Category != "" {
		query += " AND category = ?"
		args = append(args, f.Category)
	}
	query += " ORDER BY CASE severity WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 WHEN 'MEDIUM' THEN 3 WHEN 'LOW' THEN 4 ELSE 5 END, detected_at DESC"
	if f.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryIssues: %w", err)
	}
	defer rows.Close()
	issues := []Issue{}
	for rows.Next() {
		var i Issue
		var status string
		if err := rows.Scan(
			&i.ID, &i.NodeID, &i.ProjectID, &i.RuleID, &i.Severity, &i.Category,
			&i.Title, &i.Description, &i.FilePath, &i.StartLine, &i.StartCol,
			&i.Evidence, &i.Remediation, &i.InferenceChain, &i.CWE, &i.OWASP,
			&status, &i.DetectedAt,
		); err != nil {
			return nil, err
		}
		i.Status = IssueStatus(status)
		issues = append(issues, i)
	}
	return issues, rows.Err()
}

func (s *SQLiteStore) UpdateIssueStatus(ctx context.Context, id string, status IssueStatus) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE issues SET status=? WHERE id=?`,
		string(status), id,
	)
	return err
}

// ── EDGES ─────────────────────────────────────────────

func (s *SQLiteStore) WriteEdge(ctx context.Context, e Edge) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO edges
			(id, edge_type, from_node_id, to_node_id, project_id, properties)
		VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.Kind, e.FromNodeID, e.ToNodeID, e.ProjectID, e.Properties,
	)
	if err != nil {
		return fmt.Errorf("WriteEdge: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetEdge(ctx context.Context, id string) (Edge, error) {
	var e Edge
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, edge_type, from_node_id, to_node_id,
			properties, created_at
		FROM edges WHERE id = ?`, id,
	).Scan(
		&e.ID, &e.ProjectID, &e.Kind, &e.FromNodeID, &e.ToNodeID,
		&e.Properties, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return Edge{}, &NotFoundError{Kind: "edge", ID: id}
	}
	if err != nil {
		return Edge{}, fmt.Errorf("GetEdge: %w", err)
	}
	return e, nil
}

func (s *SQLiteStore) QueryEdges(ctx context.Context, f EdgeFilter) ([]Edge, error) {
	query := `
		SELECT id, project_id, edge_type, from_node_id, to_node_id,
			properties, created_at
		FROM edges WHERE project_id = ?`
	args := []interface{}{f.ProjectID}
	if f.FromNodeID != "" {
		query += " AND from_node_id = ?"
		args = append(args, f.FromNodeID)
	}
	if f.ToNodeID != "" {
		query += " AND to_node_id = ?"
		args = append(args, f.ToNodeID)
	}
	if f.Kind != "" {
		query += " AND edge_type = ?"
		args = append(args, f.Kind)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("QueryEdges: %w", err)
	}
	defer rows.Close()
	edges := []Edge{}
	for rows.Next() {
		var e Edge
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Kind, &e.FromNodeID, &e.ToNodeID,
			&e.Properties, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func (s *SQLiteStore) DeleteEdge(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM edges WHERE id=?`, id)
	return err
}

// ── GRAPH QUERIES ─────────────────────────────────────

func (s *SQLiteStore) GetProjectHealth(ctx context.Context, projectID string) (ProjectHealth, error) {
	var h ProjectHealth
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM issues WHERE project_id=? AND status='OPEN' AND severity='CRITICAL'),
			(SELECT COUNT(*) FROM issues WHERE project_id=? AND status='OPEN' AND severity='HIGH'),
			(SELECT COUNT(*) FROM issues WHERE project_id=? AND status='OPEN' AND severity='MEDIUM'),
			(SELECT COUNT(*) FROM functions f JOIN nodes n ON f.id=n.id WHERE n.project_id=? AND f.cyclomatic_complexity > 15),
			(SELECT COUNT(*) FROM functions f JOIN nodes n ON f.id=n.id WHERE n.project_id=?),
			(SELECT COUNT(*) FROM classes c JOIN nodes n ON c.id=n.id WHERE n.project_id=?),
			(SELECT COUNT(*) FROM modules m JOIN nodes n ON m.id=n.id WHERE n.project_id=?)`,
		projectID, projectID, projectID,
		projectID, projectID, projectID, projectID,
	).Scan(
		&h.CriticalIssues, &h.HighIssues, &h.MediumIssues,
		&h.ComplexFunctions, &h.TotalFunctions,
		&h.TotalClasses, &h.TotalModules,
	)
	if err != nil {
		return ProjectHealth{}, fmt.Errorf("GetProjectHealth: %w", err)
	}
	return h, nil
}
