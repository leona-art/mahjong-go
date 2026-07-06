// Package firebaseauth is the Identity bounded context's infrastructure-layer
// integration with Identity Platform (Firebase Authentication): verifying
// Firebase ID tokens and exposing the resulting uid to connect-go handlers
// via an interceptor.
package firebaseauth

import (
	"context"
	"errors"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

// ErrInvalidToken is returned by TokenVerifier.VerifyIDToken when the
// supplied ID token fails verification (expired, malformed, wrong
// audience, etc).
var ErrInvalidToken = errors.New("firebaseauth: invalid id token")

// NewClient initializes a Firebase Auth client for projectID. Point it at
// the Firebase Auth emulator locally by setting the
// FIREBASE_AUTH_EMULATOR_HOST environment variable before calling this.
func NewClient(ctx context.Context, projectID string) (*auth.Client, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("firebaseauth: init app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebaseauth: init auth client: %w", err)
	}
	return client, nil
}

// TokenVerifier verifies Firebase ID tokens against Identity Platform using
// the Firebase Admin SDK.
type TokenVerifier struct {
	client *auth.Client
}

// NewTokenVerifier wires a TokenVerifier to an already-initialized Firebase
// Auth client (see NewClient).
func NewTokenVerifier(client *auth.Client) *TokenVerifier {
	return &TokenVerifier{client: client}
}

// VerifyIDToken verifies idToken and returns the uid it was issued for.
func (v *TokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	token, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return token.UID, nil
}
