package domain

import "time"

type Project struct {
	ID       string
	Name     string
	RootPath string
	Status   string
}

type Note struct {
	ID        string
	ProjectID string
	Source    string
	Body      string
}

type Artifact struct {
	ID         string
	ProjectID  string
	Type       string
	SourcePath string
	TrustLevel string
}

type Decision struct {
	ID                string
	ProjectID         string
	Summary           string
	Why               string
	SourceNoteIDs     []string
	SourceArtifactIDs []string
}

type OpenQuestion struct {
	ID                string
	ProjectID         string
	Summary           string
	SourceNoteIDs     []string
	SourceArtifactIDs []string
}

type Packet struct {
	ID                string
	ProjectID         string
	Type              string
	Target            string
	Body              string
	DecisionIDs       []string
	OpenQuestionIDs   []string
	SourceArtifactIDs []string
}

type APIKey struct {
	ID          string
	Name        string
	TokenHash   string
	TokenPrefix string
	Scope       string
	ProjectID   string
	Revoked     bool
}

type JudgmentTrace struct {
	ID           string
	ProjectID    string
	TaskID       string
	AgentID      string
	Workflow     string
	ArtifactType string
	Decision     string
	Alternatives []string
	Rationale    string
	Constraints  []string
	SourceRefs   []string
	Language     string
	CreatedAt    time.Time
}

type HeuristicProposal struct {
	ID             string
	ProjectID      string
	OriginTraceID  string
	Workflow       string
	ArtifactType   string
	HeuristicKey   string
	CanonicalText  string
	NormalizedText string
	State          string
	SourceTraceIDs []string
	SourceRefs     []string
	ProposedBy     string
	ReviewNotes    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ApprovedHeuristic struct {
	ID               string
	ProjectID        string
	OriginProposalID string
	Workflow         string
	ArtifactType     string
	HeuristicKey     string
	CanonicalText    string
	State            string
	SourceTraceIDs   []string
	SourceRefs       []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type PacketSnapshot struct {
	ID                   string
	ProjectID            string
	PacketKind           string
	Target               string
	SchemaVersion        string
	TaskSummary          string
	RenderedBody         string
	ApprovedHeuristicIDs []string
	DecisionIDs          []string
	OpenQuestionIDs      []string
	SourceArtifactIDs    []string
	MissingContext       []string
	CreatedAt            time.Time
}

type IdempotencyRecord struct {
	ID             string
	ScopeKind      string
	ScopeProjectID string
	IdempotencyKey string
	RequestHash    string
	ResponseKind   string
	ResponseID     string
	CreatedAt      time.Time
}

type CuratorJob struct {
	ID             string
	ProjectID      string
	JobKind        string
	State          string
	DedupeKey      string
	Payload        []byte
	AttemptCount   int
	LastError      string
	LeaseOwner     string
	LeaseExpiresAt *time.Time
	AvailableAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
