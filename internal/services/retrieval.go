package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"relay/internal/domain"
	"relay/internal/lib"
)

const (
	defaultProjectRetrieveLimit = 12
	maxProjectRetrieveLimit     = 50
)

type retrievalCandidate struct {
	Hit   ProjectRetrieveHit
	Index int
}

func (s Service) ProjectRetrieve(ctx context.Context, input ProjectRetrieveInput) (ProjectRetrieveResult, error) {
	if input.Project == "" && input.ProjectID == "" {
		return ProjectRetrieveResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "project")
	}
	if strings.TrimSpace(input.Query) == "" {
		return ProjectRetrieveResult{}, lib.MissingFields("MISSING_REQUIRED_FIELDS", "query")
	}

	project, err := s.resolveProject(ctx, input.Project, input.ProjectID)
	if err != nil {
		return ProjectRetrieveResult{}, err
	}
	if err := s.enforceProjectAccess(ctx, project.ID); err != nil {
		return ProjectRetrieveResult{}, err
	}

	notes, err := s.deps.Notes.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectRetrieveResult{}, err
	}
	artifacts, err := s.deps.Artifacts.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectRetrieveResult{}, err
	}
	decisions, err := s.deps.Decisions.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectRetrieveResult{}, err
	}
	questions, err := s.deps.OpenQuestions.ListByProject(ctx, project.ID)
	if err != nil {
		return ProjectRetrieveResult{}, err
	}

	return ProjectRetrieveResult{
		ProjectID: project.ID,
		Query:     strings.TrimSpace(input.Query),
		Hits:      buildProjectRetrieveHits(input.Query, input.Limit, notes, artifacts, decisions, questions),
	}, nil
}

func buildProjectRetrieveHits(query string, limit int, notes []domain.Note, artifacts []domain.Artifact, decisions []domain.Decision, questions []domain.OpenQuestion) []ProjectRetrieveHit {
	taskTokens := tokenizeRankingText(query)
	if len(taskTokens) == 0 {
		return nil
	}

	taskSummaryLower := strings.ToLower(strings.TrimSpace(query))
	noteCandidates, noteScores := buildNoteRetrievalCandidates(notes, taskTokens, taskSummaryLower)
	artifactCandidates, artifactScores := buildArtifactRetrievalCandidates(artifacts, taskTokens, taskSummaryLower)
	decisionCandidates := buildDecisionRetrievalCandidates(decisions, taskTokens, taskSummaryLower, noteScores, artifactScores)
	questionCandidates := buildQuestionRetrievalCandidates(questions, taskTokens, taskSummaryLower, noteScores, artifactScores)

	candidates := make([]retrievalCandidate, 0, len(noteCandidates)+len(artifactCandidates)+len(decisionCandidates)+len(questionCandidates))
	candidates = append(candidates, noteCandidates...)
	candidates = append(candidates, artifactCandidates...)
	candidates = append(candidates, decisionCandidates...)
	candidates = append(candidates, questionCandidates...)

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Hit.Score != candidates[j].Hit.Score {
			return candidates[i].Hit.Score > candidates[j].Hit.Score
		}
		return candidates[i].Index > candidates[j].Index
	})

	normalizedLimit := normalizeProjectRetrieveLimit(limit)
	if len(candidates) > normalizedLimit {
		candidates = candidates[:normalizedLimit]
	}

	hits := make([]ProjectRetrieveHit, 0, len(candidates))
	for _, candidate := range candidates {
		hits = append(hits, candidate.Hit)
	}
	return hits
}

func normalizeProjectRetrieveLimit(limit int) int {
	if limit <= 0 {
		return defaultProjectRetrieveLimit
	}
	if limit > maxProjectRetrieveLimit {
		return maxProjectRetrieveLimit
	}
	return limit
}

func buildNoteRetrievalCandidates(items []domain.Note, taskTokens []string, taskSummaryLower string) ([]retrievalCandidate, map[string]int) {
	candidates := make([]retrievalCandidate, 0, len(items))
	scores := make(map[string]int, len(items))
	for index, item := range items {
		score, whyIncluded := scoreNoteForTask(item, taskTokens, taskSummaryLower)
		if score <= 0 {
			continue
		}
		scores[item.ID] = score
		candidates = append(candidates, retrievalCandidate{
			Hit: ProjectRetrieveHit{
				Kind:        "note",
				ID:          item.ID,
				Score:       score,
				Title:       summarizeText(item.Body, 120),
				Excerpt:     summarizeText(item.Body, 220),
				Source:      item.Source,
				WhyIncluded: whyIncluded,
			},
			Index: index,
		})
	}
	return candidates, scores
}

func buildArtifactRetrievalCandidates(items []domain.Artifact, taskTokens []string, taskSummaryLower string) ([]retrievalCandidate, map[string]int) {
	candidates := make([]retrievalCandidate, 0, len(items))
	scores := make(map[string]int, len(items))
	for index, item := range items {
		score, whyIncluded := scoreArtifactForTask(item, taskTokens, taskSummaryLower)
		if score <= 0 {
			continue
		}
		scores[item.ID] = score
		candidates = append(candidates, retrievalCandidate{
			Hit: ProjectRetrieveHit{
				Kind:        "artifact",
				ID:          item.ID,
				Score:       score,
				Title:       item.Type,
				SourcePath:  item.SourcePath,
				TrustLevel:  item.TrustLevel,
				WhyIncluded: whyIncluded,
			},
			Index: index,
		})
	}
	return candidates, scores
}

func buildDecisionRetrievalCandidates(items []domain.Decision, taskTokens []string, taskSummaryLower string, noteScores map[string]int, artifactScores map[string]int) []retrievalCandidate {
	candidates := make([]retrievalCandidate, 0, len(items))
	for index, item := range items {
		score, whyIncluded, sourceRefIDs := scoreDecisionForTask(item, taskTokens, taskSummaryLower, noteScores, artifactScores)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, retrievalCandidate{
			Hit: ProjectRetrieveHit{
				Kind:         "decision",
				ID:           item.ID,
				Score:        score,
				Title:        item.Summary,
				Excerpt:      summarizeText(strings.TrimSpace(item.Summary+" "+item.Why), 220),
				WhyIncluded:  whyIncluded,
				SourceRefIDs: sourceRefIDs,
			},
			Index: index,
		})
	}
	return candidates
}

func buildQuestionRetrievalCandidates(items []domain.OpenQuestion, taskTokens []string, taskSummaryLower string, noteScores map[string]int, artifactScores map[string]int) []retrievalCandidate {
	candidates := make([]retrievalCandidate, 0, len(items))
	for index, item := range items {
		score, whyIncluded, sourceRefIDs := scoreQuestionForTask(item, taskTokens, taskSummaryLower, noteScores, artifactScores)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, retrievalCandidate{
			Hit: ProjectRetrieveHit{
				Kind:         "open_question",
				ID:           item.ID,
				Score:        score,
				Title:        item.Summary,
				Excerpt:      summarizeText(item.Summary, 220),
				WhyIncluded:  whyIncluded,
				SourceRefIDs: sourceRefIDs,
			},
			Index: index,
		})
	}
	return candidates
}

func scoreNoteForTask(item domain.Note, taskTokens []string, taskSummaryLower string) (int, string) {
	candidate := strings.ToLower(strings.TrimSpace(item.Source + " " + item.Body))
	if candidate == "" {
		return 0, "recent raw capture retained as fallback evidence"
	}

	score := 0
	reasons := make([]string, 0, 2)
	if item.Source != "" && strings.Contains(taskSummaryLower, strings.ToLower(item.Source)) {
		score += 4
		reasons = append(reasons, "note source matched the current task summary")
	}
	overlapScore, matchedTokens := scoreCandidateTokenOverlap(candidate, taskTokens)
	score += overlapScore
	if len(matchedTokens) > 0 {
		reasons = append(reasons, "query tokens overlapped: "+strings.Join(matchedTokens, ", "))
	}
	if len(reasons) == 0 {
		return score, "recent raw capture retained as fallback evidence"
	}
	return score, strings.Join(reasons, "; ")
}

func scoreDecisionForTask(item domain.Decision, taskTokens []string, taskSummaryLower string, noteScores map[string]int, artifactScores map[string]int) (int, string, []string) {
	score := 0
	reasons := make([]string, 0, 3)
	candidate := strings.ToLower(strings.TrimSpace(item.Summary + " " + item.Why))
	if item.Summary != "" && strings.Contains(taskSummaryLower, strings.ToLower(item.Summary)) {
		score += 7
		reasons = append(reasons, "decision summary matched the current task summary")
	}
	overlapScore, matchedTokens := scoreCandidateTokenOverlap(candidate, taskTokens)
	score += overlapScore
	if len(matchedTokens) > 0 {
		reasons = append(reasons, "query tokens overlapped: "+strings.Join(matchedTokens, ", "))
	}
	graphBoost, sourceRefIDs := scoreGraphSupport(item.SourceNoteIDs, item.SourceArtifactIDs, noteScores, artifactScores)
	score += graphBoost
	if graphBoost > 0 {
		reasons = append(reasons, "connected canonical evidence also matched the query")
	}
	if len(reasons) == 0 {
		return score, "", sourceRefIDs
	}
	return score, strings.Join(reasons, "; "), sourceRefIDs
}

func scoreQuestionForTask(item domain.OpenQuestion, taskTokens []string, taskSummaryLower string, noteScores map[string]int, artifactScores map[string]int) (int, string, []string) {
	score := 0
	reasons := make([]string, 0, 3)
	candidate := strings.ToLower(strings.TrimSpace(item.Summary))
	if item.Summary != "" && strings.Contains(taskSummaryLower, strings.ToLower(item.Summary)) {
		score += 6
		reasons = append(reasons, "open question summary matched the current task summary")
	}
	overlapScore, matchedTokens := scoreCandidateTokenOverlap(candidate, taskTokens)
	score += overlapScore
	if len(matchedTokens) > 0 {
		reasons = append(reasons, "query tokens overlapped: "+strings.Join(matchedTokens, ", "))
	}
	graphBoost, sourceRefIDs := scoreGraphSupport(item.SourceNoteIDs, item.SourceArtifactIDs, noteScores, artifactScores)
	score += graphBoost
	if graphBoost > 0 {
		reasons = append(reasons, "connected canonical evidence also matched the query")
	}
	if len(reasons) == 0 {
		return score, "", sourceRefIDs
	}
	return score, strings.Join(reasons, "; "), sourceRefIDs
}

func scoreCandidateTokenOverlap(candidate string, taskTokens []string) (int, []string) {
	if candidate == "" {
		return 0, nil
	}
	score := 0
	matchedTokens := make([]string, 0, len(taskTokens))
	for _, token := range taskTokens {
		if !strings.Contains(candidate, token) {
			continue
		}
		score += 2
		if len(token) >= 5 {
			score++
		}
		matchedTokens = appendUniqueString(matchedTokens, token)
	}
	return score, matchedTokens
}

func scoreGraphSupport(sourceNoteIDs []string, sourceArtifactIDs []string, noteScores map[string]int, artifactScores map[string]int) (int, []string) {
	boost := 0
	sourceRefIDs := make([]string, 0, len(sourceNoteIDs)+len(sourceArtifactIDs))
	for _, noteID := range sourceNoteIDs {
		score := noteScores[noteID]
		if score <= 0 {
			continue
		}
		boost += 2 + minInt(4, score/3)
		sourceRefIDs = appendUniqueString(sourceRefIDs, noteID)
	}
	for _, artifactID := range sourceArtifactIDs {
		score := artifactScores[artifactID]
		if score <= 0 {
			continue
		}
		boost += 2 + minInt(4, score/3)
		sourceRefIDs = appendUniqueString(sourceRefIDs, artifactID)
	}
	return boost, sourceRefIDs
}

func selectRetrievedNotes(hits []ProjectRetrieveHit, items []domain.Note, limit int) ([]domain.Note, map[string]string) {
	selected := make([]domain.Note, 0, minInt(limit, len(items)))
	evidenceByID := make(map[string]string, limit)
	selectedIDs := make(map[string]struct{}, limit)
	byID := make(map[string]domain.Note, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	for _, hit := range hits {
		if len(selected) >= limit || hit.Kind != "note" {
			continue
		}
		item, ok := byID[hit.ID]
		if !ok {
			continue
		}
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
		evidenceByID[item.ID] = "retrieved against the current task summary: " + hit.WhyIncluded
	}

	for index := len(items) - 1; index >= 0 && len(selected) < limit; index-- {
		item := items[index]
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
	}

	return selected, evidenceByID
}

func selectRetrievedDecisions(hits []ProjectRetrieveHit, items []domain.Decision, limit int) []domain.Decision {
	selected := make([]domain.Decision, 0, minInt(limit, len(items)))
	selectedIDs := make(map[string]struct{}, limit)
	byID := make(map[string]domain.Decision, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	for _, hit := range hits {
		if len(selected) >= limit || hit.Kind != "decision" {
			continue
		}
		item, ok := byID[hit.ID]
		if !ok {
			continue
		}
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
	}

	for index := len(items) - 1; index >= 0 && len(selected) < limit; index-- {
		item := items[index]
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
	}

	return selected
}

func selectRetrievedQuestions(hits []ProjectRetrieveHit, items []domain.OpenQuestion, limit int) []domain.OpenQuestion {
	selected := make([]domain.OpenQuestion, 0, minInt(limit, len(items)))
	selectedIDs := make(map[string]struct{}, limit)
	byID := make(map[string]domain.OpenQuestion, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	for _, hit := range hits {
		if len(selected) >= limit || hit.Kind != "open_question" {
			continue
		}
		item, ok := byID[hit.ID]
		if !ok {
			continue
		}
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
	}

	for index := len(items) - 1; index >= 0 && len(selected) < limit; index-- {
		item := items[index]
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, item)
		selectedIDs[item.ID] = struct{}{}
	}

	return selected
}

func selectRetrievedArtifacts(hits []ProjectRetrieveHit, items []domain.Artifact, taskSummary string, limit int) []artifactSelection {
	selected := make([]artifactSelection, 0, minInt(limit, len(items)))
	selectedIDs := make(map[string]struct{}, limit)
	byID := make(map[string]domain.Artifact, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}

	for _, hit := range hits {
		if len(selected) >= limit || hit.Kind != "artifact" {
			continue
		}
		item, ok := byID[hit.ID]
		if !ok {
			continue
		}
		if _, exists := selectedIDs[item.ID]; exists {
			continue
		}
		selected = append(selected, artifactSelection{
			Artifact:    item,
			WhyIncluded: hit.WhyIncluded,
			Score:       hit.Score,
		})
		selectedIDs[item.ID] = struct{}{}
	}

	for _, fallback := range selectArtifacts(items, taskSummary, limit) {
		if len(selected) >= limit {
			break
		}
		if _, exists := selectedIDs[fallback.Artifact.ID]; exists {
			continue
		}
		selected = append(selected, fallback)
		selectedIDs[fallback.Artifact.ID] = struct{}{}
	}

	return selected
}

func summarizeRetrievalReason(hits []ProjectRetrieveHit, notes []domain.Note, decisions []domain.Decision, questions []domain.OpenQuestion, artifacts []artifactSelection) string {
	if len(hits) == 0 {
		return ""
	}
	selectedKinds := map[string]int{}
	for _, item := range notes {
		for _, hit := range hits {
			if hit.Kind == "note" && hit.ID == item.ID {
				selectedKinds["note"]++
				break
			}
		}
	}
	for _, item := range decisions {
		for _, hit := range hits {
			if hit.Kind == "decision" && hit.ID == item.ID {
				selectedKinds["decision"]++
				break
			}
		}
	}
	for _, item := range questions {
		for _, hit := range hits {
			if hit.Kind == "open_question" && hit.ID == item.ID {
				selectedKinds["open_question"]++
				break
			}
		}
	}
	for _, item := range artifacts {
		for _, hit := range hits {
			if hit.Kind == "artifact" && hit.ID == item.Artifact.ID {
				selectedKinds["artifact"]++
				break
			}
		}
	}

	total := selectedKinds["note"] + selectedKinds["decision"] + selectedKinds["open_question"] + selectedKinds["artifact"]
	if total == 0 {
		return ""
	}
	return fmt.Sprintf(
		"query-conditioned retrieval matched the current task summary across %d note(s), %d decision(s), %d open question(s), and %d artifact(s)",
		selectedKinds["note"],
		selectedKinds["decision"],
		selectedKinds["open_question"],
		selectedKinds["artifact"],
	)
}
