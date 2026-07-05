package firebaseauth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// Verifier verifies a Firebase ID token and returns the uid it was issued
// for. TokenVerifier implements it against a real Firebase Auth client;
// tests can supply a fake.
type Verifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (string, error)
}

type uidContextKey struct{}

// UIDFromContext returns the uid the auth interceptor extracted from the
// request's Firebase ID token, if any. Handlers behind NewInterceptor can
// rely on this being present.
func UIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(uidContextKey{}).(string)
	return uid, ok
}

// NewInterceptor builds a connect.Interceptor that verifies the Firebase ID
// token in the request's "Authorization: Bearer <token>" header and makes
// the resulting uid available via UIDFromContext. Requests without a valid
// token are rejected with connect.CodeUnauthenticated before reaching the
// handler.
func NewInterceptor(verifier Verifier) connect.Interceptor {
	return &authInterceptor{verifier: verifier}
}

type authInterceptor struct {
	verifier Verifier
}

func (i *authInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, err := i.authenticate(ctx, req.Header())
		if err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (i *authInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *authInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, err := i.authenticate(ctx, conn.RequestHeader())
		if err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (i *authInterceptor) authenticate(ctx context.Context, header http.Header) (context.Context, error) {
	token := bearerToken(header.Get("Authorization"))
	if token == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing bearer token"))
	}
	uid, err := i.verifier.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return context.WithValue(ctx, uidContextKey{}, uid), nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}
