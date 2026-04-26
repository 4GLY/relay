package services

import "time"

type CaptureArtifactInput struct {
	Type       string `json:"type"`
	SourcePath string `json:"source_path"`
	TrustLevel string `json:"trust_level,omitempty"`
}

type CaptureInput struct {
	Project        string                 `json:"project"`
	RepoPath       string                 `json:"repo_path"`
	HandoffPath    string                 `json:"handoff_path"`
	DesignPath     string                 `json:"design_path"`
	ExtraArtifacts []CaptureArtifactInput `json:"extra_artifacts,omitempty"`
	Note           string                 `json:"note"`
	Source         string                 `json:"source"`
	Body           string                 `json:"body"`
	IdempotencyKey string                 `json:"idempotency_key"`
}

type CaptureResult struct {
	ProjectID          string   `json:"project_id"`
	CreatedNoteIDs     []string `json:"created_note_ids"`
	CreatedArtifactIDs []string `json:"created_artifact_ids"`
}

type PromoteInput struct {
	Project           string   `json:"project"`
	Kind              string   `json:"kind"`
	Summary           string   `json:"summary"`
	Reason            string   `json:"reason"`
	SourceNoteIDs     []string `json:"source_note_ids"`
	SourceArtifactIDs []string `json:"source_artifact_ids"`
	IdempotencyKey    string   `json:"idempotency_key"`
}

type PromoteResult struct {
	Kind      string `json:"kind"`
	ObjectID  string `json:"object_id"`
	ProjectID string `json:"project_id"`
}

type ShowInput struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id"`
}

type ShowResult struct {
	ProjectID         string `json:"project_id"`
	NoteCount         int    `json:"note_count"`
	ArtifactCount     int    `json:"artifact_count"`
	DecisionCount     int    `json:"decision_count"`
	OpenQuestionCount int    `json:"open_question_count"`
	LatestPacketID    string `json:"latest_packet_id,omitempty"`
}

type ProjectGraphInput struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id"`
}

type ProjectGraphResult struct {
	ProjectID string             `json:"project_id"`
	Nodes     []ProjectGraphNode `json:"nodes"`
	Edges     []ProjectGraphEdge `json:"edges"`
}

type ProjectGraphNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title,omitempty"`
	Source     string `json:"source,omitempty"`
	SourcePath string `json:"source_path,omitempty"`
	TrustLevel string `json:"trust_level,omitempty"`
}

type ProjectGraphEdge struct {
	Type        string `json:"type"`
	From        string `json:"from"`
	To          string `json:"to"`
	Status      string `json:"status,omitempty"`
	Score       int    `json:"score,omitempty"`
	WhyIncluded string `json:"why_included,omitempty"`
}

type ProjectRetrieveInput struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id"`
	Query     string `json:"query"`
	Limit     int    `json:"limit,omitempty"`
}

type ProjectRetrieveResult struct {
	ProjectID string               `json:"project_id"`
	Query     string               `json:"query"`
	Hits      []ProjectRetrieveHit `json:"hits"`
}

type ProjectRetrieveHit struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	Score        int      `json:"score"`
	Title        string   `json:"title,omitempty"`
	Excerpt      string   `json:"excerpt,omitempty"`
	Source       string   `json:"source,omitempty"`
	SourcePath   string   `json:"source_path,omitempty"`
	TrustLevel   string   `json:"trust_level,omitempty"`
	WhyIncluded  string   `json:"why_included,omitempty"`
	SourceRefIDs []string `json:"source_ref_ids,omitempty"`
}

type PacketBuildInput struct {
	Project          string `json:"project"`
	Type             string `json:"type"`
	Target           string `json:"target"`
	Workflow         string `json:"workflow,omitempty"`
	ArtifactType     string `json:"artifact_type,omitempty"`
	TaskSummary      string `json:"task_summary,omitempty"`
	DisableStyleCues bool   `json:"disable_style_cues,omitempty"`
	DisableRetrieval bool   `json:"disable_retrieval,omitempty"`
	PersistSnapshot  bool   `json:"persist_snapshot,omitempty"`
	IdempotencyKey   string `json:"idempotency_key,omitempty"`
}

type PacketBuildResult struct {
	PacketID             string           `json:"packet_id"`
	SnapshotID           string           `json:"snapshot_id,omitempty"`
	ProjectID            string           `json:"project_id"`
	SchemaVersion        string           `json:"schema_version,omitempty"`
	Type                 string           `json:"type"`
	Target               string           `json:"target"`
	TaskSummary          string           `json:"task_summary,omitempty"`
	RetrievalMode        string           `json:"retrieval_mode,omitempty"`
	Body                 string           `json:"body"`
	RenderedBody         string           `json:"rendered_body,omitempty"`
	StyleCues            []PacketStyleCue `json:"style_cues,omitempty"`
	SupportingNotes      []PacketNote     `json:"supporting_notes,omitempty"`
	SupportingDecisions  []PacketDecision `json:"supporting_decisions,omitempty"`
	SupportingQuestions  []PacketQuestion `json:"supporting_questions,omitempty"`
	SupportingArtifacts  []PacketArtifact `json:"supporting_artifacts,omitempty"`
	WhyIncluded          []string         `json:"why_included,omitempty"`
	DecisionIDs          []string         `json:"decision_ids"`
	OpenQuestionIDs      []string         `json:"open_question_ids"`
	SourceArtifactIDs    []string         `json:"source_artifact_ids"`
	ApprovedHeuristicIDs []string         `json:"approved_heuristic_ids,omitempty"`
	MissingContext       []string         `json:"missing_context"`
}

type PacketSnapshotReadInput struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id,omitempty"`
	Type      string `json:"type,omitempty"`
	Target    string `json:"target,omitempty"`
}

type PacketSnapshotReadResult struct {
	SnapshotID           string           `json:"snapshot_id"`
	ProjectID            string           `json:"project_id"`
	SchemaVersion        string           `json:"schema_version,omitempty"`
	Type                 string           `json:"type"`
	Target               string           `json:"target"`
	TaskSummary          string           `json:"task_summary,omitempty"`
	RenderedBody         string           `json:"rendered_body,omitempty"`
	StyleCues            []PacketStyleCue `json:"style_cues,omitempty"`
	SupportingNotes      []PacketNote     `json:"supporting_notes,omitempty"`
	SupportingDecisions  []PacketDecision `json:"supporting_decisions,omitempty"`
	SupportingQuestions  []PacketQuestion `json:"supporting_questions,omitempty"`
	SupportingArtifacts  []PacketArtifact `json:"supporting_artifacts,omitempty"`
	WhyIncluded          []string         `json:"why_included,omitempty"`
	ApprovedHeuristicIDs []string         `json:"approved_heuristic_ids,omitempty"`
	DecisionIDs          []string         `json:"decision_ids"`
	OpenQuestionIDs      []string         `json:"open_question_ids"`
	SourceArtifactIDs    []string         `json:"source_artifact_ids"`
	MissingContext       []string         `json:"missing_context"`
	CreatedAt            time.Time        `json:"created_at"`
}

type PacketStyleCue struct {
	HeuristicID   string   `json:"heuristic_id"`
	HeuristicKey  string   `json:"heuristic_key,omitempty"`
	CanonicalText string   `json:"canonical_text,omitempty"`
	WhySelected   string   `json:"why_selected"`
	WhyIncluded   string   `json:"why_included,omitempty"`
	SourceSummary string   `json:"source_summary"`
	SourceRefs    []string `json:"source_refs,omitempty"`
}

type PacketNote struct {
	NoteID   string `json:"note_id"`
	Source   string `json:"source,omitempty"`
	Excerpt  string `json:"excerpt"`
	Evidence string `json:"evidence,omitempty"`
}

type PacketDecision struct {
	DecisionID string `json:"decision_id"`
	Summary    string `json:"summary"`
	Why        string `json:"why,omitempty"`
}

type PacketQuestion struct {
	QuestionID string `json:"question_id"`
	Summary    string `json:"summary"`
}

type PacketArtifact struct {
	ArtifactID  string `json:"artifact_id"`
	Type        string `json:"type"`
	SourcePath  string `json:"source_path,omitempty"`
	TrustLevel  string `json:"trust_level,omitempty"`
	WhyIncluded string `json:"why_included,omitempty"`
}

type JudgmentTraceWriteInput struct {
	Project        string   `json:"project"`
	ProjectID      string   `json:"project_id,omitempty"`
	TaskID         string   `json:"task_id"`
	AgentID        string   `json:"agent_id"`
	Workflow       string   `json:"workflow"`
	ArtifactType   string   `json:"artifact_type"`
	Decision       string   `json:"decision"`
	Alternatives   []string `json:"alternatives,omitempty"`
	Rationale      string   `json:"rationale"`
	Constraints    []string `json:"constraints,omitempty"`
	SourceRefs     []string `json:"source_refs,omitempty"`
	Language       string   `json:"language,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}

type JudgmentTraceWriteResult struct {
	TraceID      string `json:"trace_id"`
	ProjectID    string `json:"project_id"`
	CuratorJobID string `json:"curator_job_id,omitempty"`
}

type ListHeuristicProposalsInput struct {
	Project   string `json:"project,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	State     string `json:"state,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Cursor    string `json:"cursor,omitempty"`
}

type ListHeuristicProposalsResult struct {
	Items      []PendingProposalSummary `json:"items"`
	NextCursor string                   `json:"next_cursor,omitempty"`
}

type PendingProposalSummary struct {
	ProposalID     string    `json:"proposal_id"`
	ProjectID      string    `json:"project_id"`
	OriginTraceID  string    `json:"origin_trace_id,omitempty"`
	Workflow       string    `json:"workflow,omitempty"`
	ArtifactType   string    `json:"artifact_type,omitempty"`
	HeuristicKey   string    `json:"heuristic_key"`
	CanonicalText  string    `json:"canonical_text"`
	NormalizedText string    `json:"normalized_text,omitempty"`
	State          string    `json:"state"`
	SourceTraceIDs []string  `json:"source_trace_ids,omitempty"`
	SourceRefs     []string  `json:"source_refs,omitempty"`
	ProposedBy     string    `json:"proposed_by,omitempty"`
	ReviewNotes    string    `json:"review_notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ListApprovedHeuristicsInput struct {
	Project      string `json:"project,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	Workflow     string `json:"workflow,omitempty"`
	ArtifactType string `json:"artifact_type,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Cursor       string `json:"cursor,omitempty"`
}

type ListApprovedHeuristicsResult struct {
	Items      []ApprovedHeuristicSummary `json:"items"`
	NextCursor string                     `json:"next_cursor,omitempty"`
}

type ApprovedHeuristicSummary struct {
	HeuristicID      string    `json:"heuristic_id"`
	ProjectID        string    `json:"project_id"`
	OriginProposalID string    `json:"origin_proposal_id,omitempty"`
	Workflow         string    `json:"workflow,omitempty"`
	ArtifactType     string    `json:"artifact_type,omitempty"`
	HeuristicKey     string    `json:"heuristic_key"`
	CanonicalText    string    `json:"canonical_text"`
	State            string    `json:"state"`
	SourceTraceIDs   []string  `json:"source_trace_ids,omitempty"`
	SourceRefs       []string  `json:"source_refs,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type HeuristicProposalCreateInput struct {
	Project        string   `json:"project"`
	ProjectID      string   `json:"project_id,omitempty"`
	OriginTraceID  string   `json:"origin_trace_id,omitempty"`
	Workflow       string   `json:"workflow,omitempty"`
	ArtifactType   string   `json:"artifact_type,omitempty"`
	HeuristicKey   string   `json:"heuristic_key"`
	CanonicalText  string   `json:"canonical_text"`
	NormalizedText string   `json:"normalized_text,omitempty"`
	SourceTraceIDs []string `json:"source_trace_ids,omitempty"`
	SourceRefs     []string `json:"source_refs,omitempty"`
	ProposedBy     string   `json:"proposed_by,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}

type HeuristicProposalCreateResult struct {
	ProposalID string `json:"proposal_id"`
	ProjectID  string `json:"project_id"`
	State      string `json:"state"`
}

type HeuristicProposalReviewInput struct {
	Project     string `json:"project"`
	ProjectID   string `json:"project_id,omitempty"`
	ProposalID  string `json:"proposal_id"`
	Action      string `json:"action"`
	ReviewNotes string `json:"review_notes,omitempty"`
}

type HeuristicProposalReviewResult struct {
	ProposalID          string `json:"proposal_id"`
	ApprovedHeuristicID string `json:"approved_heuristic_id,omitempty"`
	ProjectID           string `json:"project_id"`
	State               string `json:"state"`
}

type ApprovedHeuristicUpdateInput struct {
	Project     string `json:"project"`
	ProjectID   string `json:"project_id,omitempty"`
	HeuristicID string `json:"heuristic_id"`
	Action      string `json:"action"`
}

type ApprovedHeuristicUpdateResult struct {
	HeuristicID string `json:"heuristic_id"`
	ProjectID   string `json:"project_id"`
	State       string `json:"state"`
}

type IssueAPIKeyInput struct {
	Name      string `json:"name"`
	Scope     string `json:"scope,omitempty"`
	Project   string `json:"project,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

type IssueAPIKeyResult struct {
	KeyID       string `json:"key_id"`
	Name        string `json:"name"`
	Token       string `json:"token"`
	TokenPrefix string `json:"token_prefix"`
	Scope       string `json:"scope"`
	ProjectID   string `json:"project_id,omitempty"`
}

type APIKeySummary struct {
	KeyID       string `json:"key_id"`
	Name        string `json:"name"`
	TokenPrefix string `json:"token_prefix"`
	Scope       string `json:"scope"`
	ProjectID   string `json:"project_id,omitempty"`
	Revoked     bool   `json:"revoked"`
}

type ListAPIKeysResult struct {
	Items []APIKeySummary `json:"items"`
}

type RevokeAPIKeyInput struct {
	KeyID string `json:"key_id"`
}

type RevokeAPIKeyResult struct {
	KeyID       string `json:"key_id"`
	Name        string `json:"name"`
	TokenPrefix string `json:"token_prefix"`
	Scope       string `json:"scope"`
	ProjectID   string `json:"project_id,omitempty"`
	Revoked     bool   `json:"revoked"`
}

type CuratorRunOptions struct {
	Owner         string
	BatchSize     int
	LeaseDuration time.Duration
	RetryBackoff  time.Duration
	MaxAttempts   int
}

type CuratorRunResult struct {
	Claimed      int      `json:"claimed"`
	Completed    int      `json:"completed"`
	Retried      int      `json:"retried"`
	Failed       int      `json:"failed"`
	ProposalIDs  []string `json:"proposal_ids"`
	ProcessedIDs []string `json:"processed_job_ids"`
}
