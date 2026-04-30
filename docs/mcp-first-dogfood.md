# MCP-First Dogfood Workflow

이 문서는 Relay를 실제 작업 기억으로 쓰는 에이전트 운영 절차를 정의한다.
핵심 원칙은 MCP가 작업 기억의 기본 입출력이고, Web은 사람이 검토해야 하는
큐와 시각화 표면을 맡는다는 것이다.

## Operating Model

Relay의 dogfood 흐름은 세 표면을 분리한다.

- MCP: 에이전트가 작업 중 캡처, 승격, 검색, packet 생성을 수행하는 기본 경로
- Web: 사람이 Style Memory, Project Explorer, Decision Graph, Packet Builder를 검토하는 경로
- Public snapshot: 외부 공유나 다음 세션 재개에 쓸 고정 handoff 산출물

MCP는 자동화에 적합한 좁은 도구 표면이다. Style Memory 제안 생성, 승인,
거절, provider key 관리는 의도적으로 공개 MCP 도구가 아니다. 이 작업들은
HTTP API, Web, 또는 운영자용 로컬 도구에서 다룬다.

## Setup

원격 MCP 엔드포인트:

```bash
export RELAY_BASE_URL="https://relay.4gly.dev"
export RELAY_CLIENT_TOKEN="issued-client-token"
```

MCP 호출은 `POST https://relay.4gly.dev/mcp`로 들어간다. 기존 MCP 소비자와의
호환을 위해 `RELAY_MCP_TOKEN`도 사용할 수 있지만, 값은 동일하게 발급된
client token이어야 한다.

로컬 stdio MCP를 직접 붙일 때:

```bash
go run ./cmd/relay-mcp
```

토큰 발급과 로컬 검증은 repo-owned skill wrapper가 기본 경로다.

```bash
./skills/relay-api-agent/scripts/setup.sh
./skills/relay-api-agent/scripts/relay-api.sh doctor
```

웹에 로그인한 사용자는 `/settings/api-keys`에서 자기 계정에 묶인 Relay
client token을 발급할 수도 있다. 이 키는 API/MCP 접근용이다. Anthropic 같은
외부 provider key는 `/settings/providers`에서 따로 관리한다.

## Daily Flow

1. 작업 시작 전에 `relay_retrieve_project`로 현재 task와 관련된 기억을 가져온다.
2. 작업 중 중요한 판단, 제약, 근거가 생기면 `relay_capture`로 남긴다.
3. 재사용 가능한 결정이나 열린 질문은 `relay_promote`로 승격한다.
4. 작업 종료 전에 `relay_build_packet`을 `persist_snapshot: true`로 호출한다.
5. 다음 세션은 `relay_latest_packet_snapshot`으로 같은 snapshot을 다시 연다.
6. 사람이 Web에서 Style Memory, Decision Graph, Packet Builder를 확인한다.

자동 write에는 항상 `idempotency_key`를 넣는다. 같은 작업을 재시도할 때
중복 note나 decision이 쌓이는 것을 막기 위한 운영 규칙이다.

## Good Captures

좋은 capture는 나중에 agent가 바로 행동할 수 있는 형태여야 한다.

- 결정: 무엇을 선택했고, 왜 선택했으며, 어떤 대안을 버렸는지 남긴다.
- 제약: 토큰, 라우팅, 배포, 보안, 데이터 소유권처럼 변경하기 어려운 조건을 남긴다.
- 근거: PR, 파일 경로, API 응답, QA run id, Argo revision 같은 확인 가능한 출처를 붙인다.
- 다음 행동: 다음 세션에서 바로 시작할 수 있는 구체적인 next step을 남긴다.

너무 큰 회고를 한 번에 저장하기보다 의미 있는 판단 단위로 나누는 편이 좋다.
Packet Builder가 필요한 근거를 고르기 쉽고, Decision Graph에서도 출처가 선명해진다.

## Web Review Loop

MCP가 기억을 쓰고 읽는 기본 경로라면, Web은 사람이 결과를 확인하는 경로다.

- Style Memory: pending proposal을 승인하거나 거절한다.
- Project Explorer: 프로젝트의 trace, proposal, packet, snapshot 요약을 확인한다.
- Decision Graph: decision, open question, artifact, packet snapshot의 연결을 확인한다.
- Packet Builder: 다음 agent handoff가 충분히 구체적인지 확인한다.

Style Memory approve/reject는 되돌릴 수 있는 편집 흐름이 아니라 review 결과를
기록하는 흐름이다. 잘못 승인한 내용은 새 proposal이나 heuristic update로
교정한다.

## Public Snapshot Boundary

`relay_build_packet`은 snapshot을 만들 수 있지만, public publish/revoke는
운영자 또는 HTTP API 책임이다. 공개 snapshot은 다음 경우에만 쓴다.

- 다른 환경의 agent에게 고정 handoff를 전달할 때
- QA나 리뷰에서 같은 packet 본문을 다시 열어야 할 때
- 외부 공개 가능한 상태로 정제된 산출물을 공유할 때

민감한 provider key, 개인 token, 내부-only URL은 capture와 packet body에 넣지 않는다.

## Verification

원격 MCP 상태 확인:

```bash
RELAY_BASE_URL=https://relay.4gly.dev ./scripts/canary.sh
```

단일 도구 호출 확인:

```bash
./examples/mcp/http/call-tool.sh relay_health '{}'
```

canonical handoff 회귀 확인:

```bash
./scripts/acceptance/v1_canonical_handoff.sh
```

마지막 스크립트는 정상 client token과 테스트 데이터 생성 권한이 필요하다.
문서만 바뀐 경우에는 `git diff --check`와 canary 수준 검증이면 충분하다.

## Common Failures

- `401 unauthorized`: `RELAY_CLIENT_TOKEN` 또는 `RELAY_MCP_TOKEN`이 발급된 client token인지 확인한다.
- `project not found`: 사람이 보는 Web URL의 `project=` 값과 MCP 호출의 `project_id` 또는 `project` 값이 같은지 확인한다.
- latest snapshot 없음: 먼저 `relay_build_packet`을 `persist_snapshot: true`로 호출했는지 확인한다.
- Style Memory 변경이 MCP에 없음: 정상이다. 공개 MCP는 style-memory mutation을 제공하지 않는다.
