package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leona-art/mahjong-go/internal/matching/application"
	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

func TestRoomQueryService_GetRoom(t *testing.T) {
	t.Run("部屋の現在の状態を取得できる", func(t *testing.T) {
		repo := newFakeRoomRepository()
		commandService := application.NewRoomCommandService(repo, func() domain.RoomID { return "room-1" })
		queryService := application.NewRoomQueryService(repo)

		roomID, _ := commandService.CreateRoom(context.Background(), application.CreateRoomCommand{
			HostUID: "host", Settings: newTestSettings(),
		})
		_ = commandService.JoinRoom(context.Background(), application.JoinRoomCommand{RoomID: roomID, UID: "guest"})

		view, err := queryService.GetRoom(context.Background(), roomID)
		if err != nil {
			t.Fatalf("GetRoom() error = %v", err)
		}
		if view.Status != domain.RoomStatusWaiting {
			t.Errorf("Status = %v, want RoomStatusWaiting", view.Status)
		}
		if got := view.Seats[1]; got != "guest" {
			t.Errorf("Seats[1] = %q, want %q", got, "guest")
		}
	})

	t.Run("存在しない部屋の取得はエラーになる", func(t *testing.T) {
		repo := newFakeRoomRepository()
		queryService := application.NewRoomQueryService(repo)

		_, err := queryService.GetRoom(context.Background(), "no-such-room")
		if !errors.Is(err, application.ErrRoomNotFound) {
			t.Errorf("GetRoom() error = %v, want ErrRoomNotFound", err)
		}
	})
}
