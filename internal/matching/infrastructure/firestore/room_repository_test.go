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

	"github.com/leona-art/mahjong-go/internal/matching/application"
	"github.com/leona-art/mahjong-go/internal/matching/domain"
	matchingfs "github.com/leona-art/mahjong-go/internal/matching/infrastructure/firestore"
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

var roomIDCounter atomic.Int64

// uniqueRoomID generates a Firestore-safe document ID. It deliberately does
// not use t.Name(), which contains "/" for subtests and is rejected by
// Firestore document IDs.
func uniqueRoomID(t *testing.T) domain.RoomID {
	t.Helper()
	return domain.RoomID(fmt.Sprintf("room-%d-%d", time.Now().UnixNano(), roomIDCounter.Add(1)))
}

func TestRoomRepository_SaveAndFindByID(t *testing.T) {
	t.Run("保存した部屋を同じ状態で取得できる", func(t *testing.T) {
		client := newTestClient(t)
		repo := matchingfs.NewRoomRepository(client)

		room, err := domain.NewRoom(uniqueRoomID(t), "host", domain.RoomSettings{
			GameLength:   domain.GameLengthHanchan,
			InitialScore: 25000,
		})
		if err != nil {
			t.Fatalf("NewRoom() error = %v", err)
		}
		if err := room.Join("guest"); err != nil {
			t.Fatalf("Join() error = %v", err)
		}

		if err := repo.Save(context.Background(), room); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		got, err := repo.FindByID(context.Background(), room.ID())
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got.HostUID() != room.HostUID() {
			t.Errorf("HostUID() = %q, want %q", got.HostUID(), room.HostUID())
		}
		if got.Seats() != room.Seats() {
			t.Errorf("Seats() = %v, want %v", got.Seats(), room.Seats())
		}
		if got.Settings() != room.Settings() {
			t.Errorf("Settings() = %v, want %v", got.Settings(), room.Settings())
		}
		if got.Status() != room.Status() {
			t.Errorf("Status() = %v, want %v", got.Status(), room.Status())
		}
	})

	t.Run("存在しない部屋の取得はErrRoomNotFoundになる", func(t *testing.T) {
		client := newTestClient(t)
		repo := matchingfs.NewRoomRepository(client)

		_, err := repo.FindByID(context.Background(), uniqueRoomID(t))
		if !errors.Is(err, application.ErrRoomNotFound) {
			t.Errorf("FindByID() error = %v, want ErrRoomNotFound", err)
		}
	})

	t.Run("Saveで更新した状態が反映される", func(t *testing.T) {
		client := newTestClient(t)
		repo := matchingfs.NewRoomRepository(client)

		room, _ := domain.NewRoom(uniqueRoomID(t), "host", domain.RoomSettings{
			GameLength:   domain.GameLengthTonpuusen,
			InitialScore: 25000,
		})
		if err := repo.Save(context.Background(), room); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			if err := room.Join(uid); err != nil {
				t.Fatalf("Join(%q) error = %v", uid, err)
			}
		}
		if err := repo.Save(context.Background(), room); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		got, err := repo.FindByID(context.Background(), room.ID())
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got.Status() != domain.RoomStatusReady {
			t.Errorf("Status() = %v, want RoomStatusReady", got.Status())
		}
	})
}

func TestRoomRepository_InvalidRoomID(t *testing.T) {
	t.Run("Saveはスラッシュを含むRoomIDを拒否する", func(t *testing.T) {
		repo := matchingfs.NewRoomRepository(nil)
		room, err := domain.NewRoom("room/with-slash", "host", domain.RoomSettings{
			GameLength:   domain.GameLengthHanchan,
			InitialScore: 25000,
		})
		if err != nil {
			t.Fatalf("NewRoom() error = %v", err)
		}

		err = repo.Save(context.Background(), room)
		if err == nil || !strings.Contains(err.Error(), "invalid room id") {
			t.Fatalf("Save() error = %v, want invalid room id error", err)
		}
	})

	t.Run("FindByIDはスラッシュを含むRoomIDを拒否する", func(t *testing.T) {
		repo := matchingfs.NewRoomRepository(nil)

		_, err := repo.FindByID(context.Background(), "room/with-slash")
		if err == nil || !strings.Contains(err.Error(), "invalid room id") {
			t.Fatalf("FindByID() error = %v, want invalid room id error", err)
		}
	})
}
