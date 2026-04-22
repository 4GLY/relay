# Relay Research Notes

이 디렉터리는 Relay의 메모리 아키텍처를 확장할 때 참고할 외부 리서치와 내부 해석을 모아둔 곳입니다.
지금은 `context graph`, `semantic retrieval`, `agent memory`를 중심으로 정리합니다.

## 문서 목록

- [Context Graph and Semantic Retrieval](./context-graph-and-semantic-retrieval.md)
  - 최근 논문과 공식 자료를 묶어 설명합니다.
  - 현재 Relay 구성과의 차이, 우선 구현 순서를 정리합니다.

## 읽는 순서

1. `Context Graph and Semantic Retrieval` 문서를 읽고 현재 Relay가 어디까지 와 있는지 확인합니다.
2. 설계 변경을 시작할 때는 먼저 `canonical graph`, `inferred retrieval`, `packet generation` 중 어느 층을 건드릴지 정합니다.
3. 실제 구현 전에는 이 리서치 노트와 현재 코드가 여전히 맞는지 다시 확인합니다.

## 유지 원칙

- 논문은 가능한 한 1차 출처를 우선 사용합니다.
- 공식 구현 자료가 있으면 논문과 함께 남깁니다.
- 이 디렉터리의 문서는 설명 문서입니다.
- 구현 계약은 계속 [API 계약](../api.md)과 [MCP 문서](../mcp.md), 그리고 코드가 소스 오브 트루스입니다.
