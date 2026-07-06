// Package domain contains the Identity bounded context's domain model: the
// Player aggregate that tracks the profile linked to a Firebase
// Authentication uid.
package domain

import (
	"errors"
	"unicode/utf8"
)

// MaxDisplayNameLength is the maximum number of runes a Player's display
// name may contain.
const MaxDisplayNameLength = 32

// UID identifies a player. It is issued by Identity Platform (Firebase
// Authentication) when the player signs in and is treated by this context
// as an opaque identifier.
type UID string

var (
	ErrUIDRequired         = errors.New("domain: uid is required")
	ErrDisplayNameRequired = errors.New("domain: display name is required")
	ErrDisplayNameTooLong  = errors.New("domain: display name is too long")
)

// Player is the aggregate root for a registered player's profile: the
// display name shown to other players, keyed by the uid Identity Platform
// issued at authentication. Player holds no credentials; Identity Platform
// is the sole owner of sign-up and login.
type Player struct {
	uid         UID
	displayName string
}

// NewPlayer creates a Player profile for uid.
func NewPlayer(uid UID, displayName string) (*Player, error) {
	if uid == "" {
		return nil, ErrUIDRequired
	}
	if err := validateDisplayName(displayName); err != nil {
		return nil, err
	}
	return &Player{uid: uid, displayName: displayName}, nil
}

// RehydratePlayer reconstructs a Player from previously persisted state. It
// is intended for repository implementations restoring a Player from
// storage and does not re-run the invariants NewPlayer enforces at
// creation time, since that state was already valid when it was saved.
func RehydratePlayer(uid UID, displayName string) *Player {
	return &Player{uid: uid, displayName: displayName}
}

func (p *Player) UID() UID            { return p.uid }
func (p *Player) DisplayName() string { return p.displayName }

func validateDisplayName(displayName string) error {
	if displayName == "" {
		return ErrDisplayNameRequired
	}
	if utf8.RuneCountInString(displayName) > MaxDisplayNameLength {
		return ErrDisplayNameTooLong
	}
	return nil
}
