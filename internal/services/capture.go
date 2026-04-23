package services

import (
	"context"
	"path/filepath"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Service) Capture(ctx context.Context, input CaptureInput) (CaptureResult, error) {
	projectName := input.Project
	if projectName == "" && input.RepoPath != "" {
		projectName = filepath.Base(input.RepoPath)
	}

	if err := validateCaptureInput(input); err != nil {
		return CaptureResult{}, err
	}
	if projectName != "" {
		if err := validateStringFieldLength("project", projectName, maxCaptureProjectLength); err != nil {
			return CaptureResult{}, err
		}
	}

	projectID := ""
	if projectName != "" {
		project, err := s.resolveCaptureProject(ctx, projectName, input.RepoPath)
		if err != nil {
			return CaptureResult{}, err
		}
		projectID = project.ID
	} else if auth, ok := AuthInfoFromContext(ctx); ok && NormalizeAPIKeyScope(auth.Scope) == APIKeyScopeProject {
		project, err := s.resolveCaptureProject(ctx, "", input.RepoPath)
		if err != nil {
			return CaptureResult{}, err
		}
		projectID = project.ID
	}

	result := CaptureResult{ProjectID: projectID}

	if input.Body != "" || input.Note != "" {
		body := input.Body
		source := input.Source
		if body == "" {
			body = input.Note
		}
		if source == "" {
			source = "manual"
		}

		noteID := noteIDFor(input.IdempotencyKey, body)
		note, err := s.deps.Notes.CreateNote(ctx, domain.Note{
			ID:        noteID,
			ProjectID: projectID,
			Source:    source,
			Body:      body,
		})
		if err != nil {
			return CaptureResult{}, err
		}
		result.CreatedNoteIDs = append(result.CreatedNoteIDs, note.ID)
	}

	artifactSpecs := []struct {
		kind     string
		path     string
		trust    string
		idSuffix string
	}{
		{kind: "git_commits", path: input.RepoPath, trust: "trusted", idSuffix: "repo"},
		{kind: "handoff_md", path: input.HandoffPath, trust: "trusted", idSuffix: "handoff"},
		{kind: "design_doc", path: input.DesignPath, trust: "trusted", idSuffix: "design"},
	}

	for _, spec := range artifactSpecs {
		if spec.path == "" || projectID == "" {
			continue
		}
		artifactID := artifactIDFor(input.IdempotencyKey, spec.idSuffix, spec.path)
		artifact, err := s.deps.Artifacts.CreateArtifact(ctx, domain.Artifact{
			ID:         artifactID,
			ProjectID:  projectID,
			Type:       spec.kind,
			SourcePath: spec.path,
			TrustLevel: spec.trust,
		})
		if err != nil {
			return CaptureResult{}, err
		}
		result.CreatedArtifactIDs = append(result.CreatedArtifactIDs, artifact.ID)
	}

	return result, nil
}

func noteIDFor(idempotencyKey string, body string) string {
	if idempotencyKey != "" {
		return lib.StableID("note", "capture:"+idempotencyKey+":note")
	}
	return lib.StableID("note", body)
}

func artifactIDFor(idempotencyKey string, suffix string, sourcePath string) string {
	if idempotencyKey != "" {
		return lib.StableID("art", "capture:"+idempotencyKey+":"+suffix)
	}
	return lib.StableID("art", suffix+":"+sourcePath)
}
