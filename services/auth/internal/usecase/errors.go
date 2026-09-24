package usecase

import "errors"

var (
	ErrEmailAlreadyTaken           = errors.New("email already taken")
	ErrInvalidPassword             = errors.New("invalid password")
	ErrPasswordIsNotStrongEnough   = errors.New("password must contain upper and lowercase letters, numbers and symbols")
	ErrPasswordHashFailed          = errors.New("password hash failed")
	ErrTokenGenerationFailed       = errors.New("failed to generate token")
	ErrVerificationTokenSaveFailed = errors.New("failed to save verification token")
	ErrVerificationEmailSendFailed = errors.New("failed to send verification email")
	ErrTokenPairGenerationFail     = errors.New("unable to generate access and refresh tokens")
	ErrTokenPairSaveFail           = errors.New("unable to save access and refresh tokens")
	ErrUserIdNotFound              = errors.New("user with such id not found")
	ErrUnauthorizedAction          = errors.New("you have no permission for this action")
	ErrInvalidUpdateEmailToken     = errors.New("invalid update token")
	ErrSameEmail                   = errors.New("new email must be different from current")
	ErrInvalidCredentials          = errors.New("invalid user or password")
	ErrSameNewPassword             = errors.New("new password must be different from current one")
	ErrInvalidRefreshToken         = errors.New("invalid or expired refresh token")
	ErrUserEmailIsNotVerified      = errors.New("user email is not verified yet")
	ErrWrongVerificationToken      = errors.New("wrong verification token")
	ErrUserActivationFail          = errors.New("user activation failed")
	ErrVerificationTokenDeleteFail = errors.New("failed to delete verification token")
	ErrEmailAlreadyVerified        = errors.New("email already verified")
)
