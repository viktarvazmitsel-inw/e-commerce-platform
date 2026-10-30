package handler

import (
	"authorization/internal/interface/http/middleware"
	"authorization/internal/interface/http/response"
	"context"
	"net/http"
	"strings"
)

type HandlerWithUserID func(w http.ResponseWriter, r *http.Request, userID string)

func WithUserID(next HandlerWithUserID) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := getUserIDFromContext(r.Context())
		userID = strings.TrimSpace(userID)

		if err != nil || userID == "" {
			response.RespondJSON(w, http.StatusUnauthorized, &response.Response{
				Success: false,
				Errors: &response.APIError{
					Code:    "UNAUTHORIZED",
					Message: "Unauthorized",
				},
			})
			return
		}

		next(w, r, userID)
	}
}

func getUserIDFromContext(ctx context.Context) (string, error) {
	id := ctx.Value(middleware.UserIDContextKey)

	if id == nil {
		return "", ErrUserIDUndefined
	}

	authorID, ok := id.(string)
	if !ok || strings.TrimSpace(authorID) == "" {
		return "", ErrUserIDInvalid
	}

	return authorID, nil
}
