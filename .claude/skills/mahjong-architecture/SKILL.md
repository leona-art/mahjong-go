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
internal/<context>/domain/          # エンティティ、値オブジェクト、ドメインイベント、集約
internal/<context>/application/     # Command/Queryユースケース（CQS）。ドメイン層のオーケストレーション、リポジトリインターフェース
internal/<context>/infrastructure/  # Firestore実装、認証連携などの具体的な実装
```

- ドメイン層は他レイヤーに依存しない。永続化という概念そのものを一切知らない純粋なビジネスルールのモデルとする（リポジトリインターフェースも置かない）
- リポジトリインターフェース（ポート）は**アプリケーション層**で宣言し、実装はインフラ層に置く。永続化を必要とするのはユースケース（アプリケーション層）でありドメイン層ではないため
- CQSの分離はアプリケーション層でのみ意識する（ドメイン層のメソッド自体にCQSは適用しない）
- connect-goのハンドラ（`cmd/server`）はアプリケーション層を呼ぶだけの薄い層にする
- **移行中**: `internal/matching`（Room集約）はまだ旧方式（`RoomRepository`インターフェースがdomain層の`internal/matching/domain/room_repository.go`にある）。`internal/identity`（Player集約）は新方式に移行済み。Matchingの移行は未着手のTODO（下記「開発フェーズ」参照）

## 主要な決定事項

- **認証**: 自前実装せず Identity Platform（Firebase Authentication）に委譲。connect-goのinterceptorでIDトークンを検証し`uid`を取得する
- **リアルタイム配信**: フロントエンドはFirestoreを直接購読しない。backendがFirestoreをwatchし、視点ごとに秘匿情報（他家の手牌など）をフィルタリングしてからConnectのServer Streamingで配信する。Firestoreへのアクセスはbackendのみに閉じる
- **境界づけられたコンテキスト**:
  - **Identity**: 認証・Playerプロフィール（uidに紐づく表示名・戦績等）
  - **Matching**: 対局開始前の「部屋」。招待制のみ（自動マッチメイキングはスコープ外）。中心はRoom集約
  - **Game**: 麻雀対局そのもの。Matchingからは席順（uidの配置）のみを受け取り、起家はGame側で開始後に決定する。コンテキスト間の連携はドメインイベント（`RoomStarted`）経由のみで、互いの永続化ストア/ドメインオブジェクトを直接参照しない

## Room集約（Matchingコンテキスト、実装済み）

- 属性: `hostUID`, `seats`（4席、空 or uid）, `settings`（半荘/東風戦、持ち点）, `status`（`Waiting`→`Ready`→`Started`）
- 不変条件: hostは必ず着席／同一uidの重複着席不可／`Started`後は`seats`変更不可
- コマンド: `NewRoom` / `Join` / `Leave` / `Start`（domain）、`CreateRoom` / `JoinRoom` / `LeaveRoom`（application, `RoomCommandService`）
- クエリ: `GetRoom` → `RoomView`（application, `RoomQueryService`）
- イベント: `RoomStarted{ RoomID, Seats }`
- 実装場所: `internal/matching/domain`, `internal/matching/application`, `internal/matching/infrastructure/firestore`

## Player集約と認証（Identityコンテキスト、実装済み）

- ユーザー登録・ログイン自体は自前実装しない。クライアントがFirebase Auth SDKで直接Identity Platformにサインアップ/ログインし、IDトークンを`Authorization: Bearer <token>`で付与してRPCを呼ぶ
- **認証**: `internal/identity/infrastructure/firebaseauth`がFirebase Admin SDKでIDトークンを検証する`connect.Interceptor`（`NewInterceptor(verifier)`）を提供する。検証成功で`uid`を`context`に埋め込み（`UIDFromContext`で取得）、失敗時は`connect.CodeUnauthenticated`を返す。`Verifier`はinterfaceなのでテストはFirebase Admin SDK非依存のフェイクに差し替えられる。ローカル開発では`firebaseauth.NewClient`が`FIREBASE_AUTH_EMULATOR_HOST`を自動で尊重する
- **Player集約**: 属性は`uid`（識別子）と`displayName`（表示名、上限`MaxDisplayNameLength`文字）
  - コマンド: `NewPlayer(uid, displayName)`（domain）、`RegisterPlayer`（application, `PlayerCommandService`） — 登録は作成のみで、既存`uid`の再登録は`ErrPlayerAlreadyRegistered`で拒否（アップサートしない）
  - クエリ: `GetPlayer(uid)` → `PlayerView`（application, `PlayerQueryService`）
  - `PlayerRepository`インターフェース（`Create`/`FindByUID`、`ErrPlayerNotFound`/`ErrPlayerAlreadyExists`）は`internal/identity/application`で宣言する。`internal/identity/domain`はPlayer集約のみを持ち、永続化については何も知らない
  - 実装場所: `internal/identity/domain`（Player集約）, `internal/identity/application`（Command/Queryサービス、`PlayerRepository`インターフェース）, `internal/identity/infrastructure/firestore`（`players`コレクション）, `internal/identity/infrastructure/firebaseauth`
  - API: `proto/identity/v1/identity.proto`の`PlayerService`。`RegisterPlayerRequest`に`uid`フィールドは無く、必ずinterceptorが検証済みcontext上の`uid`を使う（なりすまし防止）。`GetPlayer`は他プレイヤーの表示名取得のため明示的に`uid`を引数に取る

## Firestore / Auth エミュレータでのローカル動作確認

Firestore実装（`internal/*/infrastructure/firestore`）とFirebase Auth連携（`internal/identity/infrastructure/firebaseauth`）のテストは実プロジェクトではなくエミュレータに対して実行する。リポジトリルートの`firebase.json`/`.firebaserc`にプロジェクトID(`demo-mahjong`)とポート(Firestore: 8080, Auth: 9099)が固定してある。

```bash
npx firebase-tools emulators:start --only firestore,auth
FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 go test ./...
```

各エミュレータ用の環境変数が未設定時は該当テストが自動スキップされるので、エミュレータなしでも`go test ./...`は通る。詳細は`docs/design/architecture.md`の「ローカル動作確認」セクション参照。

## 開発フェーズ

1. Phase 1: Identity Platform認証（IDトークン検証interceptor）+ Player集約、Matchingコンテキスト（Room集約） — 実装済み。残: `cmd/server`への実ハンドラ配線とGameコンテキストへのハンドオフ
2. Phase 2: Gameコンテキスト（麻雀ドメインロジック本体）
3. 以降: 対局履歴・統計等の参照系コンテキスト

## 既知のTODO

- `internal/matching`の`RoomRepository`インターフェースをdomain層（`internal/matching/domain/room_repository.go`）からapplication層に移動し、Identityの`PlayerRepository`と同じ構成に揃える（`ErrRoomNotFound`も含めて移動、`internal/matching/infrastructure/firestore`の参照先を更新）。詳細は`docs/design/architecture.md`の「既知のTODO」参照

新しいコンテキストや集約を追加した場合は、このファイルと `docs/design/architecture.md` の両方を更新すること。
