package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"relay/internal/domain"
	"relay/internal/lib"
)

func (s Stores) MakePacketSnapshotPublic(ctx context.Context, snapshotID string, publicToken string, ogImagePath string) (domain.PacketSnapshot, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE packet_snapshots
		SET public_readable = TRUE,
		    public_token = $2,
		    og_image_path = $3
		WHERE id = $1
		RETURNING id, project_id, packet_kind, target, schema_version, task_summary,
		          rendered_body, style_cues, supporting_notes, supporting_decisions,
		          supporting_questions, supporting_artifacts, why_included,
		          approved_heuristic_ids, decision_ids, open_question_ids,
		          source_artifact_ids, missing_context, public_readable,
		          COALESCE(public_token, ''), og_image_path, created_at
	`, snapshotID, publicToken, ogImagePath)
	snapshot, err := scanPacketSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPacketSnapshotNotFound(err) {
			return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
		}
		return domain.PacketSnapshot{}, err
	}
	return snapshot, nil
}

func (s Stores) RevokePacketSnapshotPublic(ctx context.Context, snapshotID string) (domain.PacketSnapshot, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE packet_snapshots
		SET public_readable = FALSE
		WHERE id = $1
		RETURNING id, project_id, packet_kind, target, schema_version, task_summary,
		          rendered_body, style_cues, supporting_notes, supporting_decisions,
		          supporting_questions, supporting_artifacts, why_included,
		          approved_heuristic_ids, decision_ids, open_question_ids,
		          source_artifact_ids, missing_context, public_readable,
		          COALESCE(public_token, ''), og_image_path, created_at
	`, snapshotID)
	snapshot, err := scanPacketSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPacketSnapshotNotFound(err) {
			return domain.PacketSnapshot{}, lib.NotFound("PACKET_SNAPSHOT_NOT_FOUND", "packet snapshot not found")
		}
		return domain.PacketSnapshot{}, err
	}
	return snapshot, nil
}

func (s Stores) GetPacketSnapshotByPublicToken(ctx context.Context, token string) (domain.PacketSnapshot, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, project_id, packet_kind, target, schema_version, task_summary,
		       rendered_body, style_cues, supporting_notes, supporting_decisions,
		       supporting_questions, supporting_artifacts, why_included,
		       approved_heuristic_ids, decision_ids, open_question_ids,
		       source_artifact_ids, missing_context, public_readable,
		       COALESCE(public_token, ''), og_image_path, created_at
		FROM packet_snapshots
		WHERE public_token = $1
		  AND public_readable = TRUE
	`, token)
	snapshot, err := scanPacketSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isPacketSnapshotNotFound(err) {
			return domain.PacketSnapshot{}, lib.NotFound("PUBLIC_SNAPSHOT_NOT_FOUND", "public snapshot not found")
		}
		return domain.PacketSnapshot{}, err
	}
	return snapshot, nil
}

// isPacketSnapshotNotFound matches the lib.NotFound the existing
// scanPacketSnapshot helper returns when pgx.ErrNoRows fires through
// the RETURNING path.
func isPacketSnapshotNotFound(err error) bool {
	var appErr lib.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "PACKET_SNAPSHOT_NOT_FOUND"
	}
	return false
}
