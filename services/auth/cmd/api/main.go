package main

import (
	"authorization/internal/config"
	"authorization/internal/infrastructure/auth"
	"authorization/internal/infrastructure/broker"
	messageBroker "authorization/internal/infrastructure/broker"
	"authorization/internal/infrastructure/db/postgre"
	"authorization/internal/infrastructure/db/redis"
	tokenStorage "authorization/internal/infrastructure/db/redis"
	"authorization/internal/infrastructure/hasher"
	coreLogger "authorization/internal/infrastructure/logger"
	jwtToken "authorization/internal/infrastructure/token"
	"authorization/internal/interface/http/handler"
	middlewares "authorization/internal/interface/http/middleware"
	"authorization/internal/interface/http/response"
	"authorization/internal/usecase"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}

func main() {
	logger := log.New(os.Stdout, "auth-back: ", log.LstdFlags)
	logger.Println("Starting server on port 8080")

	coreLogger := coreLogger.NewLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	logger.Println("Core logger created")

	config := config.Load()
	logger.Println("Config loaded")

	ctx := context.Background()

	dbConfig := &postgre.PostgresConfig{
		User:     config.DatabaseUser,
		Password: config.DatabasePassword,
		Host:     config.DatabaseHost,
		Port:     config.DatabasePort,
		DBName:   config.DatabaseName,
	}

	db, err := postgre.NewPostgresPool(ctx, dbConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	logger.Println("Database connected")

	userRepository := postgre.NewUserRepository(db)
	logger.Println("User repository created")

	redisConfig := &redis.RedisConfig{
		Host:     config.RedisHost,
		Port:     config.RedisPort,
		User:     config.RedisUser,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	}

	cache, err := redis.NewRedisClient(ctx, redisConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	logger.Println("Redis connected")

	jwtKeyRepository := redis.NewJwtKeyRepository(cache)

	jwtKeyLoader := auth.NewJwtKeyLoader(jwtKeyRepository)

	privateKey, err := jwtKeyLoader.LoadPrivateKey(config.TokenPrivateKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load private key: %v", err)
	}
	logger.Println("Private key loaded")

	publicKey, err := jwtKeyLoader.LoadPublicKey(config.TokenPublicKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load public key: %v", err)
	}
	logger.Println("Public key loaded")

	tokenGenerator, err := jwtToken.NewJWTGenerator(privateKey, config.RefreshTokenTTL, config.AccessTokenTTL)
	if err != nil {
		logger.Fatalf("Failed to create token generator: %v", err)
	}
	logger.Println("Token generator created")

	verificationTokenRepository := tokenStorage.NewVerificationTokenRepository(cache, config.VerificationTokenTTL)
	logger.Println("Verification token storage created")

	userDataUpdateTokenRepository := tokenStorage.NewUserDataUpdateTokenRepository(cache, config.ResetTokenTTL)
	logger.Println("User data update token storage created")

	tokenDataExtractor := jwtToken.NewDataExtractor(publicKey)

	sessionStorage := tokenStorage.NewSessionRepository(cache, config.RefreshTokenTTL, coreLogger, tokenDataExtractor)
	logger.Println("Session storage created")

	rmqConfig := &broker.RMQConfig{
		Host:     config.RMQHost,
		Port:     config.RMQPort,
		User:     config.RMQUser,
		Password: config.RMQPassword,
	}

	rmq, err := broker.NewConnection(ctx, rmqConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	logger.Println("RabbitMQ connected")

	passwordHasher := hasher.NewPasswordHasher(10)
	logger.Println("Password hasher created")

	emailSender, err := messageBroker.NewEmailSender(ctx, rmq, coreLogger)
	if err != nil {
		logger.Fatalf("Failed to create email sender: %v", err)
	}
	logger.Println("Email sender created")

	registerUserUseCase := usecase.NewRegisterUserUseCase(userRepository, emailSender, verificationTokenRepository, passwordHasher, coreLogger)
	logger.Println("Register user use case created")

	registerUserHandler := handler.NewRegisterUserHandler(registerUserUseCase)
	logger.Println("Register user handler created")

	verifyUserUseCase := usecase.NewVerifyUserUseCase(userRepository, verificationTokenRepository, coreLogger)
	logger.Println("Verify user use case created")

	verifyUserHandler := handler.NewVerifyUserHandler(verifyUserUseCase)
	logger.Println("Verify user handler created")

	resendVerificationEmailUseCase := usecase.NewResendVerificationEmailUseCase(verificationTokenRepository, userRepository, passwordHasher, emailSender)
	logger.Println("Resend verification email use case created")

	resendVerificationEmailHandler := handler.NewResendVerificationEmailHandler(resendVerificationEmailUseCase)
	logger.Println("Resend verification email handler created")

	initiatePasswordRecoveryUseCase := usecase.NewInitiatePasswordRecoveryUseCase(userRepository, userDataUpdateTokenRepository, emailSender)
	logger.Println("Initiate password recovery use case created")

	initiatePasswordRecoveryHandler := handler.NewInitiatePasswordRecoveryHandler(initiatePasswordRecoveryUseCase)
	logger.Println("Initiate password recovery handler created")

	confirmPasswordRecoveryUseCase := usecase.NewConfirmPasswordRecoveryUseCase(userRepository, userDataUpdateTokenRepository, passwordHasher, sessionStorage, coreLogger)
	logger.Println("Confirm password recovery use case created")

	confirmPasswordRecoveryHandler := handler.NewConfirmPasswordRecoveryHandler(confirmPasswordRecoveryUseCase)
	logger.Println("Confirm password recovery handler created")

	loginUserUseCase := usecase.NewLoginUserUseCase(userRepository, sessionStorage, tokenGenerator, passwordHasher)
	logger.Println("Login user use case created")

	loginUserHandler := handler.NewLoginUserHandler(loginUserUseCase)
	logger.Println("Login user handler created")

	logoutUserUseCase := usecase.NewLogoutUserUseCase(sessionStorage)
	logger.Println("Logout user use case created")

	logoutUserHandler := handler.NewLogoutUserHandler(logoutUserUseCase)
	logger.Println("Logout user handler created")

	getUserUseCase := usecase.NewGetUserProfileUseCase(userRepository)
	logger.Println("Get user use case created")

	getUserHandler := handler.NewGetUserProfileHandler(getUserUseCase)
	logger.Println("Get user handler created")

	initiateEmailUpdateUseCase := usecase.NewInitiateEmailUpdateUseCase(userRepository, emailSender, userDataUpdateTokenRepository, passwordHasher)
	logger.Println("Initiate email update use case created")

	initiateEmailUpdateHandler := handler.NewInitiateEmailUpdateHandler(initiateEmailUpdateUseCase)
	logger.Println("Initiate email update handler created")

	confirmEmailUpdateUseCase := usecase.NewConfirmEmailUpdateUseCase(userRepository, userDataUpdateTokenRepository, sessionStorage, coreLogger)
	logger.Println("Confirm email update use case created")

	confirmEmailUpdateHandler := handler.NewConfirmEmailUpdateHandler(confirmEmailUpdateUseCase)
	logger.Println("Confirm email update handler created")

	passwordUpdateUseCase := usecase.NewPasswordUpdateUseCase(userRepository, sessionStorage, passwordHasher, coreLogger)
	logger.Println("Password update use case created")

	passwordUpdateHandler := handler.NewPasswordUpdateHandler(passwordUpdateUseCase)
	logger.Println("Password update handler created")

	refreshAccessUseCase := usecase.NewRefreshAccessUseCase(userRepository, sessionStorage, tokenGenerator)
	logger.Println("Refresh access use case created")

	refreshAccessHandler := handler.NewRefreshAccessHandler(refreshAccessUseCase)
	logger.Println("Refresh access handler created")

	updateUserProfileUseCase := usecase.NewUpdateUserProfileUseCase(userRepository)
	logger.Println("Update user profile use case created")

	updateUserProfileHandler := handler.NewUpdateUserProfileHandler(updateUserProfileUseCase)
	logger.Println("Update user profile handler created")

	profileSoftDeleteUseCase := usecase.NewProfileSoftDeleteUseCase(userRepository, sessionStorage, passwordHasher, coreLogger)
	logger.Println("Profile soft delete use case created")

	profileSoftDeleteHandler := handler.NewProfileSoftDeleteHandler(profileSoftDeleteUseCase)
	logger.Println("Profile soft delete handler created")

	listUsersUseCase := usecase.NewListUsersUseCase(userRepository)
	logger.Println("List users use case created")

	listUsersHandler := handler.NewListUsersHandler(listUsersUseCase)
	logger.Println("List users handler created")

	switchUserRoleUseCase := usecase.NewSwitchUserRoleUseCase(userRepository, sessionStorage, coreLogger)
	logger.Println("Switch user role use case created")

	switchUserRoleHandler := handler.NewSwitchUserRoleHandler(switchUserRoleUseCase)
	logger.Println("Switch user role handler created")

	publicKeyHandler, err := handler.NewPublicKeyHandler(publicKey)
	if err != nil {
		logger.Fatalf("Failed to create public key handler: %v", err)
	}
	logger.Println("Public key handler created")

	router := chi.NewRouter()
	accessVerificationMiddleware := middlewares.NewAccessVerificationMiddleware(publicKey)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Post("/api/auth/register", registerUserHandler.ServeHTTP)
	router.Post("/api/auth/verify", verifyUserHandler.ServeHTTP)
	router.Post("/api/auth/resend-verification-email", resendVerificationEmailHandler.ServeHTTP)
	router.Post("/api/auth/initiate-password-recovery", initiatePasswordRecoveryHandler.ServeHTTP)
	router.Post("/api/auth/confirm-password-recovery", confirmPasswordRecoveryHandler.ServeHTTP)
	router.Post("/api/auth/login", loginUserHandler.ServeHTTP)
	router.Post("/api/auth/refresh-access", refreshAccessHandler.ServeHTTP)
	router.Get("/api/auth/public-key", publicKeyHandler.ServeHTTP)

	router.Group(func(r chi.Router) {
		r.Use(accessVerificationMiddleware.VerifyAccess)
		r.Get("/api/auth/profile/{userID}", getUserHandler.ServeHTTP)
		r.Post("/api/auth/logout", logoutUserHandler.ServeHTTP)
		r.Post("/api/auth/initiate-email-update", initiateEmailUpdateHandler.ServeHTTP)
		r.Post("/api/auth/confirm-email-update", confirmEmailUpdateHandler.ServeHTTP)
		r.Patch("/api/auth/password-update", passwordUpdateHandler.ServeHTTP)
		r.Patch("/api/auth/update-user-profile", updateUserProfileHandler.ServeHTTP)
		r.Patch("/api/auth/profile-soft-delete", profileSoftDeleteHandler.ServeHTTP)
		r.Get("/api/auth/list-users", listUsersHandler.ServeHTTP)
		r.Patch("/api/auth/switch-user-role/{userID}", switchUserRoleHandler.ServeHTTP)
	})

	router.Route("/api/auth/check", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			response.RespondJSON(w, http.StatusOK, &response.Response{
				Success: true,
				Data: map[string]string{
					"message": "OK",
				},
			})
		})
	})

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
