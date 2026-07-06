// Package firestore is the Identity bounded context's infrastructure-layer
// implementation of application.PlayerRepository, backed by Cloud
// Firestore.
package firestore

import (
	"context"
	"fmt"
	"strings"

	fs "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/leona-art/mahjong-go/internal/identity/application"
	"github.com/leona-art/mahjong-go/internal/identity/domain"
)

const playersCollection = "players"

// PlayerRepository is a Firestore-backed application.PlayerRepository.
// Point the client at the Firestore emulator locally by setting the
// FIRESTORE_EMULATOR_HOST environment variable before constructing it.
type PlayerRepository struct {
	client *fs.Client
}

// NewPlayerRepository wires a PlayerRepository to an already-configured
// Firestore client.
func NewPlayerRepository(client *fs.Client) *PlayerRepository {
	return &PlayerRepository{client: client}
}

// playerDocument is the Firestore wire representation of a Player.
type playerDocument struct {
	DisplayName string `firestore:"displayName"`
}

func (r *PlayerRepository) Create(ctx context.Context, player *domain.Player) error {
	if err := validateUID(player.UID()); err != nil {
		return err
	}
	doc := playerDocument{DisplayName: player.DisplayName()}
	if _, err := r.client.Collection(playersCollection).Doc(string(player.UID())).Create(ctx, doc); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return application.ErrPlayerAlreadyExists
		}
		return fmt.Errorf("firestore: create player %s: %w", player.UID(), err)
	}
	return nil
}

func (r *PlayerRepository) FindByUID(ctx context.Context, uid domain.UID) (*domain.Player, error) {
	if err := validateUID(uid); err != nil {
		return nil, err
	}
	snap, err := r.client.Collection(playersCollection).Doc(string(uid)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, application.ErrPlayerNotFound
		}
		return nil, fmt.Errorf("firestore: find player %s: %w", uid, err)
	}

	var doc playerDocument
	if err := snap.DataTo(&doc); err != nil {
		return nil, fmt.Errorf("firestore: decode player %s: %w", uid, err)
	}
	return domain.RehydratePlayer(uid, doc.DisplayName), nil
}

func validateUID(uid domain.UID) error {
	if strings.Contains(string(uid), "/") {
		return fmt.Errorf("firestore: invalid uid %q: must not contain '/'", uid)
	}
	return nil
}
