# Relay Design System Hardening Plan

## 0) 목적

Relay UI 하드닝은 화면별 임시 CSS를 줄이고, `web/app/globals.css` + `web/components/relay-app-shell.tsx` 기반의 **안정적 컴포넌트 계약**으로 정착시키는 작업이다.

- 기존 제품 언어(색·타이포·레이아웃 톤)는 변경하지 않는다.
- 라우트 동작(API, 인증, 라우팅, 저장 동작)은 변경 없이 유지한다.
- 공통 UI는 `relay-*` 클래스와 데이터 속성 기반 variant로 통일한다.
- 문서에서 계약을 선행 정의하고 구현을 따라가게 하여 회귀를 줄인다.

Source of truth:

- `DESIGN.md is the canonical visual identity source` and follows the
  `google-labs-code/design.md` format: YAML token front matter plus canonical
  Markdown rationale sections.
- `DESIGN.md`: brand meaning, normative tokens, component token references,
  and section-ordered design rationale.
- `web/app/globals.css`: Tailwind v4 token projection and concrete `relay-*`
  class behavior.
- `web/components/relay/*`: React primitive API for screens.
- `docs/design-system.md`: engineering rules for applying the system without
  re-litigating brand decisions.
- Validate the contract with:

  ```bash
  npm --prefix web run design:lint
  ```

---

## 1) 설계 의도

- 릴레이는 `Face / Dissect / Refine / Transform` 흐름을 유지하는 
  **차분하고 밀도 높은 작업 공간**이어야 한다.
- 카드, 패널, 버튼, 배지, 상태 표시 같은 반복 패턴은 페이지에서 직접 그리지 않고
  프리미티브 컴포넌트(API)로 노출한다.
- 단일 라이트/다크 토큰 체계를 기반으로 페이지 간 시각·접근성 일관성을 유지한다.
- 모바일에서는 겹침 없는 가독성, 터치 가능성, 가로 스크롤 최소화를 우선한다.

---

## 2) 토큰 계약 (Token Contract)

### 2.1 단일 진입점
- 정식 소스는 `web/app/globals.css`의 `@theme` 및 `@layer base` 변수.
- 컴포넌트는 하드코딩 컬러/폰트/거리값을 사용하지 않고 **의미 토큰**만 사용한다.

### 2.2 색/표면 토큰

- Surface
  - `--canvas`, `--canvas-raised`
  - `--ink`, `--ink-muted`, `--muted`, `--problem`, `--problem-soft`, `--border`, `--border-strong`
  - `--magic-primary`, `--magic-primary-strong`, `--magic-accent`, `--magic-accent-strong`
  - `--success`, `--danger`
- Focus/Halo
  - `--halo`, `--halo-strong`, `--grain-opacity`
- Typography
  - `--font-display`, `--font-sans`, `--font-mono`
- Shape
  - `--radius-card`, `--radius-pill`

### 2.3 테마 계약
- Light/Dark는 `:root` 와 `:root[data-theme="dark"], .dark` 그리고
  `prefers-color-scheme: dark` 분기에서 변수 교체한다.
- 컴포넌트는 현재 토큰만 사용하고, 새 변수 추가 시 기존 패턴에 먼저 합의 후만 허용한다.

### 2.4 금지/권고 규칙
- 권고: 시맨틱 변수 이름만 사용. 예: `background: var(--canvas-raised)`
- 금지: 새 디자인 산출물에서 임시 hex/rgba 하드코딩 사용.
- 기존 `--color-*`(`--color-canvas` 등)는 유지되지만, 신규 도입점은 `--canvas` 계열 의미 토큰.

---

## 3) 컴포넌트 카탈로그 및 계약

### 3.1 Shell primitives (현재 구현/기준점)

- `RelayTopRail`
  - 클래스: `relay-top-rail`
  - 하위: `relay-wordmark`, `relay-wordmark-dot`, `relay-transform-ribbon`,
    `relay-transform-step`, `relay-transform-arrow`, `relay-top-actions`,
    `relay-top-link`, `relay-user-chip`
  - 상태: `relay-transform-step[data-active="true"]`, `aria-label`
- `RelayAppShell`
  - 클래스: `relay-app-shell`, `relay-shell-main`, `relay-inspector`
- 프로젝트 내비게이션(현재 Shell 내부)
  - `relay-project-rail`, `relay-rail-section`, `relay-rail-item`, `relay-rail-glyph`,
    `relay-rail-name`, `relay-badge-duckling`, `relay-badge-swan`
  - 상태: `relay-rail-item[data-active="true"]`, `relay-rail-glyph[data-kind="snapshot|pending|active"]`

### 3.2 데이터/카드 계열 (현재 정적 클래스, 향후 컴포넌트 래핑 대상)

- Card
  - 클래스: `relay-card` (+ `data-selected="true"`)
  - 계약: 테두리/배경/반경/선택 강조는 클래스 내부로 고정
- Metric
  - 클래스: `relay-metric`, `relay-metric-value`, `relay-metric-label`
  - 계약: 숫자 계량값은 텍스트 대비를 확보한 큰 폰트 우선
- Meta
  - 클래스: `relay-meta-label`

### 3.3 현재 페이지 연동에서 필요한 신규 primitive API

아래 항목은 계획상 신규 도입 예정이거나 규격 정리가 필요한 항목이다.

- `RelayPageHead`, `RelayPageActions`, `RelayPageKicker`
- `RelayCard`, `RelayCardHeader`, `RelayCardTitle`, `RelayCardKicker`
- `RelayButton`, `RelayLinkButton`
- `RelayTabs`
- `RelayMetricTile`
- `RelayStatusBadge`, `RelaySourceChip`
- `RelayMetaGrid`, `RelayListRow`
- `RelayField`, `RelayTextInput`, `RelayFeedback`, `RelayEmptyState`

각 컴포넌트는 페이지별 인라인 스타일을 제거하고,
`className` + 최소 variant prop로 동작해야 한다.

### 3.4 라우트별 하드닝 우선순위

1. Settings (Provider / API Key)
2. Onboarding + Root entry
3. Project Explorer / Trace Browser / Decision Graph / Packet Builder / Style Memory

---

## 4) Variant / 접근성 / 모바일 규칙

### 4.1 Variant 규칙(권장 기본값)

- 버튼 계열
  - `data-variant="primary|secondary|ghost|danger"`
- 링크/행동 버튼
  - `RelayLinkButton`은 `<a>` 역할 유지
- 배지/상태
  - `data-variant="default|success|warning|danger|info"`
- 빈 상태/피드백
  - `data-state="empty|loading|error|warning|success"`
- 카드 선택/탭 상태
  - `data-selected="true"`, `data-active="true"`, `aria-current="page"`

### 4.2 접근성 규칙

- Shell/페이지 구조는 시맨틱 HTML 사용
  - `header`, `nav`, `main`, `aside` 유지
- 입력/조작 요소
  - 버튼/필드 라벨은 명시적 텍스트 라벨 또는 `aria-label`
  - 에러/상태 알림은 `role="alert"` 또는 `aria-live="polite"`
- 키보드
  - 포커스 표시는 전역 `:focus-visible` 토큰 기반으로 통일
- 탭 네비게이션은 `aria-current`, 필요 시 `role="tablist|tab"`, `aria-selected`
- 아이콘 단독 텍스트는 접근 가능한 대체 텍스트 또는 `aria-hidden` + 텍스트 라벨 동시 제공

### 4.3 모바일 규칙

- `max-width: 1180px`
  - 측면 인스펙터는 숨김
- `max-width: 760px`
  - `relay-top-rail` 1열화
  - `relay-transform-ribbon` 수평 오버플로우 허용
  - `relay-project-rail` 하단 경계 전환
  - `relay-shell-main` 패딩 축소
  - `relay-trace-row` 단일 열 스택
  - `relay-packet-builder-grid`, `relay-style-memory-diff` 1열 스택
- 목표: 화면 밖 오버플로우 및 내용 겹침 방지

---

## 5) Inline-style 예외 정책

### 5.1 금지 규칙
- 신규 컴포넌트/페이지는 기본적으로 인라인 스타일 미사용.
- 인라인 스타일은 “컴포넌트 계약”으로 대체한다.

### 5.2 허용 예외
- 불가피한 런타임 계산값
  - 예: 동적 픽셀/브라우저 측정치 기반 레이아웃
- 서드파티 스타일 충돌 브릿지
  - 라이브러리 API에서 class/style 중 하나만 허용되는 경우
- 임시 마이그레이션 패치
  - 1차 하드닝에서 즉시 제거 예정인 임시 바인딩

### 5.3 허용 시 필수 메타
- 예외 사용 위치에 다음을 코드 주석으로 남김: 
  - `TODO(design-system): inline-style exception`
  - 사유 + 제거 예정 조건(예: 페이지 마이그레이션 완료 시 삭제)
- 동일 대상 스타일은 같은 목적의 클래스 추출과 함께 다음 반복 작업 티켓으로 이행

### 5.4 현재 예외 정합성(출처)
- `web/components/relay-app-shell.tsx`의 shell 예외는 `relay-contents`,
  `relay-rail-badges` 클래스로 이동했다.
- Settings 클라이언트와 settings fallback 페이지는 공통 primitive 기반으로
  전환되어 `React.CSSProperties`/`style={...}` 사용이 없다.
- 프로젝트 워크스페이스, 온보딩, 설정, 스타일메모리 화면은 공통
  primitive와 `relay-*` 클래스 기반으로 전환했다.
- 현재 허용된 주요 예외는 Style Memory의 `framer-motion` 카드 exit/enter
  상태, 승인 glow 애니메이션, diff 패널의 before/after 런타임 스타일처럼
  동적 상태를 직접 계산해야 하는 위치다.

확인 명령:

```bash
rg -n "React\\.CSSProperties|CSSProperties|style=\\{" web/app web/components
```

---

## 6) Layout Examples

### 6.1 Settings two-column form

```tsx
<RelayPageHead
  eyebrow="Transform"
  title="Claude provider"
  copy="Provider keys live in Settings, outside first-run onboarding."
  actions={<RelayStatusBadge>Settings only</RelayStatusBadge>}
/>
<div className="relay-settings-grid">
  <RelayCard variant="elevated">...</RelayCard>
  <RelayCard variant="elevated">...</RelayCard>
</div>
```

### 6.2 Project workspace with inspector

```tsx
<RelayAppShell
  activeStep="Face"
  projectHref={projectHref}
  railItems={railItems}
  inspector={<ProjectInspector />}
>
  <RelayPageHead title={projectName} actions={actions} />
  <RelayMetricTile label="Snapshots" value={snapshotCount} />
</RelayAppShell>
```

### 6.3 Tabbed review queue

```tsx
<RelayTabs
  aria-label="Proposal filters"
  items={[
    { label: "Proposals", count: pendingCount, active: state === "pending" },
    { label: "Approved", count: approvedCount, active: state === "approved" },
  ]}
/>
```

### 6.4 Dense trace list

```tsx
<RelayCard aria-label="Judgment trace list">
  <RelayListRow>
    <RelaySourceChip>design_handoff</RelaySourceChip>
    <RelayMetaGrid>...</RelayMetaGrid>
  </RelayListRow>
</RelayCard>
```

---

## 7) References

- UI kit implementation pass:
  `docs/superpowers/plans/2026-04-30-ui-kit-internal-screens-pixel-match.md`
- Hardening execution plan:
  `docs/superpowers/plans/2026-05-01-design-system-hardening.md`
- Current shell primitives:
  `web/components/relay-app-shell.tsx`
- New primitive exports:
  `web/components/relay/index.ts`
- Token and component CSS:
  `web/app/globals.css`
- DESIGN.md format specification:
  `https://github.com/google-labs-code/design.md`

---

## 8) 실행 체크리스트(요약)

- [x] `globals.css`의 `relay-*` 토큰/컴포넌트 섹션을 컴포넌트 계약 이름과 정렬
- [x] Shell/Settings 외 page-local CSS/스타일 패턴의 컴포넌트화
- [x] Settings → Onboarding → 프로젝트 워크스페이스 순으로 하드닝 확장
- [x] Variant/state/aria 규약 문서 승인 후 구현 반영
- [ ] 모바일 브레이크포인트 동작(1180/760) 회귀 점검
- [x] 남은 인라인 스타일 예외 범위와 제거 정책 문서화
