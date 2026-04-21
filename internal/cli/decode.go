package cli

import "relay/internal/services"

func decodeCaptureInput(payload map[string]any) services.CaptureInput {
	return services.CaptureInput{
		Project:        stringField(payload, "project"),
		RepoPath:       stringField(payload, "repo_path"),
		HandoffPath:    stringField(payload, "handoff_path"),
		DesignPath:     stringField(payload, "design_path"),
		Note:           stringField(payload, "note"),
		Source:         stringField(payload, "source"),
		Body:           stringField(payload, "body"),
		IdempotencyKey: stringField(payload, "idempotency_key"),
	}
}

func decodePromoteInput(payload map[string]any) services.PromoteInput {
	return services.PromoteInput{
		Project:           stringField(payload, "project"),
		Kind:              stringField(payload, "kind"),
		Summary:           stringField(payload, "summary"),
		Reason:            stringField(payload, "reason"),
		SourceNoteIDs:     stringSliceField(payload, "source_note_ids"),
		SourceArtifactIDs: stringSliceField(payload, "source_artifact_ids"),
		IdempotencyKey:    stringField(payload, "idempotency_key"),
	}
}

func decodePacketBuildInput(payload map[string]any) services.PacketBuildInput {
	return services.PacketBuildInput{
		Project: stringField(payload, "project"),
		Type:    stringField(payload, "type"),
		Target:  stringField(payload, "target"),
	}
}

func decodeShowInput(payload map[string]any) services.ShowInput {
	return services.ShowInput{
		Project:   stringField(payload, "project"),
		ProjectID: stringField(payload, "project_id"),
	}
}

func stringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}

func stringSliceField(payload map[string]any, key string) []string {
	value, ok := payload[key]
	if !ok {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
