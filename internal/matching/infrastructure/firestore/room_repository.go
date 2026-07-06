// Package firestore is the Matching bounded context's infrastructure-layer
// implementation of application.RoomRepository, backed by Cloud Firestore.
package firestore

import (
	"context"
	"fmt"
	"strings"

	fs "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/leona-art/mahjong-go/internal/matching/application"
	"github.com/leona-art/mahjong-go/internal/matching/domain"
)

const roomsCollection = "rooms"

// RoomRepository is a Firestore-backed application.RoomRepository. Point
// the client at the Firestore emulator locally by setting the
// FIRESTORE_EMULATOR_HOST environment variable before constructing it.
type RoomRepository struct {
	client *fs.Client
}

// NewRoomRepository wires a RoomRepository to an already-configured
// Firestore client.
func NewRoomRepository(client *fs.Client) *RoomRepository {
	return &RoomRepository{client: client}
}

// roomDocument is the Firestore wire representation of a Room.
type roomDocument struct {
	HostUID      string   `firestore:"hostUid"`
	Seats        []string `firestore:"seats"`
	GameLength   int      `firestore:"gameLength"`
	InitialScore int      `firestore:"initialScore"`
	Status       int      `firestore:"status"`
}

func (r *RoomRepository) Save(ctx context.Context, room *domain.Room) error {
	if err := validateRoomID(room.ID()); err != nil {
		return err
	}
	seats := room.Seats()
	doc := roomDocument{
		HostUID:      string(room.HostUID()),
		Seats:        seatsToStrings(seats),
		GameLength:   int(room.Settings().GameLength),
		InitialScore: room.Settings().InitialScore,
		Status:       int(room.Status()),
	}
	if _, err := r.client.Collection(roomsCollection).Doc(string(room.ID())).Set(ctx, doc); err != nil {
		return fmt.Errorf("firestore: save room %s: %w", room.ID(), err)
	}
	return nil
}

func (r *RoomRepository) FindByID(ctx context.Context, id domain.RoomID) (*domain.Room, error) {
	if err := validateRoomID(id); err != nil {
		return nil, err
	}
	snap, err := r.client.Collection(roomsCollection).Doc(string(id)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, application.ErrRoomNotFound
		}
		return nil, fmt.Errorf("firestore: find room %s: %w", id, err)
	}

	var doc roomDocument
	if err := snap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("firestore: decode room %s: %w", id, err)
	}
	seats, err := stringsToSeats(doc.Seats)
	if err != nil {
		return nil, fmt.Errorf("firestore: decode room %s: %w", id, err)
	}

	return domain.RehydrateRoom(
		id,
		domain.UID(doc.HostUID),
		seats,
		domain.RoomSettings{
			GameLength:   domain.GameLength(doc.GameLength),
			InitialScore: doc.InitialScore,
		},
		domain.RoomStatus(doc.Status),
	), nil
}

func seatsToStrings(seats [domain.SeatCount]domain.UID) []string {
	out := make([]string, domain.SeatCount)
	for i, uid := range seats {
		out[i] = string(uid)
	}
	return out
}

func stringsToSeats(seats []string) ([domain.SeatCount]domain.UID, error) {
	var out [domain.SeatCount]domain.UID
	if len(seats) != domain.SeatCount {
		return out, fmt.Errorf("expected %d seats, got %d", domain.SeatCount, len(seats))
	}
	for i, uid := range seats {
		out[i] = domain.UID(uid)
	}
	return out, nil
}

func validateRoomID(id domain.RoomID) error {
	if strings.Contains(string(id), "/") {
		return fmt.Errorf("firestore: invalid room id %q: must not contain '/'", id)
	}
	return nil
}
