package domain

import (
	"context"
	"errors"
)

// ErrRoomNotFound is returned by RoomRepository.FindByID when no Room
// exists for the given id.
var ErrRoomNotFound = errors.New("domain: room not found")

// RoomRepository persists and retrieves Room aggregates. Implementations
// live in the Matching context's infrastructure layer (e.g. Firestore).
type RoomRepository interface {
	Save(ctx context.Context, room *Room) error
	FindByID(ctx context.Context, id RoomID) (*Room, error)
}
