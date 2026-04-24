package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"relay/internal/contracts"
	"relay/internal/domain"
	"relay/internal/lib"
)

const packetSchemaVersionV1 = "relay.packet.v1"

func (s Service) BuildPacket(ctx context.Context, input PacketBuildInput) (PacketBuildResult, error) {
	if err := validatePacketBuildInput(input); err != nil {
		return PacketBuildResult{}, err
	}

	if input.Project == "" {
		return PacketBuildResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}

	packetType := contracts.NormalizePacketKind(input.Type)
	target := contracts.NormalizePacketTarget(input.Target)

	project, err := s.resolveProject(ctx, input.Project, "")
	if err != nil {
		return PacketBuildResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return PacketBuildResult{}, err
	}

	notes, err := s.deps.Notes.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	artifacts, err := s.deps.Artifacts.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	decisions, err := s.deps.Decisions.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}
	questions, err := s.deps.OpenQuestions.ListByProject(ctx, project.ID)
	if err != nil {
		return PacketBuildResult{}, err
	}

	taskSummary := input.TaskSummary
	if taskSummary == "" {
		taskSummary = fmt.Sprintf("resume work on %s", project.Name)
	}
	retrievalMode := "query-conditioned"
	var retrievalHits []ProjectRetrieveHit
	var selectedNotes []domain.Note
	var noteEvidence map[string]string
	var selectedDecisions []domain.Decision
	var selectedQuestions []domain.OpenQuestion
	var selectedArtifacts []artifactSelection
	if input.DisableRetrieval {
		retrievalMode = "ranking-only"
		selectedNotes, noteEvidence = selectRankedNotes(notes, taskSummary, 3)
		selectedDecisions = selectRankedDecisions(decisions, taskSummary, 3)
		selectedQuestions = selectRankedQuestions(questions, taskSummary, 3)
		selectedArtifacts = selectArtifacts(artifacts, taskSummary, 8)
	} else {
		retrievalHits = buildProjectRetrieveHits(taskSummary, 24, notes, artifacts, decisions, questions)
		selectedNotes, noteEvidence = selectRetrievedNotes(retrievalHits, notes, 3)
		selectedDecisions = selectRetrievedDecisions(retrievalHits, decisions, 3)
		selectedQuestions = selectRetrievedQuestions(retrievalHits, questions, 3)
		selectedArtifacts = selectRetrievedArtifacts(retrievalHits, artifacts, taskSummary, 8)
	}
	supportingNotes := summarizeNotes(selectedNotes, noteEvidence)
	supportingDecisions := summarizeDecisions(selectedDecisions)
	supportingQuestions := summarizeQuestions(selectedQuestions)
	supportingArtifacts := summarizeArtifacts(selectedArtifacts)
	styleCues, approvedHeuristicIDs, err := s.selectStyleCues(ctx, project.ID, input)
	if err != nil {
		return PacketBuildResult{}, err
	}
	whyIncluded := buildWhyIncluded(supportingNotes, supportingDecisions, supportingQuestions, supportingArtifacts, styleCues)
	if retrievalReason := summarizeRetrievalReason(retrievalHits, selectedNotes, selectedDecisions, selectedQuestions, selectedArtifacts); retrievalReason != "" {
		whyIncluded = append(whyIncluded, retrievalReason)
	} else if input.DisableRetrieval {
		whyIncluded = append(whyIncluded, "ranking-only packet mode kept recent and task-ranked evidence without query-conditioned retrieval")
	}
	renderedBody := buildResumeBody(packetRenderInput{
		ProjectName:         project.Name,
		TaskSummary:         taskSummary,
		TotalNotes:          len(notes),
		TotalArtifacts:      len(artifacts),
		TotalDecisions:      len(decisions),
		TotalOpenQuestions:  len(questions),
		SupportingNotes:     supportingNotes,
		SupportingDecisions: supportingDecisions,
		SupportingQuestions: supportingQuestions,
		SupportingArtifacts: supportingArtifacts,
		StyleCues:           styleCues,
		WhyIncluded:         whyIncluded,
	})

	packet := domain.Packet{
		ID:                lib.NewID("pkt"),
		ProjectID:         project.ID,
		Type:              packetType,
		Target:            target,
		Body:              renderedBody,
		DecisionIDs:       collectDecisionIDs(selectedDecisions),
		OpenQuestionIDs:   collectQuestionIDs(selectedQuestions),
		SourceArtifactIDs: collectArtifactIDsFromSelections(selectedArtifacts),
	}

	created, err := s.deps.Packets.CreatePacket(ctx, packet)
	if err != nil {
		return PacketBuildResult{}, err
	}

	result := PacketBuildResult{
		PacketID:             created.ID,
		ProjectID:            created.ProjectID,
		SchemaVersion:        packetSchemaVersionV1,
		Type:                 created.Type,
		Target:               created.Target,
		TaskSummary:          taskSummary,
		RetrievalMode:        retrievalMode,
		Body:                 created.Body,
		RenderedBody:         created.Body,
		StyleCues:            styleCues,
		SupportingNotes:      supportingNotes,
		SupportingDecisions:  supportingDecisions,
		SupportingQuestions:  supportingQuestions,
		SupportingArtifacts:  supportingArtifacts,
		WhyIncluded:          whyIncluded,
		DecisionIDs:          created.DecisionIDs,
		OpenQuestionIDs:      created.OpenQuestionIDs,
		SourceArtifactIDs:    created.SourceArtifactIDs,
		ApprovedHeuristicIDs: approvedHeuristicIDs,
		MissingContext:       collectQuestionSummaries(selectedQuestions),
	}

	if input.PersistSnapshot {
		snapshot, err := s.createPacketSnapshot(ctx, project.ID, result, input.IdempotencyKey)
		if err != nil {
			return PacketBuildResult{}, err
		}
		result.SnapshotID = snapshot.ID
	}

	return result, nil
}

type packetRenderInput struct {
	ProjectName         string
	TaskSummary         string
	TotalNotes          int
	TotalArtifacts      int
	TotalDecisions      int
	TotalOpenQuestions  int
	SupportingNotes     []PacketNote
	SupportingDecisions []PacketDecision
	SupportingQuestions []PacketQuestion
	SupportingArtifacts []PacketArtifact
	StyleCues           []PacketStyleCue
	WhyIncluded         []string
}

func buildResumeBody(input packetRenderInput) string {
	lines := []string{
		fmt.Sprintf("Project: %s", input.ProjectName),
		fmt.Sprintf("Current goal: %s", input.TaskSummary),
		fmt.Sprintf("Current state: %d note(s), %d artifact(s), %d decision(s), %d open question(s) are stored.", input.TotalNotes, input.TotalArtifacts, input.TotalDecisions, input.TotalOpenQuestions),
	}

	if len(input.SupportingNotes) > 0 {
		lines = append(lines, "Recent notes:")
		for _, note := range input.SupportingNotes {
			line := fmt.Sprintf("- %s", note.Excerpt)
			if note.Source != "" {
				line = fmt.Sprintf("- [%s] %s", note.Source, note.Excerpt)
			}
			if note.Evidence != "" {
				line += fmt.Sprintf(" (%s)", note.Evidence)
			}
			lines = append(lines, line)
		}
	}

	if len(input.SupportingDecisions) > 0 {
		lines = append(lines, "Durable decisions:")
		for _, decision := range input.SupportingDecisions {
			if decision.Why != "" {
				lines = append(lines, fmt.Sprintf("- %s: %s", decision.Summary, decision.Why))
				continue
			}
			lines = append(lines, fmt.Sprintf("- %s", decision.Summary))
		}
	}

	if len(input.SupportingQuestions) > 0 {
		lines = append(lines, "Open questions:")
		for _, question := range input.SupportingQuestions {
			lines = append(lines, fmt.Sprintf("- %s", question.Summary))
		}
	}

	if len(input.SupportingArtifacts) > 0 {
		lines = append(lines, "Trusted artifacts:")
		for _, artifact := range input.SupportingArtifacts {
			item := fmt.Sprintf("- %s", artifact.Type)
			if artifact.SourcePath != "" {
				item = fmt.Sprintf("%s (%s)", item, artifact.SourcePath)
			}
			if artifact.TrustLevel != "" {
				item = fmt.Sprintf("%s [trust=%s]", item, artifact.TrustLevel)
			}
			lines = append(lines, item)
		}
	}

	if len(input.StyleCues) > 0 {
		lines = append(lines, "Style rules to preserve:")
		for _, cue := range input.StyleCues {
			item := fmt.Sprintf("- %s", cue.HeuristicID)
			if cue.CanonicalText != "" {
				item = fmt.Sprintf("%s: %s", item, cue.CanonicalText)
			}
			lines = append(lines, item)
			if cue.WhyIncluded != "" {
				lines = append(lines, fmt.Sprintf("  why included: %s", cue.WhyIncluded))
			}
			if cue.SourceSummary != "" {
				lines = append(lines, fmt.Sprintf("  evidence: %s", cue.SourceSummary))
			}
			if len(cue.SourceRefs) > 0 {
				lines = append(lines, fmt.Sprintf("  source refs: %s", strings.Join(cue.SourceRefs, ", ")))
			}
		}
	}

	if len(input.WhyIncluded) > 0 {
		lines = append(lines, "Why this context was included:")
		for _, reason := range input.WhyIncluded {
			lines = append(lines, fmt.Sprintf("- %s", reason))
		}
	}

	lines = append(lines, "Next step: inspect the durable decisions, open questions, cited artifacts, and style rules before making new assumptions.")
	return strings.Join(lines, "\n")
}

func collectDecisionIDs(items []domain.Decision) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectQuestionIDs(items []domain.OpenQuestion) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectArtifactIDs(items []domain.Artifact) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func collectArtifactIDsFromSelections(items []artifactSelection) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.Artifact.ID)
	}
	return ids
}

func collectQuestionSummaries(items []domain.OpenQuestion) []string {
	summaries := make([]string, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, item.Summary)
	}
	return summaries
}

func (s Service) selectStyleCues(ctx context.Context, projectID string, input PacketBuildInput) ([]PacketStyleCue, []string, error) {
	if input.DisableStyleCues || s.deps.ApprovedHeuristics == nil {
		return nil, nil, nil
	}
	heuristics, err := s.deps.ApprovedHeuristics.ListApprovedHeuristicsByProject(ctx, projectID, input.Workflow, input.ArtifactType, 10)
	if err != nil {
		return nil, nil, err
	}
	heuristics = selectMostSpecificHeuristics(heuristics, 3)
	cues := make([]PacketStyleCue, 0, len(heuristics))
	ids := make([]string, 0, len(heuristics))
	for _, heuristic := range heuristics {
		ids = append(ids, heuristic.ID)
		scope := "this project"
		if heuristic.Workflow != "" {
			scope = "same workflow"
		}
		if heuristic.ArtifactType != "" {
			scope = "same workflow and artifact type"
		}
		cues = append(cues, PacketStyleCue{
			HeuristicID:   heuristic.ID,
			HeuristicKey:  heuristic.HeuristicKey,
			CanonicalText: heuristic.CanonicalText,
			WhySelected:   "selected for " + scope,
			WhyIncluded:   "approved heuristic matched the current project selectors: " + scope,
			SourceSummary: fmt.Sprintf("%d source trace(s)", len(heuristic.SourceTraceIDs)),
			SourceRefs:    heuristic.SourceRefs,
		})
	}
	return cues, ids, nil
}

func selectMostSpecificHeuristics(items []domain.ApprovedHeuristic, limit int) []domain.ApprovedHeuristic {
	if len(items) <= 1 {
		return items
	}
	selected := make([]domain.ApprovedHeuristic, 0, minInt(len(items), limit))
	for _, item := range items {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, item)
	}
	return selected
}

func (s Service) createPacketSnapshot(ctx context.Context, projectID string, result PacketBuildResult, idempotencyKey string) (domain.PacketSnapshot, error) {
	if s.deps.PacketSnapshots == nil {
		return domain.PacketSnapshot{}, lib.Misconfigured("packet snapshot store is required")
	}
	idempotencyPayload := result
	idempotencyPayload.PacketID = ""
	idempotencyPayload.SnapshotID = ""
	requestHash := normalizedRequestHash(idempotencyPayload)
	if lookup, err := s.lookupIdempotency(ctx, "packet_snapshot", projectID, idempotencyKey, requestHash); err != nil {
		return domain.PacketSnapshot{}, err
	} else if lookup.found {
		return s.deps.PacketSnapshots.GetPacketSnapshot(ctx, lookup.responseID)
	}

	snapshotID := lib.NewID("psnap")
	if idempotencyKey != "" {
		snapshotID = lib.StableID("psnap", projectID+":"+idempotencyKey)
	}
	styleCuesJSON, _ := json.Marshal(result.StyleCues)
	supportingNotesJSON, _ := json.Marshal(result.SupportingNotes)
	supportingDecisionsJSON, _ := json.Marshal(result.SupportingDecisions)
	supportingQuestionsJSON, _ := json.Marshal(result.SupportingQuestions)
	supportingArtifactsJSON, _ := json.Marshal(result.SupportingArtifacts)
	snapshot, err := s.deps.PacketSnapshots.CreatePacketSnapshot(ctx, domain.PacketSnapshot{
		ID:                   snapshotID,
		ProjectID:            projectID,
		PacketKind:           result.Type,
		Target:               result.Target,
		SchemaVersion:        result.SchemaVersion,
		TaskSummary:          result.TaskSummary,
		RenderedBody:         result.RenderedBody,
		StyleCues:            styleCuesJSON,
		SupportingNotes:      supportingNotesJSON,
		SupportingDecisions:  supportingDecisionsJSON,
		SupportingQuestions:  supportingQuestionsJSON,
		SupportingArtifacts:  supportingArtifactsJSON,
		WhyIncluded:          result.WhyIncluded,
		ApprovedHeuristicIDs: result.ApprovedHeuristicIDs,
		DecisionIDs:          result.DecisionIDs,
		OpenQuestionIDs:      result.OpenQuestionIDs,
		SourceArtifactIDs:    result.SourceArtifactIDs,
		MissingContext:       result.MissingContext,
	})
	if err != nil {
		return domain.PacketSnapshot{}, err
	}
	if err := s.recordIdempotency(ctx, "packet_snapshot", projectID, idempotencyKey, requestHash, "packet_snapshot", snapshot.ID); err != nil {
		return domain.PacketSnapshot{}, err
	}
	return snapshot, nil
}

func summarizeNotes(items []domain.Note, evidenceByID map[string]string) []PacketNote {
	summaries := make([]PacketNote, 0, len(items))
	for _, item := range items {
		evidence := "recent raw capture"
		if evidenceByID != nil {
			if override, ok := evidenceByID[item.ID]; ok && strings.TrimSpace(override) != "" {
				evidence = override
			}
		}
		summaries = append(summaries, PacketNote{
			NoteID:   item.ID,
			Source:   item.Source,
			Excerpt:  summarizeText(item.Body, 220),
			Evidence: evidence,
		})
	}
	return summaries
}

func summarizeDecisions(items []domain.Decision) []PacketDecision {
	summaries := make([]PacketDecision, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, PacketDecision{
			DecisionID: item.ID,
			Summary:    item.Summary,
			Why:        item.Why,
		})
	}
	return summaries
}

func summarizeQuestions(items []domain.OpenQuestion) []PacketQuestion {
	summaries := make([]PacketQuestion, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, PacketQuestion{
			QuestionID: item.ID,
			Summary:    item.Summary,
		})
	}
	return summaries
}

func summarizeArtifacts(items []artifactSelection) []PacketArtifact {
	summaries := make([]PacketArtifact, 0, len(items))
	for _, item := range items {
		whyIncluded := item.WhyIncluded
		if whyIncluded == "" {
			whyIncluded = "trusted artifact referenced by project memory"
		}
		summaries = append(summaries, PacketArtifact{
			ArtifactID:  item.Artifact.ID,
			Type:        item.Artifact.Type,
			SourcePath:  item.Artifact.SourcePath,
			TrustLevel:  item.Artifact.TrustLevel,
			WhyIncluded: whyIncluded,
		})
	}
	return summaries
}

func buildWhyIncluded(notes []PacketNote, decisions []PacketDecision, questions []PacketQuestion, artifacts []PacketArtifact, styleCues []PacketStyleCue) []string {
	reasons := make([]string, 0, 5)
	if len(notes) > 0 {
		reasons = append(reasons, "recent captured notes preserve raw context that has not yet been promoted into canonical decisions")
	}
	if len(decisions) > 0 {
		reasons = append(reasons, "durable decisions anchor settled choices so the next agent does not need to infer them again")
	}
	if len(questions) > 0 {
		reasons = append(reasons, "open questions surface unresolved blockers before the next agent commits to a path")
	}
	if len(artifacts) > 0 {
		reasons = append(reasons, "trusted artifacts point to concrete files or deliverables worth inspecting next")
	}
	if len(styleCues) > 0 {
		reasons = append(reasons, "approved heuristics matched the current workflow and artifact selectors and should shape continuation style")
	}
	return reasons
}

func summarizeText(input string, limit int) string {
	compact := strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
	if len(compact) <= limit {
		return compact
	}
	if limit <= 3 {
		return compact[:limit]
	}
	return compact[:limit-3] + "..."
}

func limitNotes(items []domain.Note, limit int) []domain.Note {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func limitArtifacts(items []domain.Artifact, limit int) []domain.Artifact {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

type artifactSelection struct {
	Artifact    domain.Artifact
	WhyIncluded string
	Score       int
	Index       int
}

func selectArtifacts(items []domain.Artifact, taskSummary string, limit int) []artifactSelection {
	if len(items) == 0 || limit <= 0 {
		return nil
	}

	taskTokens := tokenizeRankingText(taskSummary)
	if len(taskTokens) == 0 {
		return selectRecentArtifacts(items, limit)
	}

	selections := make([]artifactSelection, 0, len(items))
	taskSummaryLower := strings.ToLower(taskSummary)
	for index, item := range items {
		score, whyIncluded := scoreArtifactForTask(item, taskTokens, taskSummaryLower)
		selections = append(selections, artifactSelection{
			Artifact:    item,
			WhyIncluded: whyIncluded,
			Score:       score,
			Index:       index,
		})
	}

	sort.SliceStable(selections, func(i, j int) bool {
		if selections[i].Score != selections[j].Score {
			return selections[i].Score > selections[j].Score
		}
		return selections[i].Index > selections[j].Index
	})
	if len(selections) > limit {
		selections = selections[:limit]
	}
	return selections
}

func selectRecentArtifacts(items []domain.Artifact, limit int) []artifactSelection {
	recent := limitArtifacts(items, limit)
	selections := make([]artifactSelection, 0, len(recent))
	offset := len(items) - len(recent)
	for index, item := range recent {
		selections = append(selections, artifactSelection{
			Artifact:    item,
			WhyIncluded: "recent trusted artifact retained as fallback evidence",
			Index:       offset + index,
		})
	}
	return selections
}

func scoreArtifactForTask(item domain.Artifact, taskTokens []string, taskSummaryLower string) (int, string) {
	candidate := strings.ToLower(strings.TrimSpace(item.Type + " " + item.SourcePath))
	if candidate == "" {
		return 0, "recent trusted artifact retained as fallback evidence"
	}

	score := 0
	reasons := make([]string, 0, 3)
	if item.Type != "" && strings.Contains(taskSummaryLower, strings.ToLower(item.Type)) {
		score += 6
		reasons = append(reasons, "artifact type matched the current task summary")
	}
	if item.SourcePath != "" {
		sourcePathLower := strings.ToLower(item.SourcePath)
		if strings.Contains(taskSummaryLower, sourcePathLower) {
			score += 10
			reasons = append(reasons, "source path matched the current task summary")
		}
	}

	artifactTokens := tokenizeRankingText(candidate)
	artifactTokenSet := make(map[string]struct{}, len(artifactTokens))
	for _, token := range artifactTokens {
		artifactTokenSet[token] = struct{}{}
	}

	matchedTokens := make([]string, 0, len(taskTokens))
	for _, token := range taskTokens {
		if _, ok := artifactTokenSet[token]; ok {
			score += 3
			if len(token) >= 5 {
				score++
			}
			matchedTokens = appendUniqueString(matchedTokens, token)
			continue
		}
		if len(token) >= 4 && strings.Contains(candidate, token) {
			score++
			matchedTokens = appendUniqueString(matchedTokens, token)
		}
	}
	if len(matchedTokens) > 0 {
		reasons = append(reasons, "task-summary tokens overlapped: "+strings.Join(matchedTokens, ", "))
	}
	if len(reasons) == 0 {
		return score, "recent trusted artifact retained as fallback evidence"
	}
	return score, strings.Join(reasons, "; ")
}

func tokenizeRankingText(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	parts := strings.FieldsFunc(strings.ToLower(input), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	tokens := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if len(part) <= 2 {
			continue
		}
		if _, skip := artifactRankingStopWords[part]; skip {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		tokens = append(tokens, part)
	}
	return tokens
}

func appendUniqueString(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

var artifactRankingStopWords = map[string]struct{}{
	"agent":    {},
	"and":      {},
	"before":   {},
	"changes":  {},
	"checking": {},
	"continue": {},
	"current":  {},
	"docs":     {},
	"handoff":  {},
	"internal": {},
	"model":    {},
	"project":  {},
	"relay":    {},
	"resume":   {},
	"same":     {},
	"session":  {},
	"services": {},
	"task":     {},
	"user":     {},
	"wrapper":  {},
	"work":     {},
}

func limitDecisions(items []domain.Decision, limit int) []domain.Decision {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func limitQuestions(items []domain.OpenQuestion, limit int) []domain.OpenQuestion {
	if len(items) <= limit {
		return items
	}
	return items[len(items)-limit:]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
