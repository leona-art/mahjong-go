package application

import (
	"context"

	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

// RoomView is a read model of a Room's current state.
type RoomView struct {
	RoomID   domain.RoomID
	HostUID  domain.UID
	Seats    [domain.SeatCount]domain.UID
	Settings domain.RoomSettings
	Status   domain.RoomStatus
}

// RoomQueryService implements the read-only use cases for the Matching
// context's Room aggregate.
type RoomQueryService struct {
	rooms RoomRepository
}

// NewRoomQueryService wires a RoomQueryService to its repository.
func NewRoomQueryService(rooms RoomRepository) *RoomQueryService {
	return &RoomQueryService{rooms: rooms}
}

func (s *RoomQueryService) GetRoom(ctx context.Context, id domain.RoomID) (RoomView, error) {
	room, err := s.rooms.FindByID(ctx, id)
	if err != nil {
		return RoomView{}, err
	}
	return RoomView{
		RoomID:   room.ID(),
		HostUID:  room.HostUID(),
		Seats:    room.Seats(),
		Settings: room.Settings(),
		Status:   room.Status(),
	}, nil
}
