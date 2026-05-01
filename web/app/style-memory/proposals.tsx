"use client";

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent as ReactKeyboardEvent,
} from "react";
import { motion, AnimatePresence, useReducedMotion } from "framer-motion";

import {
  RelayButton,
  RelayEmptyState,
  RelayLinkButton,
  RelayPageHead,
  RelaySourceChip,
  RelayStatusBadge,
  RelayTopRail,
} from "@/components/relay";
import { cn } from "@/lib/utils";
import {
  ProposalAlreadyResolvedError,
  REJECT_REASON_CODES,
  reviewProposal,
  serializeRejectNotes,
  type ApprovedHeuristic,
  type PendingProposal,
  type RejectReasonCode,
} from "@/lib/heuristics";

const VIEW_STORAGE_KEY = "relay-style-memory-view";
type ViewMode = "single" | "batch";
type TabKey = "proposals" | "approved" | "rejected";

type Props = {
  projectId: string;
  initialPending: PendingProposal[];
  initialApproved: ApprovedHeuristic[];
  initialRejected: PendingProposal[];
  approvedFetchFailed: boolean;
  rejectedFetchFailed: boolean;
  userDisplayName?: string;
  userId: string;
};

type CardStatus =
  | { kind: "idle" }
  | { kind: "approving"; startedAt: number; controller: AbortController }
  | { kind: "rejecting"; startedAt: number; controller: AbortController }
  | { kind: "error"; message: string };

type ToastMessage = {
  id: number;
  title: string;
  sub?: string;
  tone: "success" | "danger";
};

type RejectDraft = {
  proposalId: string;
  selected?: RejectReasonCode;
  freeText: string;
};

const REASON_LABELS: Record<RejectReasonCode, string> = {
  duplicate: "duplicate",
  wrong: "factually wrong",
  too_narrow: "too narrow",
  too_broad: "too broad",
  stale: "stale",
  other: "other…",
};

function loadStoredView(): ViewMode {
  if (typeof window === "undefined") return "single";
  try {
    const raw = window.localStorage.getItem(VIEW_STORAGE_KEY);
    if (raw === "batch" || raw === "single") return raw;
  } catch {
    /* localStorage unavailable */
  }
  return "single";
}

function persistView(mode: ViewMode) {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(VIEW_STORAGE_KEY, mode);
  } catch {
    /* swallow */
  }
}

function confidenceFromRefs(p: PendingProposal): number | null {
  // V2 baseline: backend does not surface a numeric confidence; rank by
  // source breadth (more traces → more agreement). Returns 0..1.
  const traceCount = p.sourceTraceIds.length;
  if (traceCount === 0) return null;
  return Math.min(1, traceCount / 5);
}

function rankSorted(list: PendingProposal[]): PendingProposal[] {
  return [...list].sort((a, b) => {
    const aRefs = a.sourceTraceIds.length;
    const bRefs = b.sourceTraceIds.length;
    if (aRefs !== bRefs) return bRefs - aRefs;
    return a.createdAt < b.createdAt ? -1 : a.createdAt > b.createdAt ? 1 : 0;
  });
}

function isFromInteractiveTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return target.matches("input, textarea, [contenteditable=true]");
}

let toastSeq = 0;

export function Proposals({
  projectId,
  initialPending,
  initialApproved,
  initialRejected,
  approvedFetchFailed: initialApprovedFailed,
  rejectedFetchFailed: initialRejectedFailed,
  userDisplayName,
}: Props) {
  const reduceMotion = useReducedMotion();

  const [pending, setPending] = useState<PendingProposal[]>(() => rankSorted(initialPending));
  const [approved, setApproved] = useState<ApprovedHeuristic[]>(initialApproved);
  const [rejected, setRejected] = useState<PendingProposal[]>(initialRejected);
  const [approvedFailed, setApprovedFailed] = useState(initialApprovedFailed);
  const [rejectedFailed, setRejectedFailed] = useState(initialRejectedFailed);
  const [tab, setTab] = useState<TabKey>("proposals");
  const [view, setView] = useState<ViewMode>("single");
  const [viewHydrated, setViewHydrated] = useState(false);
  const [showShortcutHint, setShowShortcutHint] = useState(false);

  const [statuses, setStatuses] = useState<Record<string, CardStatus>>({});
  const [rejectDraft, setRejectDraft] = useState<RejectDraft | null>(null);
  const [focusedId, setFocusedId] = useState<string | null>(null);
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const [animatingApproveId, setAnimatingApproveId] = useState<string | null>(null);

  const cardRefs = useRef<Record<string, HTMLElement | null>>({});

  // statusesRef mirrors statuses so async callbacks can read the live value
  // without re-creating themselves on every status change. Prevents key
  // handler rebinding storms in batch mode.
  const statusesRef = useRef(statuses);
  useEffect(() => {
    statusesRef.current = statuses;
  }, [statuses]);

  // Load persisted view choice after mount (avoid SSR/CSR mismatch).
  useEffect(() => {
    setView(loadStoredView());
    setViewHydrated(true);
  }, []);

  useEffect(() => {
    if (viewHydrated) persistView(view);
  }, [view, viewHydrated]);

  useEffect(() => {
    if (typeof window.matchMedia !== "function") {
      setShowShortcutHint(true);
      return;
    }
    const media = window.matchMedia("(min-width: 640px)");
    const sync = () => setShowShortcutHint(media.matches);
    sync();
    media.addEventListener("change", sync);
    return () => media.removeEventListener("change", sync);
  }, []);

  const refetchPending = useCallback(async () => {
    try {
      const { listPendingProposals } = await import("@/lib/heuristics");
      const result = await listPendingProposals(projectId, { limit: 50 });
      setPending(rankSorted(result.items));
    } catch {
      /* keep existing list */
    }
  }, [projectId]);

  const refetchApproved = useCallback(async () => {
    const { listApprovedHeuristics } = await import("@/lib/heuristics");
    const fresh = await listApprovedHeuristics(projectId, { limit: 100 });
    setApproved(fresh.items);
    setApprovedFailed(false);
    return fresh.items;
  }, [projectId]);

  const refetchRejected = useCallback(async () => {
    const { listRejectedProposals } = await import("@/lib/heuristics");
    const fresh = await listRejectedProposals(projectId, { limit: 50 });
    setRejected(fresh.items);
    setRejectedFailed(false);
    return fresh.items;
  }, [projectId]);

  const refetchReviewState = useCallback(async () => {
    await Promise.allSettled([refetchPending(), refetchApproved(), refetchRejected()]);
  }, [refetchApproved, refetchPending, refetchRejected]);

  // F3: refetch on focus to pick up cross-tab review state changes.
  useEffect(() => {
    const onFocus = () => {
      void refetchReviewState();
    };
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [refetchReviewState]);

  const pushToast = useCallback((toast: Omit<ToastMessage, "id">) => {
    const id = ++toastSeq;
    setToasts((t) => [...t, { ...toast, id }]);
    setTimeout(() => {
      setToasts((t) => t.filter((x) => x.id !== id));
    }, 4500);
  }, []);

  // Set initial focus when batch mode toggles on.
  useEffect(() => {
    if (view === "batch" && !focusedId && pending[0]) {
      setFocusedId(pending[0].proposalId);
    }
  }, [view, focusedId, pending]);

  // Derive hero + queue.
  const hero = view === "single" ? pending[0] ?? null : null;
  const queue = view === "single" ? pending.slice(1) : pending;

  const approveProposal = useCallback(
    async (proposalId: string) => {
      const existing = statusesRef.current[proposalId];
      if (existing && (existing.kind === "approving" || existing.kind === "rejecting")) return;
      const controller = new AbortController();
      setStatuses((s) => ({
        ...s,
        [proposalId]: { kind: "approving", startedAt: Date.now(), controller },
      }));
      setAnimatingApproveId(proposalId);
      try {
        await reviewProposal({
          projectId,
          proposalId,
          action: "approve",
          signal: controller.signal,
        });
        try {
          await refetchApproved();
        } catch {
          setApprovedFailed(true);
        }
        // Hold the swan animation through its full 900ms (or 160ms reduced).
        const holdMs = reduceMotion ? 200 : 950;
        await new Promise((resolve) => setTimeout(resolve, holdMs));
        setPending((cur) => cur.filter((p) => p.proposalId !== proposalId));
        setStatuses((s) => {
          const next = { ...s };
          delete next[proposalId];
          return next;
        });
        setAnimatingApproveId((cur) => (cur === proposalId ? null : cur));
        pushToast({ title: "Swan minted", sub: "Heuristic added to the gallery.", tone: "success" });
      } catch (err) {
        setAnimatingApproveId((cur) => (cur === proposalId ? null : cur));
        if (err instanceof ProposalAlreadyResolvedError) {
          setPending((cur) => cur.filter((p) => p.proposalId !== proposalId));
          setStatuses((s) => {
            const next = { ...s };
            delete next[proposalId];
            return next;
          });
          pushToast({ title: "Already resolved", sub: "Another session resolved this card.", tone: "danger" });
          return;
        }
        const message = err instanceof Error ? err.message : "Couldn’t save — retry?";
        setStatuses((s) => ({ ...s, [proposalId]: { kind: "error", message } }));
      }
    },
    [projectId, pushToast, reduceMotion, refetchApproved],
  );

  const submitReject = useCallback(
    async (draft: RejectDraft) => {
      if (!draft.selected) return;
      if (draft.selected === "other" && draft.freeText.trim().length < 10) return;
      const proposalId = draft.proposalId;
      const controller = new AbortController();
      setStatuses((s) => ({
        ...s,
        [proposalId]: { kind: "rejecting", startedAt: Date.now(), controller },
      }));
      try {
        const reviewNotes = serializeRejectNotes(draft.selected, draft.freeText);
        await reviewProposal({
          projectId,
          proposalId,
          action: "reject",
          reviewNotes,
          signal: controller.signal,
        });
        try {
          await refetchRejected();
        } catch {
          setRejectedFailed(true);
        }
        setPending((cur) => cur.filter((p) => p.proposalId !== proposalId));
        setStatuses((s) => {
          const next = { ...s };
          delete next[proposalId];
          return next;
        });
        setRejectDraft(null);
        pushToast({ title: "Rejected", sub: `Reason: ${draft.selected}`, tone: "success" });
      } catch (err) {
        if (err instanceof ProposalAlreadyResolvedError) {
          setPending((cur) => cur.filter((p) => p.proposalId !== proposalId));
          setRejectDraft(null);
          pushToast({ title: "Already resolved", sub: "Another session resolved this card.", tone: "danger" });
          return;
        }
        const message = err instanceof Error ? err.message : "Couldn’t reject — retry?";
        setStatuses((s) => ({ ...s, [proposalId]: { kind: "error", message } }));
      }
    },
    [projectId, pushToast, refetchRejected],
  );

  const cancelInflight = useCallback((proposalId: string) => {
    const status = statusesRef.current[proposalId];
    if (!status) return;
    if (status.kind === "approving" || status.kind === "rejecting") {
      status.controller.abort();
      setStatuses((s) => {
        const next = { ...s };
        delete next[proposalId];
        return next;
      });
      setAnimatingApproveId((cur) => (cur === proposalId ? null : cur));
    }
  }, []);

  // Keyboard navigation in batch mode (Contract C).
  useEffect(() => {
    if (view !== "batch") return;
    const handler = (e: KeyboardEvent) => {
      // Contract C INPUT focus exception: ignore shortcuts inside text fields.
      if (isFromInteractiveTarget(e.target)) return;
      if (e.key === "Escape" && rejectDraft) {
        setRejectDraft(null);
        return;
      }
      const list = pending;
      if (list.length === 0) return;
      const currentIdx = list.findIndex((p) => p.proposalId === focusedId);
      const idx = currentIdx === -1 ? 0 : currentIdx;
      if (e.key === "j") {
        e.preventDefault();
        const next = list[Math.min(list.length - 1, idx + 1)];
        if (next) {
          setFocusedId(next.proposalId);
          scrollToCard(next.proposalId);
        }
      } else if (e.key === "k") {
        e.preventDefault();
        const prev = list[Math.max(0, idx - 1)];
        if (prev) {
          setFocusedId(prev.proposalId);
          scrollToCard(prev.proposalId);
        }
      } else if (e.key === "a") {
        e.preventDefault();
        const cur = list[idx];
        if (cur) void approveProposal(cur.proposalId);
      } else if (e.key === "x") {
        e.preventDefault();
        const cur = list[idx];
        if (cur) {
          setRejectDraft({ proposalId: cur.proposalId, freeText: "" });
        }
      } else if (e.key === "Enter") {
        // No-op: cards are full-detail in batch (Contract C); reserved for future expand/collapse.
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [view, pending, focusedId, rejectDraft, approveProposal]);

  function scrollToCard(id: string) {
    const el = cardRefs.current[id];
    if (!el) return;
    const top = el.getBoundingClientRect().top + window.scrollY - 64;
    window.scrollTo({ top, behavior: "smooth" });
  }

  const subTitle = useMemo(() => {
    if (pending.length === 0) return "All quiet on the swan front.";
    if (pending.length === 1) return "One proposal waiting for your judgment.";
    return `${pending.length} proposals waiting for your judgment.`;
  }, [pending.length]);

  return (
    <div className="relay-style-memory-page">
      <RelayTopRail
        activeStep="Refine"
        userLabel={userDisplayName}
        projectHref={`/style-memory?project=${encodeURIComponent(projectId)}`}
      />

      <main className="relay-style-memory-workspace" aria-label="Style Memory workspace">
        <RelayPageHead
          eyebrow={<>{projectId} · Style Memory</>}
          title={subTitle}
          actions={
            <>
            <RelayLinkButton href={`/projects/${encodeURIComponent(projectId)}/packet-builder`} variant="secondary">
              Compose handoff
            </RelayLinkButton>
            <RelayButton
              type="button"
              variant={view === "batch" ? "primary" : "secondary"}
              onClick={() => setView((v) => (v === "single" ? "batch" : "single"))}
              aria-pressed={view === "batch"}
              data-testid="view-toggle"
            >
              {view === "batch" ? "Single hero" : "View all queue"}
            </RelayButton>
            </>
          }
        />

        <nav className="relay-tabs relay-style-memory-tabs" role="tablist" aria-label="Style Memory views">
          <TabButton active={tab === "proposals"} count={pending.length} onClick={() => setTab("proposals")}>
            Proposals
          </TabButton>
          <TabButton active={tab === "approved"} count={approved.length} onClick={() => setTab("approved")}>
            Approved
          </TabButton>
          <TabButton active={tab === "rejected"} count={rejected.length} onClick={() => setTab("rejected")}>
            Rejected
          </TabButton>
        </nav>

        {tab === "proposals" && (
          <ProposalList
            view={view}
            hero={hero}
            queue={queue}
            cardRefs={cardRefs}
            focusedId={focusedId}
            statuses={statuses}
            rejectDraft={rejectDraft}
            setRejectDraft={setRejectDraft}
            submitReject={submitReject}
            approveProposal={approveProposal}
            cancelInflight={cancelInflight}
            animatingApproveId={animatingApproveId}
            reduceMotion={!!reduceMotion}
            onFocusCard={setFocusedId}
          />
        )}

        {tab === "approved" && (
          <ApprovedList
            items={approved}
            failed={approvedFailed}
            onRetry={async () => {
              try {
                await refetchApproved();
              } catch {
                /* keep failed state */
              }
            }}
          />
        )}

        {tab === "rejected" && (
          <RejectedList
            items={rejected}
            failed={rejectedFailed}
            onRetry={async () => {
              try {
                await refetchRejected();
              } catch {
                /* keep failed state */
              }
            }}
          />
        )}
      </main>

      <ToastDock toasts={toasts} />

      {view === "batch" && showShortcutHint && (
        <div className="relay-style-memory-hint" role="status" aria-live="polite">
          <kbd>j</kbd>/<kbd>k</kbd> navigate · <kbd>a</kbd> approve · <kbd>x</kbd> reject · <kbd>Esc</kbd> close
        </div>
      )}
    </div>
  );
}

function TabButton({
  active,
  count,
  children,
  onClick,
}: {
  active: boolean;
  count: number;
  children: React.ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className="relay-tab relay-style-memory-tab"
      data-active={active || undefined}
    >
      {children}
      <span className="relay-tab-count">{count}</span>
    </button>
  );
}

type ProposalListProps = {
  view: ViewMode;
  hero: PendingProposal | null;
  queue: PendingProposal[];
  cardRefs: React.MutableRefObject<Record<string, HTMLElement | null>>;
  focusedId: string | null;
  statuses: Record<string, CardStatus>;
  rejectDraft: RejectDraft | null;
  setRejectDraft: (d: RejectDraft | null) => void;
  submitReject: (d: RejectDraft) => Promise<void>;
  approveProposal: (id: string) => Promise<void>;
  cancelInflight: (id: string) => void;
  animatingApproveId: string | null;
  reduceMotion: boolean;
  onFocusCard: (id: string | null) => void;
};

function ProposalList({
  view,
  hero,
  queue,
  cardRefs,
  focusedId,
  statuses,
  rejectDraft,
  setRejectDraft,
  submitReject,
  approveProposal,
  cancelInflight,
  animatingApproveId,
  reduceMotion,
  onFocusCard,
}: ProposalListProps) {
  if (!hero && queue.length === 0) {
    return (
      <RelayEmptyState
        role="status"
        title="All quiet on the swan front."
        copy="No pending proposals. Capture some judgment traces and the curator will refill the queue."
      />
    );
  }

  return (
    <div className="relay-style-memory-proposals">
      <AnimatePresence initial={false}>
        {view === "single" && hero && (
          <ProposalCard
            key={hero.proposalId}
            proposal={hero}
            mode="hero"
            cardRefs={cardRefs}
            focused={false}
            status={statuses[hero.proposalId]}
            rejectDraft={rejectDraft}
            setRejectDraft={setRejectDraft}
            submitReject={submitReject}
            approveProposal={approveProposal}
            cancelInflight={cancelInflight}
            animating={animatingApproveId === hero.proposalId}
            reduceMotion={reduceMotion}
            onFocusCard={onFocusCard}
          />
        )}
      </AnimatePresence>

      {view === "single" && queue.length > 0 && (
        <div className="relay-style-memory-queue-label">
          <span>{queue.length} queued · resolve hero first</span>
        </div>
      )}

      <AnimatePresence initial={false}>
        {(view === "single" ? queue : queue).map((p) => (
          <ProposalCard
            key={p.proposalId}
            proposal={p}
            mode={view === "single" ? "queued" : "batch"}
            cardRefs={cardRefs}
            focused={view === "batch" && focusedId === p.proposalId}
            status={statuses[p.proposalId]}
            rejectDraft={rejectDraft}
            setRejectDraft={setRejectDraft}
            submitReject={submitReject}
            approveProposal={approveProposal}
            cancelInflight={cancelInflight}
            animating={animatingApproveId === p.proposalId}
            reduceMotion={reduceMotion}
            onFocusCard={onFocusCard}
          />
        ))}
      </AnimatePresence>
    </div>
  );
}

type CardMode = "hero" | "queued" | "batch";

type ProposalCardProps = {
  proposal: PendingProposal;
  mode: CardMode;
  cardRefs: React.MutableRefObject<Record<string, HTMLElement | null>>;
  focused: boolean;
  status?: CardStatus;
  rejectDraft: RejectDraft | null;
  setRejectDraft: (d: RejectDraft | null) => void;
  submitReject: (d: RejectDraft) => Promise<void>;
  approveProposal: (id: string) => Promise<void>;
  cancelInflight: (id: string) => void;
  animating: boolean;
  reduceMotion: boolean;
  onFocusCard: (id: string | null) => void;
};

function ProposalCard({
  proposal,
  mode,
  cardRefs,
  focused,
  status,
  rejectDraft,
  setRejectDraft,
  submitReject,
  approveProposal,
  cancelInflight,
  animating,
  reduceMotion,
  onFocusCard,
}: ProposalCardProps) {
  const isQueued = mode === "queued";
  const inflight = status?.kind === "approving" || status?.kind === "rejecting";
  const startedAt = status?.kind === "approving" || status?.kind === "rejecting" ? status.startedAt : 0;
  const elapsed = useElapsed(startedAt, !!inflight);
  const showSavingChip = inflight && elapsed >= 1500;
  const showStillSaving = inflight && elapsed >= 5000;

  const isRejecting = rejectDraft?.proposalId === proposal.proposalId;
  const cardStyle = computeCardStyle(mode, focused, animating, reduceMotion, status?.kind === "error");
  const conf = confidenceFromRefs(proposal);

  return (
    <motion.article
      ref={(el) => {
        cardRefs.current[proposal.proposalId] = el;
      }}
      initial={animating ? false : { opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={
        reduceMotion
          ? { opacity: 0, transition: { duration: 0.16 } }
          : animating
            ? { opacity: 0, scale: 0.96, transition: { duration: 0.4, delay: 0.5 } }
            : { opacity: 0, height: 0, transition: { duration: 0.22 } }
      }
      transition={{ duration: reduceMotion ? 0.16 : 0.22 }}
      style={cardStyle}
      data-card={proposal.proposalId}
      data-state={animating ? "approving" : status?.kind ?? "idle"}
      data-focused={focused ? "true" : undefined}
      aria-busy={inflight || undefined}
      onClick={() => mode === "batch" && onFocusCard(proposal.proposalId)}
    >
      {animating && !reduceMotion && <SwanGlow />}

      <div className={cn("relay-style-memory-card-top", isQueued && "relay-style-memory-card-top-queued")}>
        <RelaySourceChip>
          <span className="relay-magic-text">◈</span>
          {(proposal.workflow || "scope") + " × " + (proposal.artifactType || "any")}
        </RelaySourceChip>
        {isQueued && (
          <span className="relay-style-memory-peek" title={proposal.canonicalText}>
            {proposal.canonicalText}
          </span>
        )}
        <span className="relay-style-memory-confidence">
          {conf !== null ? (
            <>
              proposed <b className="relay-magic-text">{conf.toFixed(2)}</b> · from {proposal.sourceTraceIds.length} trace
              {proposal.sourceTraceIds.length === 1 ? "" : "s"}
            </>
          ) : (
            <>proposed · from {proposal.sourceTraceIds.length} traces</>
          )}
        </span>
      </div>

      {!isQueued && (
        <>
          <div className="relay-style-memory-diff" style={diffStyle}>
            {proposal.normalizedText ? (
              <div style={diffSideStyle("before")}>
                <span style={diffMetaStyle("before")}>Current heuristic</span>
                {proposal.normalizedText}
              </div>
            ) : (
              <div style={diffSideStyle("before")}>
                <span style={diffMetaStyle("before")}>Current heuristic</span>
                <em style={{ opacity: 0.6 }}>none — first revision</em>
              </div>
            )}
            <div style={diffSideStyle("after")}>
              <span style={diffMetaStyle("after")}>Proposed refinement</span>
              {proposal.canonicalText}
            </div>
          </div>

          {proposal.reviewNotes && <p className="relay-style-memory-rationale">{proposal.reviewNotes}</p>}

          {(proposal.sourceTraceIds.length > 0 || proposal.sourceRefs.length > 0) && (
            <div className="relay-style-memory-provenance">
              {proposal.sourceTraceIds.map((tid) => (
                <RelayStatusBadge key={tid}>
                  <span className="relay-style-memory-dot" data-variant="trace" />
                  trace · {tid.slice(0, 10)}
                </RelayStatusBadge>
              ))}
              {proposal.sourceRefs.map((ref) => (
                <RelayStatusBadge key={ref}>
                  <span className="relay-style-memory-dot" data-variant="note" />
                  ref · {ref.slice(0, 10)}
                </RelayStatusBadge>
              ))}
            </div>
          )}

          <div className="relay-style-memory-actions">
            <RelayButton
              type="button"
              variant="danger"
              onClick={() =>
                setRejectDraft({ proposalId: proposal.proposalId, freeText: "" })
              }
              disabled={inflight}
            >
              Reject
            </RelayButton>
            <RelayButton
              type="button"
              variant="primary"
              onClick={() => void approveProposal(proposal.proposalId)}
              disabled={inflight}
              data-testid={`approve-${proposal.proposalId}`}
            >
              Approve → Swan
            </RelayButton>
          </div>

          {isRejecting && rejectDraft && (
            <RejectOverlay
              draft={rejectDraft}
              onChange={setRejectDraft}
              onSubmit={() => void submitReject(rejectDraft)}
              onCancel={() => setRejectDraft(null)}
            />
          )}

          {(showSavingChip || status?.kind === "error") && (
            <div className="relay-style-memory-chip-dock">
              {status?.kind === "error" ? (
                <ErrorChip
                  message={status.message}
                  onRetry={() => void approveProposal(proposal.proposalId)}
                />
              ) : showStillSaving ? (
                <SavingChip
                  label="Still saving…"
                  onCancel={() => cancelInflight(proposal.proposalId)}
                />
              ) : (
                <SavingChip label="Saving…" />
              )}
            </div>
          )}
        </>
      )}
    </motion.article>
  );
}

function SwanGlow() {
  return (
    <motion.div
      // TODO(design-system): inline-style exception. Framer glow uses runtime
      // animation coordinates; replace if we introduce CSS animation variants.
      style={{
        position: "absolute",
        inset: 0,
        borderRadius: "12px",
        background:
          "linear-gradient(110deg, transparent 0%, color-mix(in oklab, var(--magic-primary) 40%, transparent) 35%, color-mix(in oklab, var(--magic-accent-strong) 55%, transparent) 55%, transparent 90%)",
        filter: "blur(14px)",
        pointerEvents: "none",
      }}
      initial={{ opacity: 0, x: "-10%" }}
      animate={{ opacity: [0, 1, 0], x: ["-10%", "30%", "110%"] }}
      transition={{ duration: 0.9, ease: [0.2, 0.8, 0.2, 1] }}
    />
  );
}

function useElapsed(startedAt: number, active: boolean) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const id = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(id);
  }, [active]);
  return active ? now - startedAt : 0;
}

function SavingChip({ label, onCancel }: { label: string; onCancel?: () => void }) {
  return (
    <div className="relay-style-memory-save-chip">
      <span>{label}</span>
      {onCancel && (
        <button type="button" onClick={onCancel} className="relay-link-reset relay-mono-link">
          Cancel
        </button>
      )}
    </div>
  );
}

function ErrorChip({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="relay-style-memory-save-chip" data-variant="danger">
      <span>Couldn’t save — {message.length > 40 ? "retry?" : message}</span>
      <button type="button" onClick={onRetry} className="relay-link-reset relay-mono-link">
        Retry
      </button>
    </div>
  );
}

function RejectOverlay({
  draft,
  onChange,
  onSubmit,
  onCancel,
}: {
  draft: RejectDraft;
  onChange: (d: RejectDraft) => void;
  onSubmit: () => void;
  onCancel: () => void;
}) {
  const submitDisabled =
    !draft.selected ||
    (draft.selected === "other" && draft.freeText.trim().length < 10) ||
    draft.freeText.length > 200;

  function handleKey(e: ReactKeyboardEvent<HTMLDivElement>) {
    if (e.key === "Escape") {
      e.stopPropagation();
      onCancel();
    }
  }

  return (
    <div
      className="relay-style-memory-reject"
      role="region"
      aria-label="Reject reason picker"
      onKeyDown={handleKey}
      data-testid="reject-overlay"
    >
      <p className="relay-style-memory-reject-label">Why reject?</p>
      <div className="relay-chip-row">
        {REJECT_REASON_CODES.map((code) => (
          <button
            key={code}
            type="button"
            onClick={() => onChange({ ...draft, selected: code })}
            className="relay-style-memory-reason-chip"
            data-active={draft.selected === code || undefined}
            aria-pressed={draft.selected === code}
            data-testid={`reject-chip-${code}`}
          >
            {REASON_LABELS[code]}
          </button>
        ))}
      </div>
      {draft.selected === "other" && (
        <textarea
          value={draft.freeText}
          onChange={(e) => onChange({ ...draft, freeText: e.target.value })}
          placeholder="Explain why (10–200 chars)"
          maxLength={200}
          rows={3}
          className="relay-style-memory-textarea"
          aria-label="Reject reason free text"
          data-testid="reject-other-textarea"
        />
      )}
      <div className="relay-style-memory-actions">
        <RelayButton type="button" variant="secondary" onClick={onCancel}>
          Cancel
        </RelayButton>
        <RelayButton
          type="button"
          variant="primary"
          onClick={onSubmit}
          disabled={submitDisabled}
          data-testid="reject-submit"
        >
          Submit
        </RelayButton>
      </div>
    </div>
  );
}

function ApprovedList({
  items,
  failed,
  onRetry,
}: {
  items: ApprovedHeuristic[];
  failed: boolean;
  onRetry: () => void;
}) {
  if (failed) {
    return (
      <RelayEmptyState copy="Couldn’t load approved heuristics.">
        <RelayButton type="button" variant="secondary" onClick={onRetry}>
          Retry
        </RelayButton>
      </RelayEmptyState>
    );
  }
  if (items.length === 0) {
    return (
      <RelayEmptyState copy="No approved heuristics yet — approve a proposal and it will appear here." />
    );
  }
  return (
    <ul className="relay-style-memory-list">
      {items.map((h) => (
        <li key={h.heuristicId} className="relay-list-row relay-style-memory-list-row">
          <div className="relay-list-header">
            <RelaySourceChip>
              <span className="relay-magic-text">◈</span>
              {(h.workflow || "scope") + " × " + (h.artifactType || "any")}
            </RelaySourceChip>
            <span className="relay-style-memory-muted">state: {h.state}</span>
          </div>
          <p className="relay-style-memory-list-copy">
            {h.canonicalText}
          </p>
        </li>
      ))}
    </ul>
  );
}

function RejectedList({
  items,
  failed,
  onRetry,
}: {
  items: PendingProposal[];
  failed: boolean;
  onRetry: () => void;
}) {
  if (failed) {
    return (
      <RelayEmptyState copy="Couldn’t load rejected proposals.">
        <RelayButton type="button" variant="secondary" onClick={onRetry}>
          Retry
        </RelayButton>
      </RelayEmptyState>
    );
  }
  if (items.length === 0) {
    return <RelayEmptyState copy="No rejected proposals yet." />;
  }
  return (
    <ul className="relay-style-memory-list">
      {items.map((p) => (
        <li key={p.proposalId} className="relay-list-row relay-style-memory-list-row" data-variant="danger">
          <div className="relay-list-header">
            <RelaySourceChip>
              <span className="relay-danger-text">×</span>
              {(p.workflow || "scope") + " × " + (p.artifactType || "any")}
            </RelaySourceChip>
            <span className="relay-style-memory-muted">{p.reviewNotes || "rejected"}</span>
          </div>
          <p className="relay-style-memory-list-copy">
            {p.canonicalText}
          </p>
        </li>
      ))}
    </ul>
  );
}

function ToastDock({ toasts }: { toasts: ToastMessage[] }) {
  return (
    <div className="relay-style-memory-toast-dock">
      <AnimatePresence initial={false}>
        {toasts.map((t) => (
          <motion.div
            key={t.id}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.22 }}
            role="status"
            aria-live="polite"
            className="relay-style-memory-toast"
            data-variant={t.tone}
          >
            <div className="relay-style-memory-toast-title">{t.title}</div>
            {t.sub && <div className="relay-style-memory-toast-sub">{t.sub}</div>}
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}

/* ---------------- Styles ---------------- */

// TODO(design-system): inline-style exception. Style Memory cards still compute
// Framer Motion layout/exit state dynamically; keep static styling in CSS.
function computeCardStyle(
  mode: CardMode,
  focused: boolean,
  animating: boolean,
  _reduceMotion: boolean,
  errored: boolean,
): CSSProperties {
  const base: CSSProperties = {
    border: "1px solid var(--border)",
    borderRadius: "12px",
    background: "var(--canvas-raised)",
    padding: "24px",
    position: "relative",
    overflow: "hidden",
  };
  if (mode === "queued") {
    return {
      ...base,
      padding: "13px 16px",
      background: "color-mix(in oklab, var(--problem-soft) 5%, var(--canvas-raised))",
    };
  }
  if (mode === "hero" || animating) {
    return {
      ...base,
      borderColor: "var(--magic-primary-strong)",
      boxShadow: "0 0 0 4px color-mix(in srgb, var(--magic-primary-strong) 18%, transparent)",
    };
  }
  if (mode === "batch") {
    return {
      ...base,
      borderColor: errored ? "var(--danger)" : "var(--border)",
      boxShadow: focused ? "0 0 0 2px var(--magic-primary-strong)" : undefined,
    };
  }
  return base;
}

// TODO(design-system): inline-style exception. The diff panel has before/after
// semantic colors and will move to a small RelayDiff primitive if reused.
const diffStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "16px",
  fontFamily: "var(--font-mono)",
  fontSize: "15px",
  marginBottom: "22px",
};

function diffSideStyle(side: "before" | "after"): CSSProperties {
  if (side === "before") {
    return {
      padding: "11px 13px",
      borderRadius: "8px",
      lineHeight: 1.55,
      whiteSpace: "pre-wrap",
      wordBreak: "break-word",
      background: "var(--problem-soft)",
      color: "color-mix(in oklab, var(--canvas) 88%, transparent)",
    };
  }
  return {
    padding: "11px 13px",
    borderRadius: "8px",
    lineHeight: 1.55,
    whiteSpace: "pre-wrap",
    wordBreak: "break-word",
    background: "color-mix(in oklab, var(--magic-primary) 22%, var(--canvas))",
    color: "var(--ink)",
    border: "1px solid color-mix(in oklab, var(--magic-primary) 40%, transparent)",
  };
}

function diffMetaStyle(side: "before" | "after"): CSSProperties {
  return {
    display: "block",
    fontSize: "9px",
    letterSpacing: "0.14em",
    textTransform: "uppercase",
    marginBottom: "5px",
    fontWeight: 500,
    color: side === "before"
      ? "color-mix(in oklab, var(--canvas) 60%, transparent)"
      : "var(--magic-primary-strong)",
  };
}
