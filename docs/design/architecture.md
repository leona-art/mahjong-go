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
internal/<context>/domain/          # エンティティ、値オブジェクト、ドメインイベント、集約
internal/<context>/application/     # Command/Queryユースケース（CQS）。ドメイン層を呼び出すオーケストレーション、リポジトリインターフェース
internal/<context>/infrastructure/  # リポジトリ実装（Firestore等）、認証連携、外部サービス連携
```

- ドメイン層は他レイヤーに依存しない。永続化という概念そのものを一切知らない、純粋なビジネスルールのモデルとする（リポジトリインターフェースも置かない）
- リポジトリインターフェース（ポート）は**アプリケーション層**で宣言する。永続化を必要とするのはユースケースを実行するアプリケーション層であり、ドメイン層ではないため
- アプリケーション層はドメイン層に依存し、自身が宣言したインターフェース経由でインフラ層にアクセスする（依存性逆転）
- インフラ層はアプリケーション層のインターフェースを実装し、ドメイン層の型（エンティティ・値オブジェクト）を直接利用してよい
- connect-goのハンドラ（`cmd/server`配下）はアプリケーション層のCommand/Queryサービスを呼び出すだけの薄い層にする

> 既存の`internal/matching`（Room集約）は移行前の旧方式（リポジトリインターフェースをドメイン層で宣言）のままになっている。新規コードはこの節の方式に従うこと。詳細は本ドキュメント末尾の「既知のTODO」を参照。

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
- ユーザー登録・ログイン自体は自前実装しない。クライアントがFirebase Auth SDKで直接Identity Platformにサインアップ/ログインし、取得したIDトークンを`Authorization: Bearer <token>`でRPCに付与する

#### 認証（interceptor）

- `internal/identity/infrastructure/firebaseauth`が、Firebase Admin SDK (`firebase.google.com/go/v4/auth`) でIDトークンを検証するconnect-goの`connect.Interceptor`を提供する
- `NewInterceptor(verifier)`をハンドラの`connect.WithInterceptors(...)`に渡すと、リクエストの`Authorization`ヘッダを検証し、`uid`を`context`に埋め込んでから次のハンドラを呼ぶ。検証に失敗した場合は`connect.CodeUnauthenticated`を返す
- ハンドラ側は`firebaseauth.UIDFromContext(ctx)`で認証済みの`uid`を取り出す。Verifierはinterfaceなので、テストではFirebase Admin SDKに依存しないフェイクに差し替えられる
- `firebaseauth.NewClient(ctx, projectID)`はローカル開発用に`FIREBASE_AUTH_EMULATOR_HOST`環境変数を（Firebase Admin SDKの標準挙動どおり）自動的に尊重する

#### Player集約

- **識別子**: `UID`（Identity Platformが発行する、Matchingコンテキストと共通の識別子）
- **属性**: `displayName`（表示名、上限`MaxDisplayNameLength`文字）
- **不変条件**: `uid`必須、表示名は空不可・上限文字数以内
- **コマンド**: `NewPlayer(uid, displayName)`（domain）、`RegisterPlayer`（application, `PlayerCommandService`） — 登録は作成のみで、既に存在する`uid`の再登録は`ErrPlayerAlreadyRegistered`で拒否する（アップサートしない）
- **クエリ**: `GetPlayer(uid)` → `PlayerView`（application, `PlayerQueryService`）
- `PlayerRepository`インターフェース（`Create`/`FindByUID`、`ErrPlayerNotFound`/`ErrPlayerAlreadyExists`）は**アプリケーション層**（`internal/identity/application`）で宣言する。Player集約自体（`internal/identity/domain`）は永続化について何も知らない
- 実装: `internal/identity/domain`（Player集約のみ）、`internal/identity/application`（`PlayerCommandService`/`PlayerQueryService`/`PlayerRepository`インターフェース）、`internal/identity/infrastructure/firestore`（`PlayerRepository`のFirestore実装、`players`コレクション）、`internal/identity/infrastructure/firebaseauth`（IDトークン検証・interceptor）
- API: `proto/identity/v1/identity.proto`の`PlayerService`（`RegisterPlayer`/`GetPlayer`）。`RegisterPlayerRequest`に`uid`フィールドは無く、必ずinterceptorが検証したcontext上の`uid`を使う（なりすまし防止）。`GetPlayer`は他プレイヤーの表示名取得（部屋のメンバー表示など）のため明示的に`uid`を引数に取る

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
- 実装: `internal/matching/domain`（Room集約）、`internal/matching/application`（`RoomCommandService`/`RoomQueryService`）、`internal/matching/infrastructure/firestore`（`RoomRepository`のFirestore実装）

### 3. Game（対局）context
実際の麻雀対局を扱う。Room集約からは**席順（4人の`uid`配置）のみ**を受け取り、起家（誰が最初の親か）はGame開始後にGameコンテキスト側で決定する。RoomとGameは互いのドメインオブジェクト・永続化ストアを直接参照せず、`RoomStarted`イベント経由でのみ連携する。

詳細な集約設計（手牌・山・鳴き・役判定・点数計算など）は今後のドキュメントで別途扱う。

## 開発の進め方（フェーズ）

1. **Phase 1**: Identity Platformでの認証、Matchingコンテキスト（Room集約によるホスト作成・ゲスト参加の招待制マッチング）
2. **Phase 2**: Gameコンテキスト（麻雀対局そのもののドメインロジック）
3. 以降、対局履歴・統計等の参照系コンテキストは必要に応じて追加

現時点ではPhase 1のMatchingコンテキストのRoom集約（ドメイン層・アプリケーション層・Firestore実装）と、Identityコンテキストの認証（IDトークン検証interceptor）・Player集約（ドメイン層・アプリケーション層・Firestore実装）まで完了している。次はconnect-goサーバー（`cmd/server`）への実際のハンドラ配線と、部屋開始（`Start()`）をGameコンテキストへつなぐハンドオフの実装が残っている。

## ローカル動作確認

Firestore実装のテストは、実際のGCPプロジェクトではなく**Firestoreエミュレータ**に対して実行する。

```bash
# エミュレータ起動（初回はnpxが自動でダウンロードする。gcloud/firebase CLIのインストールは不要）
npx firebase-tools emulators:start --only firestore

# 別ターミナルでテスト実行
FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 go test ./...
```

- プロジェクトID・ポートはリポジトリルートの `firebase.json` / `.firebaserc` に固定してあるため、`--project`等のオプション指定は不要
- `FIRESTORE_EMULATOR_HOST`が未設定の場合、Firestore実装のテストは自動的にスキップされる（`go test ./...`は常に成功する）
- エミュレータはインメモリで動作し、プロセスを終了すればデータは消える。永続化やセキュリティルールの検証は対象外（本設計では認証済みbackendのみがFirestoreにアクセスするため、Security Rulesは現時点で不要）

Identity Platform（Firebase Authentication）連携も同様に、実プロジェクトではなく**Firebase Authエミュレータ**に対して動作確認できる（`firebase.json`にポート9099で設定済み）。

```bash
# Firestore + Authエミュレータをまとめて起動
npx firebase-tools emulators:start --only firestore,auth

# firebaseauth.NewClientはFIREBASE_AUTH_EMULATOR_HOSTを自動で尊重する
FIREBASE_AUTH_EMULATOR_HOST=127.0.0.1:9099 FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 go test ./...
```

## 既知のTODO

- **`internal/matching`（Room集約）のリポジトリインターフェース移行**: 「レイヤー構成」節で決めた新方式（リポジトリインターフェースはドメイン層ではなくアプリケーション層で宣言する）は現時点でIdentityコンテキストにのみ適用済み。Matchingコンテキストの`RoomRepository`は今も`internal/matching/domain/room_repository.go`にインターフェースが残ったままなので、後日`internal/matching/application`側に移し、`ErrRoomNotFound`も含めて移動したうえで`internal/matching/infrastructure/firestore`の参照先を更新する（Identityでの`PlayerRepository`移行と同じ手順）。
