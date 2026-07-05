package application

import (
	"context"

	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

// PlayerView is a read model of a Player's current profile.
type PlayerView struct {
	UID         domain.UID
	DisplayName string
}

func newPlayerView(player *domain.Player) PlayerView {
	return PlayerView{UID: player.UID(), DisplayName: player.DisplayName()}
}

// PlayerQueryService implements the read-only use cases for the Identity
// context's Player aggregate.
type PlayerQueryService struct {
	players PlayerRepository
}

// NewPlayerQueryService wires a PlayerQueryService to its repository.
func NewPlayerQueryService(players PlayerRepository) *PlayerQueryService {
	return &PlayerQueryService{players: players}
}

func (s *PlayerQueryService) GetPlayer(ctx context.Context, uid domain.UID) (PlayerView, error) {
	player, err := s.players.FindByUID(ctx, uid)
	if err != nil {
		return PlayerView{}, err
	}
	return newPlayerView(player), nil
}
