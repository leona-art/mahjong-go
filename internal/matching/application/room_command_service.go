// Package application contains the Matching bounded context's application
// layer: Command/Query use cases that orchestrate the Room aggregate.
package application

import (
	"context"

	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

// RoomIDGenerator issues a new RoomID for a room being created.
type RoomIDGenerator func() domain.RoomID

// RoomCommandService implements the state-changing use cases for the
// Matching context's Room aggregate.
type RoomCommandService struct {
	rooms     RoomRepository
	newRoomID RoomIDGenerator
}

// NewRoomCommandService wires a RoomCommandService to its repository and ID
// generator.
func NewRoomCommandService(rooms RoomRepository, newRoomID RoomIDGenerator) *RoomCommandService {
	return &RoomCommandService{rooms: rooms, newRoomID: newRoomID}
}

// CreateRoomCommand creates a new Room with the host as its first occupant.
type CreateRoomCommand struct {
	HostUID  domain.UID
	Settings domain.RoomSettings
}

func (s *RoomCommandService) CreateRoom(ctx context.Context, cmd CreateRoomCommand) (domain.RoomID, error) {
	room, err := domain.NewRoom(s.newRoomID(), cmd.HostUID, cmd.Settings)
	if err != nil {
		return "", err
	}
	if err := s.rooms.Save(ctx, room); err != nil {
		return "", err
	}
	return room.ID(), nil
}

// JoinRoomCommand seats a guest in an existing Room.
type JoinRoomCommand struct {
	RoomID domain.RoomID
	UID    domain.UID
}

func (s *RoomCommandService) JoinRoom(ctx context.Context, cmd JoinRoomCommand) error {
	room, err := s.rooms.FindByID(ctx, cmd.RoomID)
	if err != nil {
		return err
	}
	if err := room.Join(cmd.UID); err != nil {
		return err
	}
	return s.rooms.Save(ctx, room)
}

// LeaveRoomCommand removes a guest from a Room before the match starts.
type LeaveRoomCommand struct {
	RoomID domain.RoomID
	UID    domain.UID
}

func (s *RoomCommandService) LeaveRoom(ctx context.Context, cmd LeaveRoomCommand) error {
	room, err := s.rooms.FindByID(ctx, cmd.RoomID)
	if err != nil {
		return err
	}
	if err := room.Leave(cmd.UID); err != nil {
		return err
	}
	return s.rooms.Save(ctx, room)
}
