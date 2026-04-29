# Relay — TODO 한글판

활성 CEO/엔지니어링/디자인 계획 바깥으로 미뤄 둔 작업 목록이다. 각 항목은 원래 결정 출처를 남겨 이후 맥락을 복구할 수 있게 한다.

## V2 구현 상태

- **S1 (인증 기반)** ✅ 완료, 2026-04-25, 브랜치 `feat/s1-auth-foundation`
  - 포함 범위: migration `0007_users.sql`, 도메인 타입, repository, OAuth provider 추상화, GitHub + Google, E4 완화책, cookie session middleware, `/v1/auth/{provider}/start|callback`, `/v1/auth/me`, `/v1/auth/logout`, Style Memory 쓰기 경로의 project-owner authorization, R1 admin path 유지, unit test, OpenAPI 업데이트.
  - **세션 refresh 정책** (V2.5 stable-token update, 2026-04-28): user session은 30일 TTL + `/v1/auth/me` 전용 7일 rolling refresh를 적용한다. refresh는 `relay_session` token 값을 유지하고 `user_sessions.expires_at`만 연장하므로, concurrent `/me` 요청이 sibling tab의 cookie를 무효화하지 않는다. Style Memory 등 다른 authenticated endpoint는 expiry 연장 없이 검증만 한다.
  - **향후 session 작업**: token-family / grace-window storage는 device/session management, "sign out other devices", token-family analytics, theft-detection semantics가 필요해질 때까지 이월한다.

## V2 CEO Plan Rev 2에서 이월된 항목

출처: `~/.gstack/projects/relay/ceo-plans/2026-04-25-v2-end-user-surface.md` Rev 2

V2 scope 재정의 이후 범위: 최소 authenticated Style Memory UI + A(Sharable Packet Snapshot URL) + C(1-click Onboarding Flow).

아래 항목은 **V2에 포함하지 않고**, V2.5 / V3 후보로 추적한다.

### V2.5 (load-bearing, V2 ship 이후 다음 제품 사이클)

**1. DESIGN.md 남은 4개 priority screen** — Project Explorer, Trace browser, Decision Graph, Packet Builder WYSIWYG
- Effort: L, UI 약 5일 + backend read-model API 약 5일
- 이월 이유: Codex finding #3. 이 화면들은 단순 UI가 아니라 backend read-model 작업이 크다. 현재 `judgment_traces`, pending proposals, approved heuristics list endpoint가 없고, project graph도 traces/heuristics를 포함하지 않는다. V2는 1개 화면만 ship한다.
- Unblocks: DESIGN.md 전체 구현, dense workspace experience, signature ribbon
- Depends on: V2 ship + 최소 한 사이클의 demand evidence
- Source: CEO plan Rev 2 Scope Decision #4
- V2.5 진입 slice: `docs/v2-5-project-explorer-scope.md`

**2. B: Public Style Profile** — `/u/{username}/style`, engineering judgment용 LinkedIn surface
- Effort: L/XL. cross-project aggregation, slug reservation, privacy state, publish UX 때문에 Rev 1의 M 추정은 크게 과소평가였다. User model은 V2 baseline에 들어갔으므로 일부 재사용 가능.
- 이월 이유: Codex finding #5. cross-project aggregation에 의존하며 이는 V3 영역이다.
- Unblocks: portfolio surface, identity moment, 개별 snapshot URL을 넘어서는 viral entry point
- Depends on: V2 demand validation, V3 candidate A(Multi-Project Style Transfer)의 cross-project heuristic aggregation primitive
- Source: CEO plan Rev 2 Scope Decision #5

**3. F: Multi-agent Live Workspace** — Claude + Codex 동시 driver visualization
- Effort: L. Codex finding #6 기준으로 현재 MCP stateless request-response 위에 얹는 시각화가 아니라 새로운 realtime event product다.
- 이월 이유: 가치가 높지만 realtime infrastructure 비용이 크다. V2 demand validation 이후 정당화한다.
- Unblocks: thesis demo, multi-agent Style Memory consumption, single-agent Mem/Reflect/Notion AI 대비 차별화
- Depends on: V2 ship, realtime infrastructure 결정(SSE, WebSocket, polling), Codex trace ingestion path 해결
- Source: CEO plan Rev 2 Scope Decision #6

**4. Inline Curator Suggestions** — approval 중 confidence + reasoning side panel
- Effort: M, backend curator extension 이후 약 2일
- 이월 이유: backend curator가 아직 proposal별 confidence/reasoning을 내보내지 않는다.
- Depends on: backend curator가 per-proposal confidence + reasoning field를 emit하도록 확장
- Source: CEO plan Rev 2 Scope Decision #8

### Eng Review failure-mode mitigations

출처: `/gstack-plan-eng-review`, 2026-04-25, Section 4 failure modes table

**E1. Onboarding KMS error UX**
- Production failure: KMS encrypt 실패(IAM/rate limit) -> silent 500 -> 사용자가 무한 retry
- Mitigation: `internal/services/onboarding.go`에서 명시적 error code를 내고, Frame 2에서 recoverable UX 표시. 문구: "Couldn't securely store your key. We'll try again automatically." + retry button
- Effort: S, 약 2시간

**E2. Snapshot revoke 시 CDN cache invalidation**
- Production failure: `public_readable=false` toggle 이후 CDN이 stale OG image를 계속 제공 -> 외부 수신자는 page는 410인데 preview는 오래된 상태로 봄
- Mitigation: `POST /v1/snapshots/{id}/publish` toggle에 CDN cache invalidation API 연결. Vercel이면 Cache-Tag invalidation, Cloudflare면 purge API.
- Effort: S, 약 2시간
- Note: CDN provider 결정 필요

**E3. 900ms transform 중 "Saving..." chip**
- Production failure: transform 중 API가 5초 걸림 -> card가 `aria-busy` 상태로 멈춘 것처럼 보임 -> 사용자는 UI가 frozen됐다고 인식
- Mitigation: `aria-busy` 1.5초 이후 subtle Fraunces italic "Saving..." chip 표시, 5초 이후 "Still saving... [Cancel]" option 표시
- Effort: S, 약 1시간

**E4. GitHub OAuth missing email scope detection** ✅ 구현 완료, 2026-04-25
- Production failure: GitHub가 valid token을 주지만 private email 때문에 user account 생성이 막힘
- Mitigation: `internal/lib/oauth/github.go`의 `Exchange`에서 `/user` 이후 항상 `/user/emails`를 호출해 primary verified address를 선택한다. 그래도 비어 있으면 NULL email user record로 OAuth flow를 계속 진행하고 auto-link step은 건너뛴다.
- Effort: S, 약 1-2시간

### V2.5 / V3 후보 (delight, load-bearing 아님)

**5. Embed Widget** — blog/README packet preview용 `<script>` embed
- Effort: M, 약 3일
- 이월 이유: viral layer. Sharable URL 트래픽 데이터가 실제로 나오면 추가한다.
- Depends on: V2 A traffic data, CSP/iframe security model
- Source: CEO plan Rev 2 Scope Decision #7

**6. Style-aware vs Control Inline Diff Toggle**
- Effort: S, 약 4시간
- 이월 이유: 이미 eval script에 있다. UI 복제는 nice-to-have.
- Source: CEO plan Rev 2 Scope Decision #9

**7. Cmd+K Command Palette**
- Effort: S, 약 4시간
- 이월 이유: power-user delight. 초기 외부 사용자 5명은 command palette 유무로 V2를 평가하지 않는다.
- Source: CEO plan Rev 2 Scope Decision #10

**8. Project Pulse Heatmap**
- Effort: S, 약 4시간
- 이월 이유: surface enhancement이며 PMF 검증의 핵심이 아니다.
- Source: CEO plan Rev 2 Scope Decision #11

**9. Replay Last Handoff Debugger**
- Effort: M, 약 1일
- 이월 이유: ops/internal tool. demand evidence를 만들지 않는다.
- Source: CEO plan Rev 2 Scope Decision #12

## V3 후보 (V2.5 이후)

출처: CEO plan Rev 2 "Phase 2 / Phase 3 Trajectory"

- **V3 candidate A**: Multi-Project Style Transfer. V1 design doc의 원래 V2 expansion으로, Style Memory를 여러 프로젝트에 걸쳐 일반화한다. B(V2.5)는 이 primitive 버전에 의존하고, full version은 V3다.
- **V3 candidate B**: Embeddable trace + widget + integrations. Viral platform layer.
- **V3 candidate C**: Multi-user / org accounts / team workspaces. 개인 demand가 검증된 경우에만 추진한다.

## V2 plan-design-review에서 남은 design debt

출처: `~/.gstack/projects/relay/ceo-plans/2026-04-25-v2-end-user-surface.md` Rev 2 + cross-model design review. Codex outside voice와 Claude subagent 모두 6/10.

### Critical — V2 구현 전에 처리해야 했던 항목

**D1. Sharable URL HTML 100% fidelity 재작성** ✅ 완료, 2026-04-25
- `/gstack-design-html`로 Rev 2 finalized. Pretext-native layout + design review critical findings 반영.
- File: `~/.gstack/projects/relay/designs/sharable-url-20260425/finalized.html`
- Pretext sibling: `~/.gstack/projects/relay/designs/sharable-url-20260425/pretext.js`
- 닫힌 finding: hero CTA, swan-contour seal, first-heuristic-above-fold, mood reduction, density modulation, memorable thing footer

**D2. Onboarding HTML 100% fidelity 재작성** ✅ 완료, 2026-04-25
- `/gstack-design-html`로 Rev 2 finalized. Pretext-native layout + 9개 design review critical findings 반영.
- File: `~/.gstack/projects/relay/designs/onboarding-20260425/finalized.html`
- Pretext sibling: `~/.gstack/projects/relay/designs/onboarding-20260425/pretext.js`
- 닫힌 finding: F1 hero memorable thing, F4 LIVE 900ms transform, "Capture your first judgment" CTA, F2 4개 error state panel, F2 magic-primary-strong glow + editorial typography, F3 active step pill, storyboard chrome 제거, F1 outcome preview, mood repetition 제거

**D3. Style Memory single gravitational card refactor**
- DESIGN.md §8 위반. 두 reviewer 모두 지적.
- Action: finalized HTML을 수정해 proposals를 confidence 기준으로 ranking하고, rank-1에만 post-approval state preview + halo + magic-accent border + Fraunces italic 적용
- Effort: S, 약 1-2시간

**D4. 900ms signature transform failure-mode design**
- Open Q #10
- Action: Style Memory finalized HTML에 optimistic UI render, `aria-busy` lock, 실패 시 220ms rewind, `--danger` chip "Couldn't save - retry?", transform 중 height lock 추가
- Effort: S, 약 1-2시간

### High — V2 implementation slice 중 처리할 항목

**D5. Sharable URL static brand signal**
- read-only surface라 motion을 재생할 수 없으므로 frozen swan-contour seal을 brand surrogate로 사용
- Effort: S, 약 1시간

**D6. Sharable URL hero CTA**
- Codex HARD REJECTION
- Effort: trivial, 약 30분

**D7. Style Memory time-depth meta line**
- 5년 단위 reflective dimension을 저비용으로 추가
- Effort: trivial, 약 30분

**D8. OG image generation pipeline + visual spec**
- Visual spec: 1200x630, `--canvas` background, Fraunces 96px headline = first heuristic title, Fraunces italic 28px rationale, swan-contour silhouette 30% opacity bottom-right, base mono meta strip
- Effort: M, 약 2-3시간

### Medium — V2.5 territory

**D9. Decision Graph magic-color misuse fix**
- magic color를 graph vocabulary로 쓰지 말고 selected/yielded transform path에만 예약한다.
- Effort: S, 약 1-2시간

**D10. Packet Builder source panel default-closed**
- source panel이 기본 open이면 packet document를 방해한다. 기본 collapsed로 두고 toggle로 승격한다.
- Effort: trivial, 약 30분

**D11. Project Explorer ops stats demoted to inspector drawer**
- curator success 같은 pipe-level system detail은 main metric strip이 아니라 inspector drawer로 이동
- Effort: S, 약 1시간

**D12. Judgment Traces first-run state**
- filter surface가 처음부터 너무 dense하다. 기본은 one trace narrative, 이후 filters로 확장.
- Effort: S, 약 1-2시간

### 누락된 first-run / empty / error / failure state

Cross-model design review는 7개 surface 전체에서 약 40개의 누락 state를 확인했다. 각 state는 visual spec, backend trigger, keyboard/accessibility behavior가 필요하다.

**Style Memory**: pending proposal 없음, approved heuristic 없음, approval failure mid-transform, edit validation, reject undo, auth expired, project 미선택, 다른 디바이스의 sync conflict, single-proposal layout

**Sharable URL**: private snapshot, expired/deleted snapshot, owner revoked public access, empty packet, missing OG image, share/copy success, broken/expired link

**Onboarding**: invalid Anthropic key, Relay URL unreachable, OAuth cancelled, secret-storage consent prompt, prior trace 없음, first packet build failure, Anthropic rate limit/quota, returning user short-circuit

**V2.5 surfaces**: Project Explorer, Judgment Traces, Decision Graph, Packet Builder의 약 16개 추가 state는 V2.5 entry 시 design review에서 spec한다.

## 운영 개선 (gstack 관련)

- **Slug mismatch는 upstream에서 해결됨**: `gstack-slug`는 `relay`, `remote-slug`는 `4GLY-relay`를 반환한다. `remote-slug`로 project artifact를 찾는 skill bash는 `relay/` directory를 놓친다. gstack issue 또는 local override 필요. 2026-04-25 `~/.gstack/projects/relay/learnings.jsonl`에 learning으로 기록.

## Rev History

- **Rev 1 (2026-04-25)**: V2 cherry-pick 이후 초기 deferred set. B + F accepted, A + C deferred.
- **Rev 2 (2026-04-25, post-Codex reframe)**: cross-model tension 이후 V2 cherry-pick이 뒤집힘. A + C가 V2로 들어오고, B + F + DESIGN.md 남은 4개 화면은 V2.5로 이동.
