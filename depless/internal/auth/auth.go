package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"

	"github.com/golang-jwt/jwt"
)

type Auth struct {
	secret string
}

func New(secret string) *Auth {
	return &Auth{
		secret: secret,
	}
}

type ctxKey uint32

const (
	userIDKey ctxKey = iota
)

func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return uuid.UUID{}, errors.New("nil userID")
	}
	if userID, ok := val.(uuid.UUID); ok {
		return userID, nil
	}

	return uuid.UUID{}, errors.New("fail assert userID")
}

type ErrAuth struct {
	error
}

func wrapErrAuth(err error) ErrAuth {
	return ErrAuth{err}
}

type Claims struct {
	jwt.StandardClaims
	UserID uuid.UUID
}

func (a *Auth) Authorize(_ context.Context, claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	authToken, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", fmt.Errorf("SignedString: %w ", err)
	}

	return authToken, nil
}

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrWrongSigningMethod = errors.New("err Wrong Signing Method")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrExtractUserID      = errors.New("err Extract User ID")
)

const (
	zeroUUID = "00000000-0000-0000-0000-000000000000"
)

func (a *Auth) ExtractUserID(ctx context.Context, authToken string) (uuid.UUID, error) {
	claims, err := a.extractClaims(ctx, authToken)
	if err != nil {
		return uuid.UUID{}, wrapErrAuth(errors.Join(ErrUnauthorized, err))
	}
	fmt.Println("claims.UserID", claims.UserID.String())
	if claims.UserID.String() == zeroUUID {
		return uuid.UUID{}, wrapErrAuth(ErrExtractUserID)
	}

	return claims.UserID, nil
}

func (a *Auth) ExtractClaims(ctx context.Context, authToken string) (Claims, error) {
	claims, err := a.extractClaims(ctx, authToken)
	if err != nil {
		return Claims{}, wrapErrAuth(errors.Join(ErrUnauthorized, err))
	}

	return claims, err
}

func (a *Auth) fetchSecret(token *jwt.Token) (any, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, ErrWrongSigningMethod
	}

	return []byte(a.secret), nil
}

func (a *Auth) extractClaims(_ context.Context, authToken string) (Claims, error) {
	token, err := jwt.Parse(authToken, a.fetchSecret)
	if err != nil {
		return Claims{}, err
	}
	if !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if token.Claims == nil {
		return Claims{}, errors.New("claims is nil")
	}
	if err = token.Claims.Valid(); err != nil {
		return Claims{}, err
	}
	var claims Claims
	b, err := json.Marshal(token.Claims)
	if err != nil {
		return Claims{}, err
	}
	if err = json.Unmarshal(b, &claims); err != nil {
		return Claims{}, err
	}

	return claims, nil
}
