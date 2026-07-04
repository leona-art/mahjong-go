// Package domain contains the Matching bounded context's domain model: the
// Room aggregate that governs invite-only matchmaking before a mahjong match
// begins.
package domain

import "errors"

// SeatCount is the number of players a Room seats before a match can start.
const SeatCount = 4

// UID identifies a player, sourced from the Identity context (Identity
// Platform). The Matching context treats it as an opaque identifier.
type UID string

// RoomID identifies a Room.
type RoomID string

// RoomStatus is the lifecycle state of a Room.
type RoomStatus int

const (
	RoomStatusWaiting RoomStatus = iota
	RoomStatusReady
	RoomStatusStarted
)

// GameLength selects the length of the match a Room's players will play.
type GameLength int

const (
	GameLengthTonpuusen GameLength = iota // 東風戦
	GameLengthHanchan                     // 半荘
)

// RoomSettings holds the match rules chosen by the host at room creation.
type RoomSettings struct {
	GameLength   GameLength
	InitialScore int
}

var (
	ErrHostUIDRequired = errors.New("domain: host uid is required")
	ErrRoomFull        = errors.New("domain: room is full")
	ErrAlreadyJoined   = errors.New("domain: player already joined")
	ErrPlayerNotInRoom = errors.New("domain: player is not in the room")
	ErrHostCannotLeave = errors.New("domain: host cannot leave the room")
	ErrRoomStarted     = errors.New("domain: room has already started")
	ErrSeatsNotFilled  = errors.New("domain: not all seats are filled")
)

// Room is the aggregate root for invite-only matchmaking: a host creates a
// room, guests join open seats, and once all seats are filled the host can
// start the match. Room only tracks seating; it has no knowledge of dealer
// assignment or match play, which belong to the Game context.
type Room struct {
	id       RoomID
	hostUID  UID
	seats    [SeatCount]UID
	settings RoomSettings
	status   RoomStatus
}

// NewRoom creates a Room with the host occupying the first seat.
func NewRoom(id RoomID, hostUID UID, settings RoomSettings) (*Room, error) {
	if hostUID == "" {
		return nil, ErrHostUIDRequired
	}
	room := &Room{
		id:       id,
		hostUID:  hostUID,
		settings: settings,
		status:   RoomStatusWaiting,
	}
	room.seats[0] = hostUID
	return room, nil
}

func (r *Room) ID() RoomID             { return r.id }
func (r *Room) HostUID() UID           { return r.hostUID }
func (r *Room) Settings() RoomSettings { return r.settings }
func (r *Room) Status() RoomStatus     { return r.status }

// Seats returns a copy of the current seating.
func (r *Room) Seats() [SeatCount]UID { return r.seats }

// Join seats uid in the first open seat.
func (r *Room) Join(uid UID) error {
	if r.status == RoomStatusStarted {
		return ErrRoomStarted
	}
	for _, seat := range r.seats {
		if seat == uid {
			return ErrAlreadyJoined
		}
	}
	for i, seat := range r.seats {
		if seat == "" {
			r.seats[i] = uid
			if r.isFull() {
				r.status = RoomStatusReady
			}
			return nil
		}
	}
	return ErrRoomFull
}

// Leave removes uid from its seat. The host cannot leave; disbanding the
// room is a separate concern not modeled here yet.
func (r *Room) Leave(uid UID) error {
	if r.status == RoomStatusStarted {
		return ErrRoomStarted
	}
	if uid == r.hostUID {
		return ErrHostCannotLeave
	}
	for i, seat := range r.seats {
		if seat == uid {
			r.seats[i] = ""
			r.status = RoomStatusWaiting
			return nil
		}
	}
	return ErrPlayerNotInRoom
}

// RoomStarted is the domain event raised when a Room transitions to
// RoomStatusStarted. The Game context consumes it to create a new match,
// deciding dealer assignment itself.
type RoomStarted struct {
	RoomID RoomID
	Seats  [SeatCount]UID
}

// Start transitions the Room to RoomStatusStarted once all seats are
// filled, returning the event the Game context reacts to.
func (r *Room) Start() (*RoomStarted, error) {
	if r.status == RoomStatusStarted {
		return nil, ErrRoomStarted
	}
	if !r.isFull() {
		return nil, ErrSeatsNotFilled
	}
	r.status = RoomStatusStarted
	return &RoomStarted{RoomID: r.id, Seats: r.seats}, nil
}

func (r *Room) isFull() bool {
	for _, seat := range r.seats {
		if seat == "" {
			return false
		}
	}
	return true
}
