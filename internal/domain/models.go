package domain

import "time"

type Project struct {
	ID          string
	Name        string
	RootPath    string
	Status      string
	OwnerUserID string
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
	OwnerUserID string
	Revoked     bool
}

type User struct {
	ID          string
	Email       string
	DisplayName string
	AvatarURL   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OAuthIdentity struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID string
	ProviderLogin  string
	VerifiedEmail  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserSession struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type OAuthState struct {
	ID         string
	Provider   string
	RedirectTo string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	ConsumedAt *time.Time
}

// UserOnboarding holds per-user onboarding state including the envelope-encrypted
// Anthropic API key. Key material is nullable: a row whose ciphertext/nonce/salt
// columns are NULL represents a user who completed onboarding once and then
// invoked DELETE /v1/onboarding (D5). The project row is preserved across the
// delete so the user can re-onboard without losing their default project.
//
// AnthropicKeyKEKVersion stays non-null in storage so V2.5 rotation can join
// against it; the application layer regenerates AadSalt on every upsert (E8).
type UserOnboarding struct {
	UserID                 string
	AnthropicKeyCiphertext []byte
	AnthropicKeyNonce      []byte
	AnthropicKeyKEKVersion uint8
	AnthropicKeyPrefix     string
	AnthropicKeyLast4      string
	AadSalt                []byte
	DefaultProjectID       string
	OnboardingCompletedAt  *time.Time
	LastValidatedAt        *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
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
	StyleCues            []byte
	SupportingNotes      []byte
	SupportingDecisions  []byte
	SupportingQuestions  []byte
	SupportingArtifacts  []byte
	WhyIncluded          []string
	ApprovedHeuristicIDs []string
	DecisionIDs          []string
	OpenQuestionIDs      []string
	SourceArtifactIDs    []string
	MissingContext       []string
	PublicReadable       bool
	PublicToken          string
	OGImagePath          string
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
