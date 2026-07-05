package firebaseauth_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/leona-art/mahjong-go/internal/identity/infrastructure/firebaseauth"
)

type fakeVerifier struct {
	uid string
	err error
}

func (f fakeVerifier) VerifyIDToken(_ context.Context, idToken string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.uid, nil
}

func newTestRequest(authorization string) connect.AnyRequest {
	req := connect.NewRequest(&struct{}{})
	if authorization != "" {
		req.Header().Set("Authorization", authorization)
	}
	return req
}

func TestInterceptor_WrapUnary(t *testing.T) {
	t.Run("有効なトークンならuidがcontextに設定される", func(t *testing.T) {
		interceptor := firebaseauth.NewInterceptor(fakeVerifier{uid: "uid-1"})
		var gotUID string
		next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			uid, ok := firebaseauth.UIDFromContext(ctx)
			if !ok {
				t.Fatal("UIDFromContext() ok = false, want true")
			}
			gotUID = uid
			return connect.NewResponse(&struct{}{}), nil
		}

		if _, err := interceptor.WrapUnary(next)(context.Background(), newTestRequest("Bearer valid-token")); err != nil {
			t.Fatalf("WrapUnary() error = %v", err)
		}
		if gotUID != "uid-1" {
			t.Errorf("uid = %q, want %q", gotUID, "uid-1")
		}
	})

	t.Run("Authorizationヘッダがない場合はUnauthenticatedになる", func(t *testing.T) {
		interceptor := firebaseauth.NewInterceptor(fakeVerifier{uid: "uid-1"})
		next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			t.Fatal("next should not be called")
			return nil, nil
		}

		_, err := interceptor.WrapUnary(next)(context.Background(), newTestRequest(""))
		assertUnauthenticated(t, err)
	})

	t.Run("トークン検証に失敗した場合はUnauthenticatedになる", func(t *testing.T) {
		interceptor := firebaseauth.NewInterceptor(fakeVerifier{err: errors.New("boom")})
		next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			t.Fatal("next should not be called")
			return nil, nil
		}

		_, err := interceptor.WrapUnary(next)(context.Background(), newTestRequest("Bearer bad-token"))
		assertUnauthenticated(t, err)
	})
}

func assertUnauthenticated(t *testing.T, err error) {
	t.Helper()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error = %v, want *connect.Error", err)
	}
	if connectErr.Code() != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want CodeUnauthenticated", connectErr.Code())
	}
}
