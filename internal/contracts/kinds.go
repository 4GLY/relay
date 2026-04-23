package contracts

type PromotionKind string

const (
	PromotionKindDecision PromotionKind = "decision"
	PromotionKindQuestion PromotionKind = "question"
)

type PacketKind string

const (
	PacketKindResume       PacketKind = "resume"
	PacketKindStyleHandoff PacketKind = "style_handoff"
)

func NormalizePacketKind(value string) string {
	switch PacketKind(value) {
	case "":
		return string(PacketKindResume)
	case PacketKindResume, PacketKindStyleHandoff:
		return value
	default:
		return value
	}
}

type PacketTarget string

const (
	PacketTargetGeneric PacketTarget = "generic"
	PacketTargetCodex   PacketTarget = "codex"
)

func NormalizePacketTarget(value string) string {
	switch PacketTarget(value) {
	case "":
		return string(PacketTargetGeneric)
	case PacketTargetGeneric, PacketTargetCodex:
		return value
	default:
		return value
	}
}

type WorkflowKind string

const (
	WorkflowDesignHandoff WorkflowKind = "design_handoff"
)

type ArtifactKind string

const (
	ArtifactKindDesignDoc     ArtifactKind = "design_doc"
	ArtifactKindDecisionNote  ArtifactKind = "decision_note"
	ArtifactKindHandoffPacket ArtifactKind = "handoff_packet"
	ArtifactKindGitCommits    ArtifactKind = "git_commits"
	ArtifactKindLegacyHandoff ArtifactKind = "handoff_md"
)

type HeuristicState string

const (
	HeuristicStatePending  HeuristicState = "pending"
	HeuristicStateApproved HeuristicState = "approved"
	HeuristicStateRejected HeuristicState = "rejected"
	HeuristicStateArchived HeuristicState = "archived"
	HeuristicStateDisabled HeuristicState = "disabled"
)

type CuratorJobState string

const (
	CuratorJobStatePending CuratorJobState = "pending"
	CuratorJobStateLeased  CuratorJobState = "leased"
	CuratorJobStateDone    CuratorJobState = "done"
	CuratorJobStateFailed  CuratorJobState = "failed"
)

type CuratorJobKind string

const (
	CuratorJobKindJudgmentTrace CuratorJobKind = "judgment_trace"
)
