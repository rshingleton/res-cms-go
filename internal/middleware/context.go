package middleware

import (
	"context"
	"res-cms-go/internal/session"
)

// contextKey is a custom type to avoid collisions in context values
type contextKey string

const (
	userKey  contextKey = "user"
	flashKey contextKey = "flash"
	dataKey  contextKey = "data"
)

// WithUser adds user session to context
func WithUser(ctx context.Context, user *session.Session) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// WithFlash adds flash message to context
func WithFlash(ctx context.Context, msg string) context.Context {
	return context.WithValue(ctx, flashKey, msg)
}

// WithData adds template data to context
func WithData(ctx context.Context, data map[string]interface{}) context.Context {
	return context.WithValue(ctx, dataKey, data)
}

// GetUser returns user from context
func GetUser(ctx context.Context) *session.Session {
	user, _ := ctx.Value(userKey).(*session.Session)
	return user
}

// GetFlash returns flash message from context
func GetFlash(ctx context.Context) string {
	msg, _ := ctx.Value(flashKey).(string)
	return msg
}

// GetData returns template data from context
func GetData(ctx context.Context) map[string]interface{} {
	data, _ := ctx.Value(dataKey).(map[string]interface{})
	if data == nil {
		return make(map[string]interface{})
	}
	return data
}
