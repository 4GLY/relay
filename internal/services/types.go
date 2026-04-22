package services

type CaptureInput struct {
	Project        string `json:"project"`
	RepoPath       string `json:"repo_path"`
	HandoffPath    string `json:"handoff_path"`
	DesignPath     string `json:"design_path"`
	Note           string `json:"note"`
	Source         string `json:"source"`
	Body           string `json:"body"`
	IdempotencyKey string `json:"idempotency_key"`
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

type PacketBuildInput struct {
	Project string `json:"project"`
	Type    string `json:"type"`
	Target  string `json:"target"`
}

type PacketBuildResult struct {
	PacketID          string   `json:"packet_id"`
	ProjectID         string   `json:"project_id"`
	Type              string   `json:"type"`
	Target            string   `json:"target"`
	Body              string   `json:"body"`
	DecisionIDs       []string `json:"decision_ids"`
	OpenQuestionIDs   []string `json:"open_question_ids"`
	SourceArtifactIDs []string `json:"source_artifact_ids"`
	MissingContext    []string `json:"missing_context"`
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
