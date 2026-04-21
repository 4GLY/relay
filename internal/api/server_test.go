package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"relay/internal/domain"
	"relay/internal/lib"
	"relay/internal/services"
)

type fakeProjectStore struct {
	projects map[string]domain.Project
}

func (s *fakeProjectStore) EnsureProject(_ context.Context, project domain.Project) (domain.Project, error) {
	if s.projects == nil {
		s.projects = map[string]domain.Project{}
	}
	s.projects[project.Name] = project
	return project, nil
}

func (s *fakeProjectStore) GetByName(_ context.Context, name string) (domain.Project, error) {
	project, ok := s.projects[name]
	if !ok {
		return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
	}
	return project, nil
}

func (s *fakeProjectStore) GetByID(_ context.Context, id string) (domain.Project, error) {
	for _, project := range s.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
}

type fakeNoteStore struct{ items []domain.Note }

func (s *fakeNoteStore) CreateNote(_ context.Context, note domain.Note) (domain.Note, error) {
	s.items = append(s.items, note)
	return note, nil
}
func (s *fakeNoteStore) CountByProject(_ context.Context, projectID string) (int, error) {
	count := 0
	for _, item := range s.items {
		if item.ProjectID == projectID {
			count++
		}
	}
	return count, nil
}
func (s *fakeNoteStore) ListByProject(_ context.Context, projectID string) ([]domain.Note, error) {
	var items []domain.Note
	for _, item := range s.items {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	return items, nil
}

type fakeArtifactStore struct{}

func (s *fakeArtifactStore) CreateArtifact(_ context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	return artifact, nil
}
func (s *fakeArtifactStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeArtifactStore) ListByProject(_ context.Context, _ string) ([]domain.Artifact, error) {
	return nil, nil
}

type fakeDecisionStore struct{}

func (s *fakeDecisionStore) CreateDecision(_ context.Context, decision domain.Decision) (domain.Decision, error) {
	return decision, nil
}
func (s *fakeDecisionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeDecisionStore) ListByProject(_ context.Context, _ string) ([]domain.Decision, error) {
	return nil, nil
}

type fakeOpenQuestionStore struct{}

func (s *fakeOpenQuestionStore) CreateOpenQuestion(_ context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	return question, nil
}
func (s *fakeOpenQuestionStore) CountByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *fakeOpenQuestionStore) ListByProject(_ context.Context, _ string) ([]domain.OpenQuestion, error) {
	return nil, nil
}

type fakePacketStore struct{ latest domain.Packet }

func (s *fakePacketStore) CreatePacket(_ context.Context, packet domain.Packet) (domain.Packet, error) {
	s.latest = packet
	return packet, nil
}
func (s *fakePacketStore) LatestByProject(_ context.Context, _ string) (domain.Packet, error) {
	return s.latest, nil
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleProjectShowUsesProjectID(t *testing.T) {
	projectID := lib.ProjectID("relay")
	handler := Handler{
		services: services.New(services.Dependencies{
			Projects: &fakeProjectStore{
				projects: map[string]domain.Project{
					"relay": {ID: projectID, Name: "relay"},
				},
			},
			Notes: &fakeNoteStore{
				items: []domain.Note{
					{ID: "note_1", ProjectID: projectID, Source: "chat", Body: "hello"},
				},
			},
			Artifacts:     &fakeArtifactStore{},
			Decisions:     &fakeDecisionStore{},
			OpenQuestions: &fakeOpenQuestionStore{},
			Packets:       &fakePacketStore{},
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/projects/"+projectID, bytes.NewReader(nil))
	rec := httptest.NewRecorder()

	handler.handleProjectShow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(projectID)) {
		t.Fatalf("expected response to include project id, got %s", rec.Body.String())
	}
}
