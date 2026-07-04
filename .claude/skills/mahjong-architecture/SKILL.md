---
name: mahjong-architecture
description: Domain-driven design knowledge for the mahjong-go server — bounded contexts (Identity, Matching, Game), layering conventions (domain/application/infrastructure), the Room aggregate, and the Firestore-relay realtime design. Use when designing or implementing anything under internal/, discussing aggregates, bounded contexts, matching/room/game domain logic, CQS command/query layering, or realtime delivery for this project.
---

# mahjong-go アーキテクチャ知識

このスキルは `mahjong-go` のドメイン設計に関する意思決定を要約する。詳細は
`docs/design/architecture.md` を参照すること。ドメイン層・アプリケーション層・
Matching/Gameコンテキストに関わる作業を始める前に、必ずこのファイルと
`docs/design/architecture.md` を読むこと。

## レイヤー構成（コンテキストごとに共通）

```
internal/<context>/domain/          # エンティティ、値オブジェクト、ドメインイベント、集約、リポジトリインターフェース
internal/<context>/application/     # Command/Queryユースケース（CQS）。ドメイン層のオーケストレーション
internal/<context>/infrastructure/  # Firestore実装、認証連携などの具体的な実装
```

- ドメイン層は他レイヤーに依存しない。リポジトリのインターフェースはドメイン層で宣言し、実装はインフラ層に置く
- CQSの分離はアプリケーション層でのみ意識する（ドメイン層のメソッド自体にCQSは適用しない）
- connect-goのハンドラ（`cmd/server`）はアプリケーション層を呼ぶだけの薄い層にする

## 主要な決定事項

- **認証**: 自前実装せず Identity Platform（Firebase Authentication）に委譲。connect-goのinterceptorでIDトークンを検証し`uid`を取得する
- **リアルタイム配信**: フロントエンドはFirestoreを直接購読しない。backendがFirestoreをwatchし、視点ごとに秘匿情報（他家の手牌など）をフィルタリングしてからConnectのServer Streamingで配信する。Firestoreへのアクセスはbackendのみに閉じる
- **境界づけられたコンテキスト**:
  - **Identity**: 認証・Playerプロフィール（uidに紐づく表示名・戦績等）
  - **Matching**: 対局開始前の「部屋」。招待制のみ（自動マッチメイキングはスコープ外）。中心はRoom集約
  - **Game**: 麻雀対局そのもの。Matchingからは席順（uidの配置）のみを受け取り、起家はGame側で開始後に決定する。コンテキスト間の連携はドメインイベント（`RoomStarted`）経由のみで、互いの永続化ストア/ドメインオブジェクトを直接参照しない

## Room集約（Matchingコンテキスト、実装済み: `internal/matching/domain`）

- 属性: `hostUID`, `seats`（4席、空 or uid）, `settings`（半荘/東風戦、持ち点）, `status`（`Waiting`→`Ready`→`Started`）
- 不変条件: hostは必ず着席／同一uidの重複着席不可／`Started`後は`seats`変更不可
- コマンド: `NewRoom` / `Join` / `Leave` / `Start`
- イベント: `RoomStarted{ RoomID, Seats }`

## 開発フェーズ

1. Phase 1: Identity Platform認証 + Matchingコンテキスト（Room集約）
2. Phase 2: Gameコンテキスト（麻雀ドメインロジック本体）
3. 以降: 対局履歴・統計等の参照系コンテキスト

新しいコンテキストや集約を追加した場合は、このファイルと `docs/design/architecture.md` の両方を更新すること。
