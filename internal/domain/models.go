package domain

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
