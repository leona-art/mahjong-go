// Package application contains the Identity bounded context's application
// layer: Command/Query use cases that orchestrate the Player aggregate.
package application

import (
	"context"
	"errors"

	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

// ErrPlayerAlreadyRegistered is returned by RegisterPlayer when the
// caller's uid already has a Player profile.
var ErrPlayerAlreadyRegistered = errors.New("application: player already registered")

// PlayerCommandService implements the state-changing use cases for the
// Identity context's Player aggregate.
type PlayerCommandService struct {
	players domain.PlayerRepository
}

// NewPlayerCommandService wires a PlayerCommandService to its repository.
func NewPlayerCommandService(players domain.PlayerRepository) *PlayerCommandService {
	return &PlayerCommandService{players: players}
}

// RegisterPlayerCommand creates a Player profile for an authenticated uid.
// UID must come from a verified Firebase ID token, never from unauthenticated
// client input.
type RegisterPlayerCommand struct {
	UID         domain.UID
	DisplayName string
}

func (s *PlayerCommandService) RegisterPlayer(ctx context.Context, cmd RegisterPlayerCommand) (PlayerView, error) {
	player, err := domain.NewPlayer(cmd.UID, cmd.DisplayName)
	if err != nil {
		return PlayerView{}, err
	}
	if err := s.players.Create(ctx, player); err != nil {
		if errors.Is(err, domain.ErrPlayerAlreadyExists) {
			return PlayerView{}, ErrPlayerAlreadyRegistered
		}
		return PlayerView{}, err
	}
	return newPlayerView(player), nil
}
