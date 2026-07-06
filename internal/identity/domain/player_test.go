package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

func TestNewPlayer(t *testing.T) {
	t.Run("uidと表示名を持つPlayerを作成できる", func(t *testing.T) {
		player, err := domain.NewPlayer("uid-1", "たろう")
		if err != nil {
			t.Fatalf("NewPlayer() error = %v", err)
		}
		if got := player.UID(); got != "uid-1" {
			t.Errorf("UID() = %q, want %q", got, "uid-1")
		}
		if got := player.DisplayName(); got != "たろう" {
			t.Errorf("DisplayName() = %q, want %q", got, "たろう")
		}
	})

	t.Run("uidが空の場合はエラーになる", func(t *testing.T) {
		if _, err := domain.NewPlayer("", "たろう"); !errors.Is(err, domain.ErrUIDRequired) {
			t.Errorf("NewPlayer() error = %v, want ErrUIDRequired", err)
		}
	})

	t.Run("表示名が空の場合はエラーになる", func(t *testing.T) {
		if _, err := domain.NewPlayer("uid-1", ""); !errors.Is(err, domain.ErrDisplayNameRequired) {
			t.Errorf("NewPlayer() error = %v, want ErrDisplayNameRequired", err)
		}
	})

	t.Run("表示名が上限文字数以内なら作成できる", func(t *testing.T) {
		name := strings.Repeat("あ", domain.MaxDisplayNameLength)
		if _, err := domain.NewPlayer("uid-1", name); err != nil {
			t.Errorf("NewPlayer() error = %v, want nil", err)
		}
	})

	t.Run("表示名が上限文字数を超える場合はエラーになる", func(t *testing.T) {
		name := strings.Repeat("あ", domain.MaxDisplayNameLength+1)
		if _, err := domain.NewPlayer("uid-1", name); !errors.Is(err, domain.ErrDisplayNameTooLong) {
			t.Errorf("NewPlayer() error = %v, want ErrDisplayNameTooLong", err)
		}
	})
}
