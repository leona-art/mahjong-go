package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leona-art/mahjong-go/internal/identity/application"
	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

func TestPlayerQueryService_GetPlayer(t *testing.T) {
	t.Run("登録済みのPlayerを取得できる", func(t *testing.T) {
		commands, repo := newTestCommandService()
		if err := commands.RegisterPlayer(context.Background(), application.RegisterPlayerCommand{
			UID:         "uid-1",
			DisplayName: "たろう",
		}); err != nil {
			t.Fatalf("RegisterPlayer() error = %v", err)
		}
		queries := application.NewPlayerQueryService(repo)

		view, err := queries.GetPlayer(context.Background(), "uid-1")
		if err != nil {
			t.Fatalf("GetPlayer() error = %v", err)
		}
		if view.UID != "uid-1" || view.DisplayName != "たろう" {
			t.Errorf("GetPlayer() view = %+v, want {uid-1 たろう}", view)
		}
	})

	t.Run("未登録のuidの取得はErrPlayerNotFoundになる", func(t *testing.T) {
		repo := newFakePlayerRepository()
		queries := application.NewPlayerQueryService(repo)

		_, err := queries.GetPlayer(context.Background(), "unknown")
		if !errors.Is(err, domain.ErrPlayerNotFound) {
			t.Errorf("GetPlayer() error = %v, want ErrPlayerNotFound", err)
		}
	})
}
