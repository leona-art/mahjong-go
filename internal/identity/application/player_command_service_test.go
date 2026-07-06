package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/leona-art/mahjong-go/internal/identity/application"
	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

func TestPlayerCommandService_RegisterPlayer(t *testing.T) {
	t.Run("未登録のuidはPlayerとして登録できる", func(t *testing.T) {
		service, repo := newTestCommandService()

		err := service.RegisterPlayer(context.Background(), application.RegisterPlayerCommand{
			UID:         "uid-1",
			DisplayName: "たろう",
		})
		if err != nil {
			t.Fatalf("RegisterPlayer() error = %v", err)
		}

		stored, err := repo.FindByUID(context.Background(), "uid-1")
		if err != nil {
			t.Fatalf("FindByUID() error = %v", err)
		}
		if stored.DisplayName() != "たろう" {
			t.Errorf("stored DisplayName() = %q, want %q", stored.DisplayName(), "たろう")
		}
	})

	t.Run("登録済みのuidの再登録は拒否される", func(t *testing.T) {
		service, _ := newTestCommandService()
		ctx := context.Background()

		if err := service.RegisterPlayer(ctx, application.RegisterPlayerCommand{UID: "uid-1", DisplayName: "たろう"}); err != nil {
			t.Fatalf("RegisterPlayer() error = %v", err)
		}

		err := service.RegisterPlayer(ctx, application.RegisterPlayerCommand{UID: "uid-1", DisplayName: "じろう"})
		if !errors.Is(err, application.ErrPlayerAlreadyRegistered) {
			t.Errorf("RegisterPlayer() error = %v, want ErrPlayerAlreadyRegistered", err)
		}
	})

	t.Run("表示名が空の場合は登録が拒否される", func(t *testing.T) {
		service, _ := newTestCommandService()

		err := service.RegisterPlayer(context.Background(), application.RegisterPlayerCommand{UID: "uid-1", DisplayName: ""})
		if !errors.Is(err, domain.ErrDisplayNameRequired) {
			t.Errorf("RegisterPlayer() error = %v, want ErrDisplayNameRequired", err)
		}
	})
}
