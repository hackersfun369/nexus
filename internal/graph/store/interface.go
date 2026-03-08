package store

import (
	"context"
	"time"
)

type GraphStore interface {
	CreateProject(ctx context.Context, p Project) error
	GetProject(ctx context.Context, id string) (Project, error)
	UpdateProject(ctx context.Context, p Project) error
	DeleteProject(ctx context.Context, id string) error

	WriteModule(ctx context.Context, m Module) error
	GetModule(ctx context.Context, id string) (Module, error)
	GetModuleByPath(ctx context.Context, projectID, filePath string) (Module, error)
	QueryModules(ctx context.Context, f ModuleFilter) ([]Module, error)
	DeleteModule(ctx context.Context, id string) error

	WriteFunction(ctx context.Context, f Function) error
	GetFunction(ctx context.Context, id string) (Function, error)
	QueryFunctions(ctx context.Context, f FunctionFilter) ([]Function, error)
	DeleteFunction(ctx context.Context, id string) error

	WriteClass(ctx context.Context, c Class) error
	GetClass(ctx context.Context, id string) (Class, error)
	QueryClasses(ctx context.Context, f ClassFilter) ([]Class, error)
	DeleteClass(ctx context.Context, id string) error

	WriteIssue(ctx context.Context, i Issue) error
	GetIssue(ctx context.Context, id string) (Issue, error)
	QueryIssues(ctx context.Context, f IssueFilter) ([]Issue, error)
	UpdateIssueStatus(ctx context.Context, id string, status IssueStatus) error

	WriteEdge(ctx context.Context, e Edge) error
	GetEdge(ctx context.Context, id string) (Edge, error)
	QueryEdges(ctx context.Context, f EdgeFilter) ([]Edge, error)
	DeleteEdge(ctx context.Context, id string) error

	GetProjectHealth(ctx context.Context, projectID string) (ProjectHealth, error)

	Migrate(ctx context.Context) error
	Close() error
}

type Project struct {
	ID              string
	Name            string
	RootPath        string
	Platform        string
	PrimaryLanguage string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastAnalyzed    *time.Time
	Version         string
	Description     string
}

type Module struct {
	ID            string
	ProjectID     string
	FilePath      string
	QualifiedName string
	Language      string
	LinesOfCode   int
	ParseStatus   string
	ParseErrors   string
	CycleRisk     float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Checksum      string
}

type Function struct {
	ID                   string
	ProjectID            string
	ModuleID             string
	Name                 string
	QualifiedName        string
	Language             string
	StartLine            int
	StartCol             int
	EndLine              int
	EndCol               int
	Visibility           string
	Parameters           string
	ReturnType           string
	IsAsync              bool
	IsStatic             bool
	IsAbstract           bool
	IsConstructor        bool
	CyclomaticComplexity int
	LinesOfCode          int
	ParameterCount       int
	NestingDepth         int
	FanIn                int
	FanOut               int
	TestCoverage         *float64
	DocComment           string
	Annotations          string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Checksum             string
}

type Class struct {
	ID                     string
	ProjectID              string
	ModuleID               string
	Name                   string
	QualifiedName          string
	Language               string
	Kind                   string
	StartLine              int
	StartCol               int
	EndLine                int
	EndCol                 int
	Visibility             string
	MethodCount            int
	FieldCount             int
	LinesOfCode            int
	LackOfCohesion         float64
	CouplingBetweenObjects int
	DocComment             string
	Annotations            string
	IsAbstract             bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
	Checksum               string
}

type Issue struct {
	ID                  string
	NodeID              string
	ProjectID           string
	RuleID              string
	Severity            string
	Category            string
	Title               string
	Description         string
	FilePath            string
	StartLine           int
	StartCol            int
	Evidence            string
	Remediation         string
	InferenceChain      string
	CWE                 string
	OWASP               string
	Status              IssueStatus
	DetectedAt          time.Time
	ResolvedAt          *time.Time
	ResolvedBy          string
	FalsePositiveReason string
}

type Edge struct {
	ID         string
	ProjectID  string
	Kind       string
	FromNodeID string
	ToNodeID   string
	Properties string
	CreatedAt  time.Time
}

type ProjectHealth struct {
	CriticalIssues   int
	HighIssues       int
	MediumIssues     int
	ComplexFunctions int
	TotalFunctions   int
	TotalClasses     int
	TotalModules     int
}

type ModuleFilter struct {
	ProjectID string
	Language  string
	Limit     int
	Offset    int
}

type FunctionFilter struct {
	ProjectID     string
	ModuleID      string
	MinComplexity int
	Limit         int
	Offset        int
}

type ClassFilter struct {
	ProjectID string
	ModuleID  string
	Limit     int
	Offset    int
}

type IssueFilter struct {
	ProjectID string
	Severity  string
	Status    IssueStatus
	Category  string
	Limit     int
	Offset    int
}

type EdgeFilter struct {
	ProjectID  string
	FromNodeID string
	ToNodeID   string
	Kind       string
}

type IssueStatus string

const (
	IssueStatusOpen          IssueStatus = "OPEN"
	IssueStatusAcknowledged  IssueStatus = "ACKNOWLEDGED"
	IssueStatusResolved      IssueStatus = "RESOLVED"
	IssueStatusFalsePositive IssueStatus = "FALSE_POSITIVE"
)

type NotFoundError struct {
	Kind string
	ID   string
}

func (e *NotFoundError) Error() string {
	return e.Kind + " not found: " + e.ID
}
