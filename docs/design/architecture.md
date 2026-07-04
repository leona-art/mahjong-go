# アーキテクチャ設計

オンライン対戦麻雀サーバーの全体設計をまとめたドキュメント。実装前の設計議論の結果を記録し、以降の実装はここに従う。

## 全体方針

- **ドメイン駆動設計（DDD）**をベースに、ドメイン層を中心に据える
- ドメイン層自体にCQS（コマンド/クエリ分離）は持ち込まない。CQSの考え方は**アプリケーション層**でCommand/Queryのユースケースを分離する形で適用する
- 認証・永続化・外部サービス連携は**インフラ層**に寄せ、ドメイン層・アプリケーション層はそれらに依存しない
- API層は [Connect](https://connectrpc.com/)（connect-go）
- インフラは Google Cloud（Cloud Run + Firestore + Identity Platform を想定）

## レイヤー構成

各境界づけられたコンテキストは、以下の3層で構成する。

```
internal/<context>/domain/          # エンティティ、値オブジェクト、ドメインイベント、集約、リポジトリインターフェース
internal/<context>/application/     # Command/Queryユースケース（CQS）。ドメイン層を呼び出すオーケストレーション
internal/<context>/infrastructure/  # リポジトリ実装（Firestore等）、認証連携、外部サービス連携
```

- ドメイン層は他レイヤーに依存しない（インフラ層のインターフェースはドメイン層で宣言し、実装はインフラ層に置く）
- アプリケーション層はドメイン層のみに依存し、インフラ層には依存性逆転（インターフェース経由）でアクセスする
- connect-goのハンドラ（`cmd/server`配下）はアプリケーション層のCommand/Queryサービスを呼び出すだけの薄い層にする

## 認証

自前でユーザー登録・ログインは実装せず、**Identity Platform**（Firebase Authentication）に認証そのものを委譲する。

- connect-goのinterceptorでFirebase IDトークンを検証し、`uid`を取り出す
- 各コンテキストは`uid`をプレイヤーの識別子として参照するのみで、パスワード管理等は持たない
- プレイヤーの表示名・戦績等のプロフィール情報は別途Playerとして`uid`に紐づけて管理する（Identity/Playerコンテキスト、詳細は今後設計）

## リアルタイム配信

FirestoreをGCP上のリアルタイム基盤として使うが、**フロントエンドから直接Firestoreを購読しない**。

- ドメイン層・アプリケーション層は集約の状態更新のみを行う
- 状態はインフラ層（Repository実装）がFirestoreに永続化する
- backendがFirestoreの変更をwatchし、プレイヤーごとの視点でフィルタリング（秘匿情報の除去）したうえで、connect-goのServer StreamingでクライアントへPushする
- フロントエンドがFirestoreの認証情報を直接持つことはない。Firestoreへのアクセスはbackendのみ

この構成を選んだ理由（直接購読と比較したトレードオフ）:

| 観点 | フロント直接購読 | backend中継（採用） |
|---|---|---|
| レイテンシ・コスト | 有利（Google管理のFirestore listenerのみ） | backendがstreaming接続を保持するためコスト増、追加のホップで遅延増 |
| 秘匿情報（手牌等）の扱い | Security Rules + document分割が必要で複雑 | コードで一元的にフィルタリング可能。ドキュメント構造は素直な集約の形でよい |
| クライアント資格情報 | FirestoreのSDK認証情報をクライアントに配る必要がある | Firestoreアクセスはbackendのみに閉じる |

麻雀は「他家の手牌を見せない」という秘匿情報の扱いが必須要件のため、パフォーマンス面のデメリットを許容してbackend中継方式を採用する。

## 境界づけられたコンテキスト

### 1. Identity（認証・プレイヤー）context
- Identity Platformでの認証と、`uid`に紐づくPlayerプロフィール（表示名・戦績など）を扱う
- 詳細設計は今後

### 2. Matching（マッチング）context
対局を開始するまでの「部屋」を扱う。ランダムマッチメイキングは範囲外（スコープ外）とし、まずは**ホストが部屋を作成し、招待されたゲストが参加する**招待制のみをサポートする。

#### Room集約

- **識別子**: `RoomID`
- **属性**:
  - `hostUID`: 部屋を作成したプレイヤー
  - `seats`: 4席。各席は空 or プレイヤーの`uid`
  - `settings`: 対局設定（半荘/東風戦、持ち点など）
  - `status`: `Waiting` → `Ready`（4人揃った） → `Started`
- **不変条件**:
  - hostは必ずいずれかの席に着席する
  - 同一`uid`が複数の席に重複して着席することはない
  - `Started`後は`seats`を変更できない
- **コマンド（ふるまい）**: `NewRoom(hostUID, settings)` / `Join(uid)` / `Leave(uid)` / `Start()`
- **ドメインイベント**: `RoomStarted{ RoomID, Seats }` — Gameコンテキストが起動するトリガー

### 3. Game（対局）context
実際の麻雀対局を扱う。Room集約からは**席順（4人の`uid`配置）のみ**を受け取り、起家（誰が最初の親か）はGame開始後にGameコンテキスト側で決定する。RoomとGameは互いのドメインオブジェクト・永続化ストアを直接参照せず、`RoomStarted`イベント経由でのみ連携する。

詳細な集約設計（手牌・山・鳴き・役判定・点数計算など）は今後のドキュメントで別途扱う。

## 開発の進め方（フェーズ）

1. **Phase 1**: Identity Platformでの認証、Matchingコンテキスト（Room集約によるホスト作成・ゲスト参加の招待制マッチング）
2. **Phase 2**: Gameコンテキスト（麻雀対局そのもののドメインロジック）
3. 以降、対局履歴・統計等の参照系コンテキストは必要に応じて追加

現時点ではPhase 1のMatchingコンテキストのRoom集約（ドメイン層）から実装を開始する。
