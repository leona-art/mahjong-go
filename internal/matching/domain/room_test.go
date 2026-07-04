package domain_test

import (
	"errors"
	"testing"

	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

func newTestSettings() domain.RoomSettings {
	return domain.RoomSettings{GameLength: domain.GameLengthHanchan, InitialScore: 25000}
}

func TestNewRoom(t *testing.T) {
	t.Run("host is seated in the first seat", func(t *testing.T) {
		room, err := domain.NewRoom("room-1", "host", newTestSettings())
		if err != nil {
			t.Fatalf("NewRoom() error = %v", err)
		}
		if got := room.Seats()[0]; got != "host" {
			t.Errorf("Seats()[0] = %q, want %q", got, "host")
		}
		if room.Status() != domain.RoomStatusWaiting {
			t.Errorf("Status() = %v, want RoomStatusWaiting", room.Status())
		}
	})

	t.Run("rejects empty host uid", func(t *testing.T) {
		if _, err := domain.NewRoom("room-1", "", newTestSettings()); !errors.Is(err, domain.ErrHostUIDRequired) {
			t.Errorf("NewRoom() error = %v, want ErrHostUIDRequired", err)
		}
	})
}

func TestRoom_Join(t *testing.T) {
	t.Run("becomes ready once all seats are filled", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			if err := room.Join(uid); err != nil {
				t.Fatalf("Join(%q) error = %v", uid, err)
			}
		}
		if room.Status() != domain.RoomStatusReady {
			t.Errorf("Status() = %v, want RoomStatusReady", room.Status())
		}
	})

	t.Run("rejects joining a full room", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		if err := room.Join("p5"); !errors.Is(err, domain.ErrRoomFull) {
			t.Errorf("Join() error = %v, want ErrRoomFull", err)
		}
	})

	t.Run("rejects duplicate uid", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		if err := room.Join("host"); !errors.Is(err, domain.ErrAlreadyJoined) {
			t.Errorf("Join() error = %v, want ErrAlreadyJoined", err)
		}
	})

	t.Run("rejects joining a started room", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		if _, err := room.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if err := room.Join("p5"); !errors.Is(err, domain.ErrRoomStarted) {
			t.Errorf("Join() error = %v, want ErrRoomStarted", err)
		}
	})
}

func TestRoom_Leave(t *testing.T) {
	t.Run("frees the seat and reverts to waiting", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		if err := room.Leave("p2"); err != nil {
			t.Fatalf("Leave() error = %v", err)
		}
		if room.Status() != domain.RoomStatusWaiting {
			t.Errorf("Status() = %v, want RoomStatusWaiting", room.Status())
		}
		if err := room.Join("p2"); err != nil {
			t.Errorf("Join() after Leave() error = %v", err)
		}
	})

	t.Run("host cannot leave", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		if err := room.Leave("host"); !errors.Is(err, domain.ErrHostCannotLeave) {
			t.Errorf("Leave() error = %v, want ErrHostCannotLeave", err)
		}
	})

	t.Run("rejects leaving a player not in the room", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		if err := room.Leave("nobody"); !errors.Is(err, domain.ErrPlayerNotInRoom) {
			t.Errorf("Leave() error = %v, want ErrPlayerNotInRoom", err)
		}
	})

	t.Run("rejects leaving a started room", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		if _, err := room.Start(); err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if err := room.Leave("p2"); !errors.Is(err, domain.ErrRoomStarted) {
			t.Errorf("Leave() error = %v, want ErrRoomStarted", err)
		}
	})
}

func TestRoom_Start(t *testing.T) {
	t.Run("rejects starting before all seats are filled", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		if _, err := room.Start(); !errors.Is(err, domain.ErrSeatsNotFilled) {
			t.Errorf("Start() error = %v, want ErrSeatsNotFilled", err)
		}
	})

	t.Run("emits RoomStarted with the final seating", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		event, err := room.Start()
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if event.RoomID != room.ID() {
			t.Errorf("event.RoomID = %v, want %v", event.RoomID, room.ID())
		}
		if event.Seats != room.Seats() {
			t.Errorf("event.Seats = %v, want %v", event.Seats, room.Seats())
		}
		if room.Status() != domain.RoomStatusStarted {
			t.Errorf("Status() = %v, want RoomStatusStarted", room.Status())
		}
	})

	t.Run("rejects starting an already started room", func(t *testing.T) {
		room, _ := domain.NewRoom("room-1", "host", newTestSettings())
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = room.Join(uid)
		}
		_, _ = room.Start()
		if _, err := room.Start(); !errors.Is(err, domain.ErrRoomStarted) {
			t.Errorf("Start() error = %v, want ErrRoomStarted", err)
		}
	})
}
