package services

import (
	"strings"

	"relay/internal/domain"
)

const (
	projectGraphEdgePossibleSupport = "possible_support"
	projectGraphEdgePossibleAnswer  = "possible_answer"
	inferredEdgeStatusCandidate     = "candidate"
	minInferredEdgeScore            = 5
)

func buildInferredSupportEdges(decisions []domain.Decision, questions []domain.OpenQuestion, notes []domain.Note, artifacts []domain.Artifact) []ProjectGraphEdge {
	edges := make([]ProjectGraphEdge, 0, len(decisions)*(len(notes)+len(artifacts))+len(questions)*(len(notes)+len(artifacts)))
	for _, decision := range decisions {
		edges = append(edges, inferSupportEdgesForDecision(decision, notes, artifacts)...)
	}
	for _, question := range questions {
		edges = append(edges, inferSupportEdgesForQuestion(question, notes, artifacts)...)
	}
	return edges
}

func inferSupportEdgesForDecision(decision domain.Decision, notes []domain.Note, artifacts []domain.Artifact) []ProjectGraphEdge {
	return inferSupportEdges(
		decision.ID,
		decision.Summary+" "+decision.Why,
		decision.SourceNoteIDs,
		decision.SourceArtifactIDs,
		projectGraphEdgePossibleSupport,
		notes,
		artifacts,
	)
}

func inferSupportEdgesForQuestion(question domain.OpenQuestion, notes []domain.Note, artifacts []domain.Artifact) []ProjectGraphEdge {
	return inferSupportEdges(
		question.ID,
		question.Summary,
		question.SourceNoteIDs,
		question.SourceArtifactIDs,
		projectGraphEdgePossibleAnswer,
		notes,
		artifacts,
	)
}

func inferSupportEdges(fromID string, sourceText string, canonicalNoteIDs []string, canonicalArtifactIDs []string, edgeType string, notes []domain.Note, artifacts []domain.Artifact) []ProjectGraphEdge {
	sourceTokens := tokenizeRankingText(sourceText)
	if len(sourceTokens) == 0 {
		return nil
	}

	canonicalTargets := make(map[string]struct{}, len(canonicalNoteIDs)+len(canonicalArtifactIDs))
	for _, noteID := range canonicalNoteIDs {
		canonicalTargets[noteID] = struct{}{}
	}
	for _, artifactID := range canonicalArtifactIDs {
		canonicalTargets[artifactID] = struct{}{}
	}

	edges := make([]ProjectGraphEdge, 0, len(notes)+len(artifacts))
	for _, note := range notes {
		if _, isCanonical := canonicalTargets[note.ID]; isCanonical {
			continue
		}
		score, whyIncluded := inferSupportEdgeScore(sourceTokens, note.Source+" "+note.Body)
		if score < minInferredEdgeScore {
			continue
		}
		edges = append(edges, ProjectGraphEdge{
			Type:        edgeType,
			From:        fromID,
			To:          note.ID,
			Status:      inferredEdgeStatusCandidate,
			Score:       score,
			WhyIncluded: whyIncluded,
		})
	}

	for _, artifact := range artifacts {
		if _, isCanonical := canonicalTargets[artifact.ID]; isCanonical {
			continue
		}
		score, whyIncluded := inferSupportEdgeScore(sourceTokens, artifact.Type+" "+artifact.SourcePath)
		if score < minInferredEdgeScore {
			continue
		}
		edges = append(edges, ProjectGraphEdge{
			Type:        edgeType,
			From:        fromID,
			To:          artifact.ID,
			Status:      inferredEdgeStatusCandidate,
			Score:       score,
			WhyIncluded: whyIncluded,
		})
	}

	return edges
}

func inferSupportEdgeScore(sourceTokens []string, candidateText string) (int, string) {
	candidate := strings.ToLower(strings.TrimSpace(candidateText))
	if candidate == "" {
		return 0, ""
	}
	score, matchedTokens := scoreCandidateTokenOverlap(candidate, sourceTokens)
	if score < minInferredEdgeScore || len(matchedTokens) == 0 {
		return score, ""
	}
	return score, "shared inferred support tokens: " + strings.Join(matchedTokens, ", ")
}

func indexGraphEdgesByFrom(edges []ProjectGraphEdge, statuses ...string) map[string][]ProjectGraphEdge {
	filterStatuses := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		filterStatuses[status] = struct{}{}
	}

	byFrom := make(map[string][]ProjectGraphEdge)
	for _, edge := range edges {
		if len(filterStatuses) > 0 {
			if _, ok := filterStatuses[edge.Status]; !ok {
				continue
			}
		}
		byFrom[edge.From] = append(byFrom[edge.From], edge)
	}
	return byFrom
}
