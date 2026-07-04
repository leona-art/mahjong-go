package application_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/leona-art/mahjong-go/internal/matching/application"
	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

// fakeRoomRepository is an in-memory domain.RoomRepository used to test the
// application layer in isolation from any real persistence.
type fakeRoomRepository struct {
	mu    sync.Mutex
	rooms map[domain.RoomID]*domain.Room
}

func newFakeRoomRepository() *fakeRoomRepository {
	return &fakeRoomRepository{rooms: make(map[domain.RoomID]*domain.Room)}
}

func (f *fakeRoomRepository) Save(_ context.Context, room *domain.Room) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rooms[room.ID()] = room
	return nil
}

func (f *fakeRoomRepository) FindByID(_ context.Context, id domain.RoomID) (*domain.Room, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	room, ok := f.rooms[id]
	if !ok {
		return nil, domain.ErrRoomNotFound
	}
	return room, nil
}

func newTestSettings() domain.RoomSettings {
	return domain.RoomSettings{GameLength: domain.GameLengthHanchan, InitialScore: 25000}
}

// newTestCommandService wires a RoomCommandService to a fresh fake
// repository and a sequential RoomID generator.
func newTestCommandService() (*application.RoomCommandService, *fakeRoomRepository) {
	repo := newFakeRoomRepository()
	nextID := 0
	generator := func() domain.RoomID {
		nextID++
		return domain.RoomID(fmt.Sprintf("room-%d", nextID))
	}
	return application.NewRoomCommandService(repo, generator), repo
}
