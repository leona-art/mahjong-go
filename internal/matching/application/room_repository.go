package application

import (
	"context"
	"errors"

	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

// ErrRoomNotFound is returned by RoomRepository.FindByID when no Room
// exists for the given id.
var ErrRoomNotFound = errors.New("application: room not found")

// RoomRepository persists and retrieves Room aggregates. It is declared
// here rather than in the domain layer so that domain stays a pure model
// of Room's business rules with no notion that it is ever persisted; the
// application layer is what orchestrates use cases against storage, so it
// owns this port. Implementations live in the Matching context's
// infrastructure layer (e.g. Firestore).
type RoomRepository interface {
	Save(ctx context.Context, room *domain.Room) error
	FindByID(ctx context.Context, id domain.RoomID) (*domain.Room, error)
}
