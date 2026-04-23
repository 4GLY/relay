# Context Graph and Semantic Retrieval

Relay의 메모리 아키텍처를 확장하려는 개발자를 위한 리서치 요약입니다.
이 문서를 읽고 나면 최근 GraphRAG, agent memory, semantic retrieval 흐름이 무엇을 해결하려는지 이해하고, 현재 Relay와의 갭을 빠르게 판단할 수 있습니다.

## 현재 질문

Relay를 `API-first second-brain backend`에서 더 나아가, 장기 작업을 위한 `context graph + semantic retrieval + packet composer` 구조로 확장하려면 무엇을 참고해야 하는가?

이 문서는 그 질문에 답하기 위해 최근 자료를 세 덩어리로 정리합니다.

- graph 기반 retrieval과 global/local reasoning
- agent memory와 장기 기억 관리
- semantic retrieval 품질 향상 기법

## 핵심 결론

- `context graph`와 `semantic retrieval`은 대체 관계가 아닙니다.
- graph는 확정된 관계와 근거를 보존합니다.
- semantic retrieval은 graph 밖에 있는 관련 맥락을 찾아 graph를 보완합니다.
- Relay는 이미 graph의 씨앗은 있습니다.
  하지만 graph 조회, retrieval, ranking, eval 계층은 아직 거의 없습니다.
- 현재 Relay와 최근 연구의 갭은 `개념적으로는 중간`, `기능적으로는 큽니다`.

## 현재 제품 방향과의 연결

이 문서는 기술 베이스라인을 정리하는 문서입니다.
제품 wedge 자체를 정의하는 문서는 아닙니다.

현재 Relay의 더 날카로운 제품 방향은 다음과 같이 보는 편이 맞습니다.

- Relay의 목표는 generic memory backend가 아니라 `cross-agent style continuity`입니다.
- 더 정확히는, 같은 프로젝트 안에서 한 모델이나 세션이 하던 설계를 다른 모델 또는 다음 세션이 이어받을 때
  `사용자의 평소 판단 방식`까지 같이 이어지는 경험을 만드는 것입니다.
- 따라서 `context graph`, `semantic retrieval`, `packet composer`는 제품 그 자체가 아니라
  그 경험을 가능하게 하는 하부 구조입니다.

이 기준에서 보면:

- graph는 `무엇이 무엇에서 나왔는가`를 보존하는 층
- semantic retrieval은 `직접 연결은 없지만 지금 필요한 관련 맥락`을 찾는 층
- style memory는 `이 사용자가 보통 어떻게 판단하는가`를 보존하는 층
- handoff packet은 이 셋을 묶어 다음 agent에 전달하는 층

즉, Relay를 지금 시점에 가장 잘 설명하는 문장은
`context graph + semantic retrieval + style-aware handoff packet`에 더 가깝습니다.

## 1. GraphRAG와 graph 기반 retrieval

### 1.1. GraphRAG의 출발점

- Darren Edge 외, [From Local to Global: A Graph RAG Approach to Query-Focused Summarization](https://arxiv.org/abs/2404.16130), arXiv, 2024-04
  - global query는 단순 semantic retrieval만으로 답하기 어렵다는 문제를 정면으로 다룹니다.
  - entity/relationship graph, community hierarchy, summary를 함께 만들어 large corpus의 전역 질문에 답합니다.
  - Relay 입장에서는 `packet`을 단순 resume 요약이 아니라 graph-aware summary로 만들 수 있다는 힌트를 줍니다.

### 1.2. GraphRAG 전체 지형도

- Boci Peng 외, [Graph Retrieval-Augmented Generation: A Survey](https://arxiv.org/abs/2408.08921), arXiv, 2024-08
  - GraphRAG를 `Graph-Based Indexing`, `Graph-Guided Retrieval`, `Graph-Enhanced Generation`으로 나눠 설명합니다.
  - Relay처럼 아직 구조를 고르는 단계에서는 이 구분이 유용합니다.

- Haoyu Han 외, [Retrieval-Augmented Generation with Graphs (GraphRAG)](https://arxiv.org/abs/2501.00309), arXiv, 2025-01
  - query processor, retriever, organizer, generator, data source로 더 넓게 프레임을 잡습니다.
  - Relay에서 API와 packet build를 나중에 `query processor + organizer`로 분리할 때 참고하기 좋습니다.

### 1.3. 질문별 subgraph retrieval

- Mufei Li 외, [Simple Is Effective: The Roles of Graphs and Large Language Models in Knowledge-Graph-Based Retrieval-Augmented Generation](https://arxiv.org/abs/2410.20724), ICLR 2025
  - `SubgraphRAG` 계열입니다.
  - 전체 graph를 그대로 LLM에 넘기지 않고, 질문에 맞는 subgraph를 효율적으로 뽑는 데 집중합니다.
  - Relay에 그대로 대입하면 `GET /v1/projects/{id}/graph`보다 `질문별 subgraph projection`이 더 중요하다는 결론으로 이어집니다.

### 1.4. Graph를 탐색하는 agent

- Shilong Li 외, [GraphReader: Building Graph-based Agent to Enhance Long-Context Abilities of Large Language Models](https://aclanthology.org/2024.findings-emnlp.746/), Findings of EMNLP 2024
  - 긴 텍스트를 graph로 바꾸고, agent가 node와 neighbor를 읽으며 coarse-to-fine으로 탐색합니다.
  - Relay가 나중에 MCP tool이나 packet consumer를 통해 graph를 능동 탐색하게 만들고 싶다면 이 흐름이 직접 참고 대상입니다.

### 1.5. 공식 구현과 후속 흐름

- Microsoft Research, [Project GraphRAG](https://www.microsoft.com/en-us/research/project/graphrag/)
- Microsoft, [microsoft/graphrag](https://github.com/microsoft/graphrag)
- Microsoft Research, [Introducing DRIFT Search](https://www.microsoft.com/en-us/research/blog/introducing-drift-search-combining-global-and-local-search-methods-to-improve-quality-and-efficiency/)
  - DRIFT는 global summary와 local refinement를 같이 쓰는 질의 흐름입니다.
  - Relay의 이상적인 query planner는 이쪽에 더 가깝습니다.
  - 즉, 최근 decision과 open question만 보는 local mode와 프로젝트 전체를 훑는 global mode를 나누고, 필요하면 둘을 섞는 방식입니다.

## 2. Agent memory와 장기 기억

### 2.1. Graph를 장기 기억으로 쓰는 흐름

- Bernal Jiménez Gutiérrez 외, [HippoRAG: Neurobiologically Inspired Long-Term Memory for Large Language Models](https://arxiv.org/abs/2405.14831), arXiv, 2024-05
  - graph와 Personalized PageRank를 이용해 long-term memory retrieval을 구성합니다.
  - graph를 retrieval index로 쓴다는 점에서 Relay가 `canonical edge`를 retrieval에도 활용할 수 있음을 보여줍니다.

- Bernal Jiménez Gutiérrez 외, [From RAG to Memory: Non-Parametric Continual Learning for Large Language Models](https://proceedings.mlr.press/v267/gutierrez25a.html), ICML 2025
  - HippoRAG 2입니다.
  - 핵심 메시지는 `vector retrieval만으로는 인간형 장기 기억을 흉내 내기 어렵다`는 점입니다.
  - graph 구조를 넣는다고 모든 것이 해결되는 것도 아니고, factual memory가 떨어질 수 있다는 경고도 중요합니다.
  - Relay에 주는 시사점은 명확합니다.
    `graph를 넣더라도 passage retrieval과 factual evidence retrieval을 같이 설계해야 한다`는 뜻입니다.

### 2.2. Agentic memory와 동적 링크

- Wujiang Xu 외, [A-MEM: Agentic Memory for LLM Agents](https://arxiv.org/abs/2502.12110), NeurIPS 2025
  - Zettelkasten 스타일의 동적 연결과 memory evolution을 제안합니다.
  - 새 메모리가 들어오면 과거 메모리와의 연결을 찾고, 기존 표현도 갱신합니다.
  - Relay의 `inferred edge`, `memory evolution`, `same_theme` 같은 보조 관계 설계와 가장 가까운 자료입니다.

### 2.3. 계층형 memory management

- Jiazheng Kang 외, [Memory OS of AI Agent](https://aclanthology.org/2025.emnlp-main.1318/), EMNLP 2025
  - short-term, mid-term, long-term personal memory 계층을 둡니다.
  - graph 그 자체보다 memory lifecycle 관리가 중심입니다.
  - Relay가 장기적으로 `session context`, `project memory`, `cross-project memory`를 분리한다면 참고 가치가 큽니다.

- Yu Wang, Xi Chen, [MIRIX: Multi-Agent Memory System for LLM-Based Agents](https://arxiv.org/abs/2507.07957), arXiv, 2025-07
  - 아직 preprint지만, memory를 여러 타입으로 나누고 multi-agent orchestration을 넣습니다.
  - 지금 Relay에는 다소 이른 주제지만, 멀티모달 memory나 agent 협업으로 갈 경우 watchlist로 둘 만합니다.

## 3. Semantic retrieval 품질 향상

### 3.1. Contextual Retrieval

- Anthropic, [Introducing Contextual Retrieval](https://www.anthropic.com/engineering/contextual-retrieval), 2024-09-19
  - chunk를 그냥 embedding하지 않고, 문서 전체 문맥을 반영한 짧은 설명을 앞에 붙입니다.
  - embedding과 BM25를 함께 쓰고, reranking까지 붙이는 실무 흐름을 제안합니다.
  - Relay에서 design doc, handoff, repo-derived artifact를 chunking할 때 가장 바로 적용하기 쉽습니다.

### 3.2. Late Chunking

- Jina AI, [Late Chunking: Contextual Chunk Embeddings Using Long-Context Embedding Models](https://arxiv.org/abs/2409.04701), arXiv, 2024-09
  - 문서를 먼저 길게 읽고, 나중에 chunk representation을 뽑는 방식입니다.
  - chunk가 문맥을 잃는 문제를 줄이는 데 초점이 있습니다.
  - 문서 하나가 길고, 각 chunk가 혼자서는 의미가 약한 Relay artifact에 특히 잘 맞습니다.

### 3.3. Embedding 모델 선택

- BAAI, [M3-Embedding](https://arxiv.org/abs/2402.03216), arXiv, 2024-02
  - dense retrieval, sparse retrieval, multi-vector retrieval을 한 모델에서 다룹니다.
  - 다국어와 장문 retrieval을 같이 고려할 수 있습니다.
  - Relay가 한국어와 영어가 섞인 작업 기록을 다룰 가능성이 높다면 주목할 만합니다.

## 4. 평가와 운영

- Microsoft Research, [BenchmarkQED](https://www.microsoft.com/en-us/research/blog/benchmarkqed-automated-benchmarking-of-rag-systems/), 2025-06-05
  - local/global query를 나눠서 RAG를 평가하는 도구입니다.
  - Relay가 단순히 검색이 되는지보다, `resume packet`이 실제로 더 좋은 작업 재개를 돕는지 평가하려면 이런 관점이 필요합니다.

## 5. 현재 Relay가 이미 가진 것

현재 Relay는 완전히 빈 상태는 아닙니다.
아래 구성은 이미 graph의 씨앗으로 볼 수 있습니다.

- 제품 정의: [README](../../README.md)
  - Relay는 `API-first second-brain backend`로 정의되어 있습니다.
- 도메인 모델: [internal/domain/models.go](../../internal/domain/models.go)
  - `Project`, `Note`, `Artifact`, `Decision`, `OpenQuestion`, `Packet`이 있습니다.
- 관계 저장:
  - [migrations/0001_initial.sql](../../migrations/0001_initial.sql)
  - `decisions.source_note_ids`
  - `decisions.source_artifact_ids`
  - `open_questions.source_note_ids`
  - `open_questions.source_artifact_ids`
  - `packets.decision_ids`
  - `packets.open_question_ids`
  - `packets.source_artifact_ids`
- 서비스 흐름: [internal/services/services.go](../../internal/services/services.go)
  - `Capture`
  - `Promote`
  - `BuildPacket`

즉, 현재 Relay는 이미 아래 canonical graph를 암시합니다.

```text
Project
  -> Note
  -> Artifact
  -> Decision
  -> OpenQuestion
  -> Packet

Decision
  -> derived_from -> Note
  -> derived_from -> Artifact

OpenQuestion
  -> derived_from -> Note
  -> derived_from -> Artifact

Packet
  -> includes -> Decision
  -> includes -> OpenQuestion
  -> includes -> Artifact
```

하지만 중요한 공백도 분명합니다.

현재 Relay에는 아직 아래 층이 없습니다.

- 사용자의 판단 습관을 담는 `heuristic` 또는 `preference` 계층
- 판단이 어떤 근거에서 나왔는지 남기는 `judgment trace`
- 다음 agent에게 style cue를 함께 넘기는 `style-aware handoff packet`

## 6. 현재 Relay와 리서치 사이의 갭

### 한 줄 평가

- 개념적 갭: 중간
- 구현 갭: 큼

Relay는 이미 좋은 도메인 명사와 일부 관계를 갖고 있습니다.
하지만 최근 연구가 강조하는 `query-conditioned retrieval`, `hybrid local/global search`, `inferred edge`, `ranking`, `evaluation`은 거의 없습니다.

### 상세 갭 표

| 영역 | 현재 Relay | 최근 자료의 방향 | 갭 판단 | 다음 단계 |
| --- | --- | --- | --- | --- |
| Canonical graph | 관계가 JSON 배열 형태로 암시적으로 저장됨 | 명시적 edge, graph query, provenance 추적 | 중간 | `nodes/edges` projection 추가 |
| Graph query API | `show`, `build packet`만 있음 | query별 subgraph retrieval, graph traversal | 큼 | `GET /v1/projects/{id}/graph` 또는 `POST /v1/graph/query` 추가 |
| Semantic retrieval | 없음 | embedding, BM25, reranking, contextual chunking | 큼 | chunk 테이블과 hybrid retrieval 추가 |
| Inferred edge | 없음 | semantic similarity 기반 후보 관계와 승격 흐름 | 큼 | `inferred_edges` 저장소 추가 |
| Packet generation | 프로젝트 전체 요약에 가까움 | query-conditioned evidence selection과 ranking | 큼 | packet composer를 retrieval-aware로 분리 |
| Global/local search | 구분 없음 | GraphRAG, DRIFT처럼 질의 유형별 모드 분리 | 큼 | local/global query planner 설계 |
| Memory hierarchy | 프로젝트 단일 레벨 | short/mid/long-term, episodic/semantic 분리 | 중간 | session layer와 long-term layer 분리 검토 |
| Judgment continuity | 없음 | A-MEM류 memory evolution, user heuristic 축적, style-aware handoff | 큼 | `judgment_traces`, `heuristics`, `style-aware packet` 추가 |
| Evaluation | 사실상 수동 smoke test 중심 | local/global query benchmark, LLM judge, win rate | 큼 | 작은 내부 benchmark 세트 추가 |
| Explainability | 일부 source id만 있음 | retrieval 경로, edge provenance, why-this-context | 중간 | packet에 `why included` 필드 추가 |

## 7. 지금 Relay에 필요한 우선순위

최근 논문을 다 따라가려 하면 과합니다.
현재 Relay 기준으로는 아래 순서가 가장 현실적입니다.

### 0단계. 제품 계약을 먼저 고정하기

지금 Relay에서 제일 먼저 필요한 것은 graph DB가 아니라
`같은 프로젝트 안에서 model-to-model 또는 session-to-session handoff를 어떻게 증명할 것인가`에 대한 계약입니다.

최소 목표:

- approved heuristics
- same-project style-aware handoff packet
- 하나의 instrumented demo flow

이 계약이 먼저 고정돼야 그 뒤의 graph와 retrieval도
`무엇을 더 잘 찾기 위한 구조인지`가 분명해집니다.

### 1단계. Canonical graph를 먼저 드러내기

- 별도 graph DB부터 도입하지 않습니다.
- 현재 테이블을 그대로 읽어 `graph projection`을 만듭니다.
- 최소 목표:
  - project별 `nodes[]`
  - project별 `edges[]`
  - edge type: `derived_from`, `includes`

이 단계가 있어야 나중에 semantic retrieval 결과를 어디에 붙일지 기준이 생깁니다.

### 2단계. Semantic retrieval을 `graph 보완 계층`으로 넣기

- `note`, `artifact`, 필요하면 `decision` 본문을 chunking합니다.
- embedding만 넣지 말고 BM25나 lexical fallback도 같이 둡니다.
- 가능하면 contextual retrieval 또는 late chunking을 적용합니다.
- 다만 첫 demo에서는 artifact 크기가 작다면 whole-artifact handling도 허용할 수 있습니다.

이 단계의 목표는 `비슷한 맥락 찾기`입니다.
아직 graph의 공식 edge를 늘리는 단계는 아닙니다.

### 3단계. Inferred edge를 도입하기

- semantic retrieval 결과를 곧바로 canonical edge로 저장하지 않습니다.
- `related_to`, `possible_support`, `possible_answer` 같은 후보 관계로 저장합니다.
- score와 status를 둡니다.

이렇게 해야 explainability와 안정성을 같이 잡을 수 있습니다.

### 4단계. Packet composer를 retrieval-aware로 바꾸기

현재 `BuildPacket`은 프로젝트 전체를 훑어 summary를 만드는 성격이 강합니다.
다음 단계에서는 packet을 아래 입력으로 만들도록 바꾸는 게 좋습니다.

- 현재 질문 또는 작업 목표
- 직접 연결된 canonical node
- semantic retrieval로 찾은 관련 node
- approved heuristic
- recency, trust, importance ranking

여기서 packet은 단순 resume가 아니라
`state + evidence + style cue`를 함께 담는 handoff packet으로 진화해야 합니다.

### 5단계. 작은 eval 세트를 만들기

- 최소 20~50개 정도의 local/global query를 준비합니다.
- packet 품질과 retrieval 품질을 따로 봅니다.
- `정답을 찾았는가`만 보지 말고 `재개 가능성`, `근거 추적 가능성`도 같이 봅니다.
- Relay의 현재 wedge를 생각하면 `style continuity` eval도 별도로 둬야 합니다.
  예:
  - handoff 후 manual summary 없이 이어졌는가
  - approved heuristic이 실제로 packet에 재사용됐는가
  - 사용자가 continuation을 보고 `내 방식과 맞다`고 판단했는가

## 8. Relay 기준으로 보면 갭이 정말 큰가

`완전히 다른 제품을 다시 만들어야 할 정도`는 아닙니다.
하지만 `현재 구현이 연구 흐름을 이미 대부분 따라가고 있다`고 보기에는 무리입니다.

더 정확히 말하면 이렇습니다.

- 도메인 모델의 방향은 좋습니다.
- graph의 seed도 이미 있습니다.
- 하지만 retrieval engine과 query planner는 아직 거의 없습니다.
- 그리고 `사용자 판단 스타일`을 명시적으로 다루는 층도 아직 없습니다.
- 그래서 지금 Relay는 `memory backend v0`에는 가깝지만,
  `graph-aware memory system`이나 `personal operating system`이라고 부르기에는 아직 이릅니다.

실무 감각으로 정리하면:

- 모델링 갭은 중간
- 저장 구조 갭은 중간
- retrieval과 eval 갭은 큼
- packet intelligence 갭은 큼

## 9. 추천 읽기 순서

빠르게 감을 잡으려면 아래 순서로 읽는 것이 좋습니다.

1. `From Local to Global`
2. `Introducing DRIFT Search`
3. `From RAG to Memory`
4. `A-MEM`
5. `Introducing Contextual Retrieval`
6. `BenchmarkQED`

## 10. 확인 메모

- 이 문서는 2026-04-22 기준으로 정리했습니다.
- 논문은 주로 arXiv와 ACL/PMLR 같은 1차 출처를 사용했습니다.
- `MIRIX`는 아직 preprint이므로 watchlist 성격으로만 다뤘습니다.
- 현재 Relay 상태 평가는 이 저장소의 [README](../../README.md), [docs/api.md](../api.md), [docs/mcp.md](../mcp.md), [internal/domain/models.go](../../internal/domain/models.go), [internal/services/services.go](../../internal/services/services.go), [migrations/0001_initial.sql](../../migrations/0001_initial.sql)을 기준으로 했습니다.
