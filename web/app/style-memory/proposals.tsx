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
  approvedFetchFailed: boolean;
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
  approvedFetchFailed: initialApprovedFailed,
  userDisplayName,
}: Props) {
  const reduceMotion = useReducedMotion();

  const [pending, setPending] = useState<PendingProposal[]>(() => rankSorted(initialPending));
  const [approved, setApproved] = useState<ApprovedHeuristic[]>(initialApproved);
  const [approvedFailed, setApprovedFailed] = useState(initialApprovedFailed);
  const [tab, setTab] = useState<TabKey>("proposals");
  const [view, setView] = useState<ViewMode>("single");
  const [viewHydrated, setViewHydrated] = useState(false);

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

  // F3: refetch on focus to pick up cross-tab approvals.
  useEffect(() => {
    const onFocus = () => {
      // Soft refetch: re-fire pending list. We swallow errors silently here;
      // the main UI keeps last known state on transient failure.
      void refetchPending();
    };
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const refetchPending = useCallback(async () => {
    try {
      const { listPendingProposals } = await import("@/lib/heuristics");
      const result = await listPendingProposals(projectId, { limit: 50 });
      setPending(rankSorted(result.items));
    } catch {
      /* keep existing list */
    }
  }, [projectId]);

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
          proposalId,
          action: "approve",
          signal: controller.signal,
        });
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
    [pushToast, reduceMotion],
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
          proposalId,
          action: "reject",
          reviewNotes,
          signal: controller.signal,
        });
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
    [pushToast],
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
    if (view === "batch") return `${pending.length} in queue · batch view`;
    if (pending.length === 1) return "One hero in focus.";
    return `One hero in focus. ${pending.length - 1} queued behind.`;
  }, [pending.length, view]);

  return (
    <div style={layoutStyle}>
      <header style={railStyle}>
        <div style={wordmarkStyle}>
          Relay<span style={{ color: "var(--magic-primary-strong)" }}>.</span>
        </div>
        <div style={{ justifySelf: "end", display: "flex", gap: "10px", alignItems: "center" }}>
          <span style={muteMonoStyle}>{userDisplayName ?? "signed in"}</span>
        </div>
      </header>

      <main style={workspaceStyle} aria-label="Style Memory workspace">
        <div style={wsHeadStyle}>
          <h2 style={wsTitleStyle}>Style Memory</h2>
          <div style={wsSubStyle}>{subTitle}</div>
          <button
            type="button"
            onClick={() => setView((v) => (v === "single" ? "batch" : "single"))}
            style={viewToggleStyle(view === "batch")}
            aria-pressed={view === "batch"}
            data-testid="view-toggle"
          >
            {view === "batch" ? "Single hero" : "View all queue"}
          </button>
        </div>

        <nav style={tabsStyle} role="tablist" aria-label="Style Memory views">
          <TabButton active={tab === "proposals"} count={pending.length} onClick={() => setTab("proposals")}>
            Proposals
          </TabButton>
          <TabButton active={tab === "approved"} count={approved.length} onClick={() => setTab("approved")}>
            Approved
          </TabButton>
          <TabButton active={tab === "rejected"} count={0} onClick={() => setTab("rejected")}>
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
                const { listApprovedHeuristics } = await import("@/lib/heuristics");
                const fresh = await listApprovedHeuristics(projectId, { limit: 100 });
                setApproved(fresh.items);
                setApprovedFailed(false);
              } catch {
                /* keep failed state */
              }
            }}
          />
        )}

        {tab === "rejected" && (
          <p style={emptyStateStyle}>
            Rejected heuristics live in the curator pipeline as negative-memory signal. Surfacing them in the UI ships in V2.5.
          </p>
        )}
      </main>

      <ToastDock toasts={toasts} />

      {view === "batch" && (
        <div style={hintBarStyle} role="status" aria-live="polite">
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
    <button type="button" role="tab" aria-selected={active} onClick={onClick} style={tabBtnStyle(active)}>
      {children}
      <span style={tabCountStyle}>{count}</span>
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
      <div style={emptyStateBlockStyle} role="status">
        <h3 style={{ fontFamily: "var(--font-display)", fontWeight: 500, fontSize: "22px", marginBottom: "8px" }}>
          All quiet on the swan front.
        </h3>
        <p style={{ fontFamily: "var(--font-display)", fontStyle: "italic", color: "var(--ink-muted)" }}>
          No pending proposals. Capture some judgment traces and the curator will refill the queue.
        </p>
      </div>
    );
  }

  return (
    <div style={proposalsStyle}>
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
        <div style={queueLabelStyle}>
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

      <div style={cardTopStyle(isQueued)}>
        <span style={scopeChipStyle}>
          <span style={{ color: "var(--magic-primary)" }}>◈</span>
          {(proposal.workflow || "scope") + " × " + (proposal.artifactType || "any")}
        </span>
        {isQueued && (
          <span style={peekTextStyle} title={proposal.canonicalText}>
            {proposal.canonicalText}
          </span>
        )}
        <span style={confidenceStyle}>
          {conf !== null ? (
            <>
              proposed <b style={confBoldStyle}>{conf.toFixed(2)}</b> · from {proposal.sourceTraceIds.length} trace
              {proposal.sourceTraceIds.length === 1 ? "" : "s"}
            </>
          ) : (
            <>proposed · from {proposal.sourceTraceIds.length} traces</>
          )}
        </span>
      </div>

      {!isQueued && (
        <>
          <div style={diffStyle}>
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

          {proposal.reviewNotes && (
            <p style={rationaleStyle}>{proposal.reviewNotes}</p>
          )}

          {(proposal.sourceTraceIds.length > 0 || proposal.sourceRefs.length > 0) && (
            <div style={provenanceStyle}>
              {proposal.sourceTraceIds.map((tid) => (
                <span key={tid} style={sourceChipStyle("trace")}>
                  trace #{tid.slice(0, 6)}
                </span>
              ))}
              {proposal.sourceRefs.map((ref) => (
                <span key={ref} style={sourceChipStyle("note")}>
                  ref #{ref.slice(0, 6)}
                </span>
              ))}
            </div>
          )}

          <div style={actionsStyle}>
            <button
              type="button"
              style={btnStyle("danger")}
              onClick={() =>
                setRejectDraft({ proposalId: proposal.proposalId, freeText: "" })
              }
              disabled={inflight}
            >
              Reject
            </button>
            <button
              type="button"
              style={btnStyle("primary")}
              onClick={() => void approveProposal(proposal.proposalId)}
              disabled={inflight}
              data-testid={`approve-${proposal.proposalId}`}
            >
              Approve → Swan
            </button>
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
            <div style={savingChipDockStyle}>
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
    <div style={savingChipStyle}>
      <span>{label}</span>
      {onCancel && (
        <button type="button" onClick={onCancel} style={cancelLinkStyle}>
          Cancel
        </button>
      )}
    </div>
  );
}

function ErrorChip({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div style={{ ...savingChipStyle, borderColor: "var(--danger)", color: "var(--danger)" }}>
      <span>Couldn’t save — {message.length > 40 ? "retry?" : message}</span>
      <button type="button" onClick={onRetry} style={cancelLinkStyle}>
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
      style={rejectOverlayStyle}
      role="region"
      aria-label="Reject reason picker"
      onKeyDown={handleKey}
      data-testid="reject-overlay"
    >
      <p style={rejectOverlayLabelStyle}>Why reject?</p>
      <div style={chipRowStyle}>
        {REJECT_REASON_CODES.map((code) => (
          <button
            key={code}
            type="button"
            onClick={() => onChange({ ...draft, selected: code })}
            style={reasonChipStyle(draft.selected === code)}
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
          style={rejectTextareaStyle}
          aria-label="Reject reason free text"
          data-testid="reject-other-textarea"
        />
      )}
      <div style={{ display: "flex", justifyContent: "flex-end", gap: "8px", marginTop: "10px" }}>
        <button type="button" style={btnStyle("ghost")} onClick={onCancel}>
          Cancel
        </button>
        <button
          type="button"
          style={btnStyle("primary")}
          onClick={onSubmit}
          disabled={submitDisabled}
          data-testid="reject-submit"
        >
          Submit
        </button>
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
      <div style={emptyStateBlockStyle}>
        <p style={{ marginBottom: "12px" }}>Couldn’t load approved heuristics.</p>
        <button type="button" style={btnStyle("ghost")} onClick={onRetry}>
          Retry
        </button>
      </div>
    );
  }
  if (items.length === 0) {
    return (
      <div style={emptyStateBlockStyle}>
        <p>No approved heuristics yet — approve a duckling and it will appear here.</p>
      </div>
    );
  }
  return (
    <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: "10px" }}>
      {items.map((h) => (
        <li
          key={h.heuristicId}
          style={{
            border: "1px solid var(--border)",
            borderRadius: "10px",
            padding: "14px 16px",
            background: "var(--canvas-raised)",
            display: "flex",
            flexDirection: "column",
            gap: "6px",
          }}
        >
          <div style={{ display: "flex", justifyContent: "space-between", gap: "12px" }}>
            <span style={scopeChipStyle}>
              <span style={{ color: "var(--magic-primary)" }}>◈</span>
              {(h.workflow || "scope") + " × " + (h.artifactType || "any")}
            </span>
            <span style={muteMonoStyle}>state: {h.state}</span>
          </div>
          <p style={{ fontFamily: "var(--font-mono)", fontSize: "12px", color: "var(--ink-muted)", margin: 0 }}>
            {h.canonicalText}
          </p>
        </li>
      ))}
    </ul>
  );
}

function ToastDock({ toasts }: { toasts: ToastMessage[] }) {
  return (
    <div style={toastDockStyle}>
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
            style={{
              ...toastStyle,
              borderColor: t.tone === "danger" ? "var(--danger)" : "var(--magic-primary-strong)",
            }}
          >
            <div style={toastTitleStyle}>{t.title}</div>
            {t.sub && <div style={toastSubStyle}>{t.sub}</div>}
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}

/* ---------------- Styles ---------------- */

const layoutStyle: CSSProperties = {
  minHeight: "100vh",
  background: "var(--canvas)",
  color: "var(--ink)",
  display: "flex",
  flexDirection: "column",
};

const railStyle: CSSProperties = {
  height: "56px",
  display: "grid",
  gridTemplateColumns: "1fr auto",
  alignItems: "center",
  padding: "0 24px",
  borderBottom: "1px solid var(--border)",
  background: "color-mix(in oklab, var(--canvas) 88%, transparent)",
  position: "sticky",
  top: 0,
  zIndex: 30,
};

const wordmarkStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontWeight: 600,
  fontSize: "22px",
  letterSpacing: "-0.02em",
  fontVariationSettings: '"opsz" 48, "SOFT" 30',
};

const muteMonoStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  color: "var(--muted)",
  letterSpacing: "0.04em",
};

const workspaceStyle: CSSProperties = {
  width: "100%",
  maxWidth: "960px",
  margin: "0 auto",
  padding: "28px 32px 80px",
  flex: 1,
};

const wsHeadStyle: CSSProperties = {
  display: "flex",
  alignItems: "baseline",
  gap: "20px",
  paddingBottom: "18px",
  borderBottom: "1px solid var(--border)",
  marginBottom: "22px",
  flexWrap: "wrap",
};

const wsTitleStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontWeight: 500,
  fontSize: "30px",
  letterSpacing: "-0.022em",
  fontVariationSettings: '"opsz" 48, "SOFT" 40',
  margin: 0,
};

const wsSubStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  fontSize: "14px",
  color: "var(--ink-muted)",
  flex: 1,
};

function viewToggleStyle(active: boolean): CSSProperties {
  return {
    fontFamily: "var(--font-mono)",
    fontSize: "11px",
    letterSpacing: "0.08em",
    textTransform: "uppercase",
    padding: "6px 12px",
    borderRadius: "999px",
    border: `1px solid ${active ? "var(--magic-primary-strong)" : "var(--border-strong)"}`,
    background: active ? "color-mix(in oklab, var(--magic-primary) 25%, var(--canvas-raised))" : "transparent",
    color: "var(--ink)",
    cursor: "pointer",
  };
}

const tabsStyle: CSSProperties = {
  display: "flex",
  gap: "4px",
  marginBottom: "18px",
};

function tabBtnStyle(active: boolean): CSSProperties {
  return {
    fontFamily: "var(--font-sans)",
    fontSize: "12px",
    fontWeight: 600,
    padding: "7px 14px",
    borderRadius: "999px",
    color: active ? "var(--canvas)" : "var(--muted)",
    background: active ? "var(--ink)" : "transparent",
    border: "1px solid transparent",
    letterSpacing: "0.02em",
    cursor: "pointer",
    display: "inline-flex",
    alignItems: "center",
    gap: "6px",
  };
}

const tabCountStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  opacity: 0.75,
  letterSpacing: "0.04em",
};

const proposalsStyle: CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "14px",
};

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
    padding: "20px 22px",
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
      borderLeft: "3px solid var(--magic-accent-strong)",
      paddingLeft: "25px",
      borderColor: "var(--magic-primary-strong)",
      boxShadow:
        "0 0 0 4px var(--halo), 0 24px 40px -24px color-mix(in oklab, var(--magic-accent-strong) 40%, transparent)",
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

function cardTopStyle(queued: boolean): CSSProperties {
  if (queued) {
    return {
      display: "grid",
      gridTemplateColumns: "auto 1fr auto",
      gap: "14px",
      alignItems: "center",
    };
  }
  return {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "flex-start",
    marginBottom: "14px",
    gap: "16px",
    flexWrap: "wrap",
  };
}

const peekTextStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "11.5px",
  color: "var(--ink-muted)",
  overflow: "hidden",
  textOverflow: "ellipsis",
  whiteSpace: "nowrap",
  letterSpacing: "0.01em",
  minWidth: 0,
};

const scopeChipStyle: CSSProperties = {
  display: "inline-flex",
  gap: "6px",
  alignItems: "center",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  padding: "5px 9px",
  borderRadius: "6px",
  background: "var(--problem-soft)",
  color: "var(--canvas)",
  letterSpacing: "0.04em",
};

const confidenceStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  color: "var(--muted)",
  letterSpacing: "0.04em",
};

const confBoldStyle: CSSProperties = {
  color: "var(--magic-primary-strong)",
  fontWeight: 500,
  fontVariantNumeric: "tabular-nums",
};

const diffStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "8px",
  fontFamily: "var(--font-mono)",
  fontSize: "11.5px",
  marginBottom: "16px",
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

const rationaleStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  fontWeight: 400,
  fontSize: "15px",
  color: "var(--ink-muted)",
  padding: "4px 0 0 14px",
  marginBottom: "14px",
  borderLeft: "2px solid color-mix(in oklab, var(--magic-accent) 50%, transparent)",
  fontVariationSettings: '"opsz" 48',
};

const provenanceStyle: CSSProperties = {
  display: "flex",
  gap: "6px",
  flexWrap: "wrap",
  marginBottom: "16px",
};

function sourceChipStyle(_kind: "trace" | "note"): CSSProperties {
  return {
    display: "inline-flex",
    alignItems: "center",
    gap: "6px",
    fontFamily: "var(--font-mono)",
    fontSize: "10.5px",
    padding: "3px 9px",
    borderRadius: "999px",
    border: "1px solid var(--border-strong)",
    color: "var(--ink-muted)",
    letterSpacing: "0.03em",
  };
}

const actionsStyle: CSSProperties = {
  display: "flex",
  gap: "8px",
  justifyContent: "flex-end",
  flexWrap: "wrap",
};

function btnStyle(variant: "primary" | "ghost" | "danger"): CSSProperties {
  const base: CSSProperties = {
    fontFamily: "var(--font-sans)",
    fontWeight: 600,
    fontSize: "12.5px",
    padding: "8px 15px",
    borderRadius: "8px",
    border: "1px solid transparent",
    letterSpacing: "0.01em",
    cursor: "pointer",
  };
  if (variant === "primary") return { ...base, background: "var(--ink)", color: "var(--canvas)" };
  if (variant === "danger") {
    return {
      ...base,
      background: "transparent",
      color: "var(--danger)",
      borderColor: "color-mix(in oklab, var(--danger) 35%, transparent)",
    };
  }
  return {
    ...base,
    background: "transparent",
    color: "var(--ink-muted)",
    borderColor: "var(--border-strong)",
  };
}

const queueLabelStyle: CSSProperties = {
  display: "flex",
  alignItems: "center",
  gap: "12px",
  margin: "10px 2px 2px",
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  color: "var(--muted)",
};

const rejectOverlayStyle: CSSProperties = {
  marginTop: "14px",
  paddingTop: "14px",
  borderTop: "1px dashed var(--border)",
  display: "flex",
  flexDirection: "column",
  gap: "10px",
};

const rejectOverlayLabelStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "10px",
  letterSpacing: "0.14em",
  textTransform: "uppercase",
  color: "var(--muted)",
  margin: 0,
};

const chipRowStyle: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "6px",
};

function reasonChipStyle(active: boolean): CSSProperties {
  return {
    fontFamily: "var(--font-mono)",
    fontSize: "11px",
    padding: "5px 10px",
    borderRadius: "999px",
    border: `1px solid ${active ? "var(--magic-primary-strong)" : "var(--border-strong)"}`,
    background: active ? "color-mix(in oklab, var(--magic-primary) 22%, var(--canvas-raised))" : "transparent",
    color: "var(--ink)",
    cursor: "pointer",
    letterSpacing: "0.03em",
  };
}

const rejectTextareaStyle: CSSProperties = {
  fontFamily: "var(--font-mono)",
  fontSize: "12px",
  padding: "10px 12px",
  borderRadius: "8px",
  border: "1px solid var(--border-strong)",
  background: "var(--canvas-raised)",
  color: "var(--ink)",
  resize: "vertical",
  minHeight: "70px",
};

const savingChipDockStyle: CSSProperties = {
  position: "absolute",
  top: "12px",
  right: "12px",
};

const savingChipStyle: CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: "8px",
  padding: "5px 10px",
  borderRadius: "999px",
  background: "var(--canvas-raised)",
  border: "1px solid var(--border-strong)",
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  fontSize: "12px",
  color: "var(--ink-muted)",
};

const cancelLinkStyle: CSSProperties = {
  background: "transparent",
  border: "none",
  color: "var(--magic-primary-strong)",
  fontFamily: "var(--font-mono)",
  fontSize: "11px",
  cursor: "pointer",
  padding: 0,
};

const emptyStateStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  fontSize: "16px",
  color: "var(--ink-muted)",
  padding: "40px 0",
};

const emptyStateBlockStyle: CSSProperties = {
  border: "1px dashed var(--border-strong)",
  borderRadius: "12px",
  padding: "32px 24px",
  textAlign: "center",
  color: "var(--ink-muted)",
};

const toastDockStyle: CSSProperties = {
  position: "fixed",
  right: "20px",
  bottom: "20px",
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  zIndex: 50,
  pointerEvents: "none",
};

const toastStyle: CSSProperties = {
  background: "var(--canvas-raised)",
  border: "1px solid var(--magic-primary-strong)",
  color: "var(--ink)",
  borderRadius: "10px",
  padding: "12px 16px",
  fontSize: "13px",
  boxShadow: "0 0 0 4px var(--halo), 0 20px 40px -20px rgba(0,0,0,0.15)",
  maxWidth: "320px",
  pointerEvents: "auto",
};

const toastTitleStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontWeight: 600,
  fontSize: "14px",
  marginBottom: "4px",
  letterSpacing: "-0.01em",
};

const toastSubStyle: CSSProperties = {
  fontFamily: "var(--font-display)",
  fontStyle: "italic",
  color: "var(--ink-muted)",
  fontSize: "12.5px",
};

const hintBarStyle: CSSProperties = {
  position: "fixed",
  bottom: "20px",
  left: "20px",
  padding: "10px 14px",
  background: "color-mix(in oklab, var(--ink) 92%, transparent)",
  color: "var(--canvas)",
  borderRadius: "999px",
  fontFamily: "var(--font-mono)",
  fontSize: "10.5px",
  letterSpacing: "0.08em",
  textTransform: "uppercase",
  zIndex: 40,
};
