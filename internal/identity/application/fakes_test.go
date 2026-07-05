package application_test

import (
	"context"
	"sync"

	"github.com/leona-art/mahjong-go/internal/identity/application"
	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

// fakePlayerRepository is an in-memory domain.PlayerRepository used to test
// the application layer in isolation from any real persistence.
type fakePlayerRepository struct {
	mu      sync.Mutex
	players map[domain.UID]*domain.Player
}

func newFakePlayerRepository() *fakePlayerRepository {
	return &fakePlayerRepository{players: make(map[domain.UID]*domain.Player)}
}

func (f *fakePlayerRepository) Create(_ context.Context, player *domain.Player) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.players[player.UID()]; ok {
		return domain.ErrPlayerAlreadyExists
	}
	f.players[player.UID()] = player
	return nil
}

func (f *fakePlayerRepository) FindByUID(_ context.Context, uid domain.UID) (*domain.Player, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	player, ok := f.players[uid]
	if !ok {
		return nil, domain.ErrPlayerNotFound
	}
	return player, nil
}

// newTestCommandService wires a PlayerCommandService to a fresh fake
// repository.
func newTestCommandService() (*application.PlayerCommandService, *fakePlayerRepository) {
	repo := newFakePlayerRepository()
	return application.NewPlayerCommandService(repo), repo
}
