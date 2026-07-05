package domain

import (
	"context"
	"errors"
)

var (
	// ErrPlayerNotFound is returned by PlayerRepository.FindByUID when no
	// Player exists for the given uid.
	ErrPlayerNotFound = errors.New("domain: player not found")

	// ErrPlayerAlreadyExists is returned by PlayerRepository.Create when a
	// Player already exists for the uid being registered.
	ErrPlayerAlreadyExists = errors.New("domain: player already exists")
)

// PlayerRepository persists and retrieves Player profiles. Implementations
// live in the Identity context's infrastructure layer (e.g. Firestore).
type PlayerRepository interface {
	// Create persists a new Player, failing with ErrPlayerAlreadyExists if
	// a profile already exists for its uid. Registration is create-only:
	// there is no upsert, since a uid registers its profile at most once.
	Create(ctx context.Context, player *Player) error
	FindByUID(ctx context.Context, uid UID) (*Player, error)
}
