package handler

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"authorization/internal/domain"
	"authorization/internal/usecase"
)

var (
	nameRegex  = regexp.MustCompile(`^\p{L}{2,30}$`)
	phoneRegex = regexp.MustCompile(`^\+?\d{1,4}?[-.\s]?\(?\d{1,3}?\)?[-.\s]?\d{1,4}[-.\s]?\d{1,4}[-.\s]?\d{1,9}$`)
)

type RegisterUserUseCase interface {
	Execute(ctx context.Context, input usecase.RegisterInput) (usecase.RegisterUserResult, error)
}

type RegisterUserHandler struct {
	regUseCase RegisterUserUseCase
}

type UserData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Phone    string `json:"phone"`
}

func NewRegisterUser(uc RegisterUserUseCase) *RegisterUserHandler {
	return &RegisterUserHandler{
		regUseCase: uc,
	}
}

func (h *RegisterUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondJSON(w, http.StatusMethodNotAllowed, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "METHOD_NOT_ALLOWED",
				Message: "method not allowed",
			},
		})
		return
	}

	var userData UserData
	if !DecodeJSON(w, r, &userData) {
		return
	}

	errorsDetails := validateUserData(&userData)
	if len(errorsDetails) > 0 {
		RespondJSON(w, http.StatusBadRequest, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INVALID_USER_DATA",
				Message: "invalid user data",
				Details: errorsDetails,
			},
		})
		return
	}

	registerInput := usecase.RegisterInput{
		Email:    userData.Email,
		Password: userData.Password,
		Name:     userData.Name,
		Surname:  userData.Surname,
		Phone:    userData.Phone,
	}

	registerResult, err := h.regUseCase.Execute(r.Context(), registerInput)
	if err != nil {
		h.handleUsecaseError(w, err)
		return
	}

	if !registerResult.VerificationSent {
		RespondJSON(w, http.StatusAccepted, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "EMAIL_DELIVERY_FAIL",
				Message: "registration partially successful",
			},
		})
		return
	}

	RespondJSON(w, http.StatusOK, &Response{
		Success: true,
		Data:    map[string]string{"message": "check your email"},
	})
}

func validateUserData(uData *UserData) map[string]string {
	errorsDetails := make(map[string]string)

	uData.Email = strings.TrimSpace(uData.Email)
	uData.Email = strings.ToLower(uData.Email)
	if !domain.IsEmailValid(uData.Email) {
		errorsDetails["email"] = "email must be a valid email address"
	}
	if len(uData.Password) < 8 || len(uData.Password) > 50 {
		errorsDetails["password"] = "password must be from 8 to 50 characters"
	}
	if !nameRegex.MatchString(uData.Name) {
		errorsDetails["name"] = "name must consist of two or more letters"
	}
	if !nameRegex.MatchString(uData.Surname) {
		errorsDetails["surname"] = "surname must consist of two or more letters"
	}
	if !phoneRegex.MatchString(uData.Phone) {
		errorsDetails["phone"] = "incorrect phone number format"
	}

	return errorsDetails
}

func (h *RegisterUserHandler) handleUsecaseError(w http.ResponseWriter, err error) {
	errorsDetails := make(map[string]string)
	statusCode := http.StatusBadRequest

	switch {
	case errors.Is(err, usecase.ErrPasswordIsNotStrongEnough):
		errorsDetails["password"] = err.Error()
	case errors.Is(err, domain.ErrEmptyEmail), errors.Is(err, domain.ErrInvalidEmail):
		errorsDetails["email"] = err.Error()
	case errors.Is(err, domain.ErrEmptyName):
		errorsDetails["name"] = err.Error()
	case errors.Is(err, domain.ErrEmptySurname):
		errorsDetails["surname"] = err.Error()
	case errors.Is(err, domain.ErrInvalidPhone):
		errorsDetails["phone"] = err.Error()
	case errors.Is(err, usecase.ErrEmailAlreadyTaken):
		errorsDetails["email"] = err.Error()
		statusCode = http.StatusConflict
	default:
		RespondJSON(w, http.StatusInternalServerError, &Response{
			Success: false,
			Errors: &APIError{
				Code:    "INTERNAL_ERROR",
				Message: "registration failed",
			},
		})
		return
	}

	RespondJSON(w, statusCode, &Response{
		Success: false,
		Errors: &APIError{
			Code:    "REGISTRATION_FAILED",
			Message: "registration failed",
			Details: errorsDetails,
		},
	})
}
