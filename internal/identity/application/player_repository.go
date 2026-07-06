package application

import (
	"context"
	"errors"

	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

var (
	// ErrPlayerNotFound is returned by PlayerRepository.FindByUID when no
	// Player exists for the given uid.
	ErrPlayerNotFound = errors.New("application: player not found")

	// ErrPlayerAlreadyExists is returned by PlayerRepository.Create when a
	// Player already exists for the uid being registered.
	ErrPlayerAlreadyExists = errors.New("application: player already exists")
)

// PlayerRepository persists and retrieves Player profiles. It is declared
// here rather than in the domain layer so that domain stays a pure model
// of Player's business rules with no notion that it is ever persisted;
// the application layer is what orchestrates use cases against storage,
// so it owns this port. Implementations live in the Identity context's
// infrastructure layer (e.g. Firestore).
type PlayerRepository interface {
	// Create persists a new Player, failing with ErrPlayerAlreadyExists if
	// a profile already exists for its uid. Registration is create-only:
	// there is no upsert, since a uid registers its profile at most once.
	Create(ctx context.Context, player *domain.Player) error
	FindByUID(ctx context.Context, uid domain.UID) (*domain.Player, error)
}
