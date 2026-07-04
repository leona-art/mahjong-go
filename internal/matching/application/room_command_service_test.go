package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leona-art/mahjong-go/internal/matching/application"
	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

func TestRoomCommandService_CreateRoom(t *testing.T) {
	t.Run("部屋を作成しホストを着席させる", func(t *testing.T) {
		service, repo := newTestCommandService()

		roomID, err := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID:  "host",
			Settings: newTestSettings(),
		})
		if err != nil {
			t.Fatalf("CreateRoom() error = %v", err)
		}

		room, err := repo.FindByID(context.Background(), roomID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got := room.Seats()[0]; got != "host" {
			t.Errorf("Seats()[0] = %q, want %q", got, "host")
		}
	})

	t.Run("host uidが空の場合はエラーになる", func(t *testing.T) {
		service, _ := newTestCommandService()

		_, err := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID:  "",
			Settings: newTestSettings(),
		})
		if !errors.Is(err, domain.ErrHostUIDRequired) {
			t.Errorf("CreateRoom() error = %v, want ErrHostUIDRequired", err)
		}
	})
}

func TestRoomCommandService_JoinRoom(t *testing.T) {
	t.Run("空席に参加できる", func(t *testing.T) {
		service, repo := newTestCommandService()
		roomID, _ := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID: "host", Settings: newTestSettings(),
		})

		if err := service.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: roomID, UID: "guest"}); err != nil {
			t.Fatalf("JoinRoom() error = %v", err)
		}

		room, _ := repo.FindByID(context.Background(), roomID)
		if got := room.Seats()[1]; got != "guest" {
			t.Errorf("Seats()[1] = %q, want %q", got, "guest")
		}
	})

	t.Run("存在しない部屋への参加はエラーになる", func(t *testing.T) {
		service, _ := newTestCommandService()

		err := service.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: "no-such-room", UID: "guest"})
		if !errors.Is(err, domain.ErrRoomNotFound) {
			t.Errorf("JoinRoom() error = %v, want ErrRoomNotFound", err)
		}
	})

	t.Run("満席の部屋への参加は拒否される", func(t *testing.T) {
		service, _ := newTestCommandService()
		roomID, _ := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID: "host", Settings: newTestSettings(),
		})
		for _, uid := range []domain.UID{"p2", "p3", "p4"} {
			_ = service.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: roomID, UID: uid})
		}

		err := service.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: roomID, UID: "p5"})
		if !errors.Is(err, domain.ErrRoomFull) {
			t.Errorf("JoinRoom() error = %v, want ErrRoomFull", err)
		}
	})
}

func TestRoomCommandService_LeaveRoom(t *testing.T) {
	t.Run("参加者が退出すると席が空く", func(t *testing.T) {
		service, repo := newTestCommandService()
		roomID, _ := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID: "host", Settings: newTestSettings(),
		})
		_ = service.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: roomID, UID: "guest"})

		if err := service.LeaveRoom(context.Background(), application.LeaveRoomCommand{RoomID: roomID, UID: "guest"}); err != nil {
			t.Fatalf("LeaveRoom() error = %v", err)
		}

		room, _ := repo.FindByID(context.Background(), roomID)
		if got := room.Seats()[1]; got != "" {
			t.Errorf("Seats()[1] = %q, want empty", got)
		}
	})

	t.Run("hostの退出は拒否される", func(t *testing.T) {
		service, _ := newTestCommandService()
		roomID, _ := service.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID: "host", Settings: newTestSettings(),
		})

		err := service.LeaveRoom(context.Background(), application.LeaveRoomCommand{RoomID: roomID, UID: "host"})
		if !errors.Is(err, domain.ErrHostCannotLeave) {
			t.Errorf("LeaveRoom() error = %v, want ErrHostCannotLeave", err)
		}
	})
}
