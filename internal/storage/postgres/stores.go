package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"relay/internal/domain"
	"relay/internal/lib"
)

type Stores struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) Stores {
	return Stores{db: db}
}

func (s Stores) EnsureProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO projects (id, name, root_path, status)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    root_path = COALESCE(EXCLUDED.root_path, projects.root_path),
		    status = EXCLUDED.status
	`, project.ID, project.Name, project.RootPath, project.Status)
	if err != nil {
		return domain.Project{}, err
	}
	return s.GetByName(ctx, project.Name)
}

func (s Stores) GetByName(ctx context.Context, name string) (domain.Project, error) {
	var project domain.Project
	err := s.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(root_path, ''), status
		FROM projects
		WHERE name = $1
	`, name).Scan(&project.ID, &project.Name, &project.RootPath, &project.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
	}
	return project, err
}

func (s Stores) GetByID(ctx context.Context, id string) (domain.Project, error) {
	var project domain.Project
	err := s.db.QueryRow(ctx, `
		SELECT id, name, COALESCE(root_path, ''), status
		FROM projects
		WHERE id = $1
	`, id).Scan(&project.ID, &project.Name, &project.RootPath, &project.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Project{}, lib.NotFound("PROJECT_NOT_FOUND", "project not found")
	}
	return project, err
}

func (s Stores) CreateNote(ctx context.Context, note domain.Note) (domain.Note, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO notes (id, project_id, source, body)
		VALUES ($1, NULLIF($2, ''), $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, note.ID, note.ProjectID, note.Source, note.Body)
	if err != nil {
		return domain.Note{}, err
	}
	return note, nil
}

func (s Stores) CountNotesByProject(ctx context.Context, projectID string) (int, error) {
	return countByProject(ctx, s.db, "notes", projectID)
}

func (s Stores) ListNotesByProject(ctx context.Context, projectID string) ([]domain.Note, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, COALESCE(project_id, ''), source, body
		FROM notes
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Note
	for rows.Next() {
		var item domain.Note
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Source, &item.Body); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreateArtifact(ctx context.Context, artifact domain.Artifact) (domain.Artifact, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO artifacts (id, project_id, type, source_path, trust_level)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)
		ON CONFLICT (id) DO NOTHING
	`, artifact.ID, artifact.ProjectID, artifact.Type, artifact.SourcePath, artifact.TrustLevel)
	if err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s Stores) CountArtifactsByProject(ctx context.Context, projectID string) (int, error) {
	return countByProject(ctx, s.db, "artifacts", projectID)
}

func (s Stores) ListArtifactsByProject(ctx context.Context, projectID string) ([]domain.Artifact, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, project_id, type, COALESCE(source_path, ''), trust_level
		FROM artifacts
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Artifact
	for rows.Next() {
		var item domain.Artifact
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Type, &item.SourcePath, &item.TrustLevel); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreateDecision(ctx context.Context, decision domain.Decision) (domain.Decision, error) {
	sourceNoteIDs, _ := json.Marshal(decision.SourceNoteIDs)
	sourceArtifactIDs, _ := json.Marshal(decision.SourceArtifactIDs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO decisions (id, project_id, summary, why, source_note_ids, source_artifact_ids)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, decision.ID, decision.ProjectID, decision.Summary, decision.Why, sourceNoteIDs, sourceArtifactIDs)
	if err != nil {
		return domain.Decision{}, err
	}
	return decision, nil
}

func (s Stores) CountDecisionsByProject(ctx context.Context, projectID string) (int, error) {
	return countByProject(ctx, s.db, "decisions", projectID)
}

func (s Stores) ListDecisionsByProject(ctx context.Context, projectID string) ([]domain.Decision, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, project_id, summary, why, source_note_ids, source_artifact_ids
		FROM decisions
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Decision
	for rows.Next() {
		var item domain.Decision
		var noteIDs []byte
		var artifactIDs []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Summary, &item.Why, &noteIDs, &artifactIDs); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(noteIDs, &item.SourceNoteIDs)
		_ = json.Unmarshal(artifactIDs, &item.SourceArtifactIDs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreateOpenQuestion(ctx context.Context, question domain.OpenQuestion) (domain.OpenQuestion, error) {
	sourceNoteIDs, _ := json.Marshal(question.SourceNoteIDs)
	sourceArtifactIDs, _ := json.Marshal(question.SourceArtifactIDs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO open_questions (id, project_id, summary, source_note_ids, source_artifact_ids)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, question.ID, question.ProjectID, question.Summary, sourceNoteIDs, sourceArtifactIDs)
	if err != nil {
		return domain.OpenQuestion{}, err
	}
	return question, nil
}

func (s Stores) CountOpenQuestionsByProject(ctx context.Context, projectID string) (int, error) {
	return countByProject(ctx, s.db, "open_questions", projectID)
}

func (s Stores) ListOpenQuestionsByProject(ctx context.Context, projectID string) ([]domain.OpenQuestion, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, project_id, summary, source_note_ids, source_artifact_ids
		FROM open_questions
		WHERE project_id = $1
		ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.OpenQuestion
	for rows.Next() {
		var item domain.OpenQuestion
		var noteIDs []byte
		var artifactIDs []byte
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Summary, &noteIDs, &artifactIDs); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(noteIDs, &item.SourceNoteIDs)
		_ = json.Unmarshal(artifactIDs, &item.SourceArtifactIDs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) CreatePacket(ctx context.Context, packet domain.Packet) (domain.Packet, error) {
	decisionIDs, _ := json.Marshal(packet.DecisionIDs)
	openQuestionIDs, _ := json.Marshal(packet.OpenQuestionIDs)
	sourceArtifactIDs, _ := json.Marshal(packet.SourceArtifactIDs)
	_, err := s.db.Exec(ctx, `
		INSERT INTO packets (id, project_id, type, target, body, decision_ids, open_question_ids, source_artifact_ids)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8::jsonb)
		ON CONFLICT (id) DO NOTHING
	`, packet.ID, packet.ProjectID, packet.Type, packet.Target, packet.Body, decisionIDs, openQuestionIDs, sourceArtifactIDs)
	if err != nil {
		return domain.Packet{}, err
	}
	return packet, nil
}

func (s Stores) LatestByProject(ctx context.Context, projectID string) (domain.Packet, error) {
	var packet domain.Packet
	err := s.db.QueryRow(ctx, `
		SELECT id, project_id, type, target, body
		FROM packets
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, projectID).Scan(&packet.ID, &packet.ProjectID, &packet.Type, &packet.Target, &packet.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Packet{}, nil
	}
	return packet, err
}

func (s Stores) CreateAPIKey(ctx context.Context, key domain.APIKey) (domain.APIKey, error) {
	_, err := s.db.Exec(ctx, `
		INSERT INTO api_keys (id, name, token_hash, token_prefix)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO NOTHING
	`, key.ID, key.Name, key.TokenHash, key.TokenPrefix)
	if err != nil {
		return domain.APIKey{}, err
	}
	return key, nil
}

func (s Stores) GetByTokenHash(ctx context.Context, tokenHash string) (domain.APIKey, error) {
	var key domain.APIKey
	err := s.db.QueryRow(ctx, `
		SELECT id, name, token_hash, token_prefix, revoked_at IS NOT NULL
		FROM api_keys
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash).Scan(&key.ID, &key.Name, &key.TokenHash, &key.TokenPrefix, &key.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND", "api key not found")
	}
	return key, err
}

func (s Stores) ListAPIKeys(ctx context.Context) ([]domain.APIKey, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, token_hash, token_prefix, revoked_at IS NOT NULL
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.APIKey
	for rows.Next() {
		var item domain.APIKey
		if err := rows.Scan(&item.ID, &item.Name, &item.TokenHash, &item.TokenPrefix, &item.Revoked); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s Stores) RevokeAPIKey(ctx context.Context, keyID string) (domain.APIKey, error) {
	var key domain.APIKey
	err := s.db.QueryRow(ctx, `
		UPDATE api_keys
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE id = $1
		RETURNING id, name, token_hash, token_prefix, revoked_at IS NOT NULL
	`, keyID).Scan(&key.ID, &key.Name, &key.TokenHash, &key.TokenPrefix, &key.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.APIKey{}, lib.NotFound("API_KEY_NOT_FOUND_BY_ID", "api key not found")
	}
	return key, err
}

func countByProject(ctx context.Context, db *pgxpool.Pool, table string, projectID string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM " + table + " WHERE project_id = $1"
	err := db.QueryRow(ctx, query, projectID).Scan(&count)
	return count, err
}
