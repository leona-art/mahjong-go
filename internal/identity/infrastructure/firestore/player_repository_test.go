package firestore_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fs "cloud.google.com/go/firestore"

	"github.com/leona-art/mahjong-go/internal/identity/application"
	"github.com/leona-art/mahjong-go/internal/identity/domain"
	identityfs "github.com/leona-art/mahjong-go/internal/identity/infrastructure/firestore"
)

// newTestClient connects to a Firestore emulator. Start one locally with:
//
//	npx firebase-tools emulators:start --only firestore
//
// then run these tests with FIRESTORE_EMULATOR_HOST=127.0.0.1:8080 set (see
// firebase.json at the repo root for the configured port). The test skips
// itself when the emulator is not running so `go test ./...` still passes
// without it.
func newTestClient(t *testing.T) *fs.Client {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set; skipping Firestore emulator tests")
	}
	client, err := fs.NewClient(context.Background(), "demo-mahjong")
	if err != nil {
		t.Fatalf("fs.NewClient() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

var uidCounter atomic.Int64

// uniqueUID generates a Firestore-safe document ID. It deliberately does
// not use t.Name(), which contains "/" for subtests and is rejected by
// Firestore document IDs.
func uniqueUID(t *testing.T) domain.UID {
	t.Helper()
	return domain.UID(fmt.Sprintf("uid-%d-%d", time.Now().UnixNano(), uidCounter.Add(1)))
}

func TestPlayerRepository_CreateAndFindByUID(t *testing.T) {
	t.Run("作成したPlayerを同じ状態で取得できる", func(t *testing.T) {
		client := newTestClient(t)
		repo := identityfs.NewPlayerRepository(client)

		uid := uniqueUID(t)
		player, err := domain.NewPlayer(uid, "たろう")
		if err != nil {
			t.Fatalf("NewPlayer() error = %v", err)
		}

		if err := repo.Create(context.Background(), player); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		got, err := repo.FindByUID(context.Background(), uid)
		if err != nil {
			t.Fatalf("FindByUID() error = %v", err)
		}
		if got.UID() != player.UID() {
			t.Errorf("UID() = %q, want %q", got.UID(), player.UID())
		}
		if got.DisplayName() != player.DisplayName() {
			t.Errorf("DisplayName() = %q, want %q", got.DisplayName(), player.DisplayName())
		}
	})

	t.Run("存在しないPlayerの取得はErrPlayerNotFoundになる", func(t *testing.T) {
		client := newTestClient(t)
		repo := identityfs.NewPlayerRepository(client)

		_, err := repo.FindByUID(context.Background(), uniqueUID(t))
		if !errors.Is(err, application.ErrPlayerNotFound) {
			t.Errorf("FindByUID() error = %v, want ErrPlayerNotFound", err)
		}
	})

	t.Run("既に存在するuidでのCreateはErrPlayerAlreadyExistsになる", func(t *testing.T) {
		client := newTestClient(t)
		repo := identityfs.NewPlayerRepository(client)

		uid := uniqueUID(t)
		player, _ := domain.NewPlayer(uid, "たろう")
		if err := repo.Create(context.Background(), player); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		duplicate, _ := domain.NewPlayer(uid, "じろう")
		err := repo.Create(context.Background(), duplicate)
		if !errors.Is(err, application.ErrPlayerAlreadyExists) {
			t.Errorf("Create() error = %v, want ErrPlayerAlreadyExists", err)
		}
	})
}

func TestPlayerRepository_InvalidUID(t *testing.T) {
	t.Run("Createはスラッシュを含むuidを拒否する", func(t *testing.T) {
		repo := identityfs.NewPlayerRepository(nil)
		player, err := domain.NewPlayer("uid/with-slash", "たろう")
		if err != nil {
			t.Fatalf("NewPlayer() error = %v", err)
		}

		err = repo.Create(context.Background(), player)
		if err == nil || !strings.Contains(err.Error(), "invalid uid") {
			t.Fatalf("Create() error = %v, want invalid uid error", err)
		}
	})

	t.Run("FindByUIDはスラッシュを含むuidを拒否する", func(t *testing.T) {
		repo := identityfs.NewPlayerRepository(nil)

		_, err := repo.FindByUID(context.Background(), "uid/with-slash")
		if err == nil || !strings.Contains(err.Error(), "invalid uid") {
			t.Fatalf("FindByUID() error = %v, want invalid uid error", err)
		}
	})
}
