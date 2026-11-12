package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"authorization/internal/config"
	authToken "authorization/internal/infrastructure/auth"
	"authorization/internal/infrastructure/broker"
	"authorization/internal/infrastructure/db/postgre"
	"authorization/internal/infrastructure/db/redis"
	"authorization/internal/infrastructure/hasher"
	"authorization/internal/infrastructure/healthcheck"
	coreLogger "authorization/internal/infrastructure/logger"
	jwtToken "authorization/internal/infrastructure/token"
	"authorization/internal/interface/http/handler"
	middlewares "authorization/internal/interface/http/middleware"
	"authorization/internal/usecase"
)

func main() {
	logger := log.New(os.Stdout, "auth-back: ", log.LstdFlags)

	coreLogger := coreLogger.NewLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	logger.Println("Core logger created")

	err := godotenv.Load(".env")
	if err != nil {
		logger.Printf("Error loading .env file: %v", err)
	}

	config := config.Load()
	logger.Println("Config loaded")

	ctx := context.Background()

	appCtx, appCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer appCancel()

	reqCtx, cancelRequests := context.WithCancel(ctx)
	defer cancelRequests()

	initCtx, initCancel := context.WithTimeout(ctx, 5*time.Second)
	defer initCancel()

	dbConfig := &postgre.PostgresConfig{
		User:     config.DatabaseUser,
		Password: config.DatabasePassword,
		Host:     config.DatabaseHost,
		Port:     config.DatabasePort,
		DBName:   config.DatabaseName,
	}

	db, err := postgre.NewPostgresPool(initCtx, dbConfig)
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

	cache, err := redis.NewRedisClient(initCtx, redisConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	logger.Println("Redis connected")

	privateKey, err := authToken.LoadPrivateKey(config.TokenPrivateKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load private key: %v", err)
	}
	logger.Println("Private key loaded")

	publicKey, err := authToken.LoadPublicKey(config.TokenPublicKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load public key: %v", err)
	}
	logger.Println("Public key loaded")

	err = authToken.ValidateKeys(privateKey, publicKey)
	if err != nil {
		logger.Fatalf("Failed to validate keys: %v", err)
	}
	logger.Println("Keys validated")

	tokenGenerator, err := jwtToken.NewJWTGenerator(privateKey, config.RefreshTokenTTL, config.AccessTokenTTL)
	if err != nil {
		logger.Fatalf("Failed to create token generator: %v", err)
	}
	logger.Println("Token generator created")

	verificationTokenRepository := redis.NewVerificationTokenRepository(cache, config.VerificationTokenTTL)
	logger.Println("Verification token storage created")

	userDataUpdateTokenRepository := redis.NewUserDataUpdateTokenRepository(cache, config.ResetTokenTTL)
	logger.Println("User data update token storage created")

	tokenDataExtractor := jwtToken.NewDataExtractor(publicKey)
	logger.Println("Token data extractor created")

	sessionStorage := redis.NewSessionRepository(cache, config.RefreshTokenTTL, coreLogger, tokenDataExtractor)
	logger.Println("Session storage created")

	rmqConfig := &broker.RMQConfig{
		Host:     config.RMQHost,
		Port:     config.RMQPort,
		User:     config.RMQUser,
		Password: config.RMQPassword,
	}

	rmq, err := broker.NewConnection(initCtx, rmqConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	logger.Println("RabbitMQ connected")

	passwordHasher := hasher.NewPasswordHasher(10)
	logger.Println("Password hasher created")

	emailSender, err := broker.NewEmailSender(ctx, rmq, coreLogger)
	if err != nil {
		logger.Fatalf("Failed to create email sender: %v", err)
	}
	logger.Println("Email sender created")

	registerUserUseCase := usecase.NewRegisterUserUseCase(
		userRepository,
		emailSender,
		verificationTokenRepository,
		passwordHasher,
		coreLogger,
	)
	logger.Println("Register user use case created")

	registerUserHandler := handler.NewRegisterUserHandler(registerUserUseCase)
	logger.Println("Register user handler created")

	verifyUserUseCase := usecase.NewVerifyUserUseCase(userRepository, verificationTokenRepository, coreLogger)
	logger.Println("Verify user use case created")

	verifyUserHandler := handler.NewVerifyUserHandler(verifyUserUseCase)
	logger.Println("Verify user handler created")

	resendVerificationEmailUseCase := usecase.NewResendVerificationEmailUseCase(
		verificationTokenRepository,
		userRepository,
		passwordHasher,
		emailSender,
	)
	logger.Println("Resend verification email use case created")

	resendVerificationEmailHandler := handler.NewResendVerificationEmailHandler(resendVerificationEmailUseCase)
	logger.Println("Resend verification email handler created")

	initiatePasswordRecoveryUseCase := usecase.NewInitiatePasswordRecoveryUseCase(
		userRepository,
		userDataUpdateTokenRepository,
		emailSender,
	)
	logger.Println("Initiate password recovery use case created")

	initiatePasswordRecoveryHandler := handler.NewInitiatePasswordRecoveryHandler(initiatePasswordRecoveryUseCase)
	logger.Println("Initiate password recovery handler created")

	confirmPasswordRecoveryUseCase := usecase.NewConfirmPasswordRecoveryUseCase(
		userRepository,
		userDataUpdateTokenRepository,
		passwordHasher,
		sessionStorage,
		coreLogger,
	)
	logger.Println("Confirm password recovery use case created")

	confirmPasswordRecoveryHandler := handler.NewConfirmPasswordRecoveryHandler(confirmPasswordRecoveryUseCase)
	logger.Println("Confirm password recovery handler created")

	loginUserUseCase := usecase.NewLoginUserUseCase(
		userRepository,
		sessionStorage,
		tokenGenerator,
		passwordHasher,
	)
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

	initiateEmailUpdateUseCase := usecase.NewInitiateEmailUpdateUseCase(
		userRepository,
		emailSender,
		userDataUpdateTokenRepository,
		passwordHasher,
	)
	logger.Println("Initiate email update use case created")

	initiateEmailUpdateHandler := handler.NewInitiateEmailUpdateHandler(initiateEmailUpdateUseCase)
	logger.Println("Initiate email update handler created")

	confirmEmailUpdateUseCase := usecase.NewConfirmEmailUpdateUseCase(
		userRepository,
		userDataUpdateTokenRepository,
		sessionStorage,
		coreLogger,
	)
	logger.Println("Confirm email update use case created")

	confirmEmailUpdateHandler := handler.NewConfirmEmailUpdateHandler(confirmEmailUpdateUseCase)
	logger.Println("Confirm email update handler created")

	passwordUpdateUseCase := usecase.NewPasswordUpdateUseCase(
		userRepository,
		sessionStorage,
		passwordHasher,
		coreLogger,
	)
	logger.Println("Password update use case created")

	passwordUpdateHandler := handler.NewPasswordUpdateHandler(passwordUpdateUseCase)
	logger.Println("Password update handler created")

	refreshAccessUseCase := usecase.NewRefreshAccessUseCase(
		userRepository,
		sessionStorage,
		tokenGenerator,
	)
	logger.Println("Refresh access use case created")

	refreshAccessHandler := handler.NewRefreshAccessHandler(refreshAccessUseCase)
	logger.Println("Refresh access handler created")

	updateUserProfileUseCase := usecase.NewUpdateUserProfileUseCase(userRepository)
	logger.Println("Update user profile use case created")

	updateUserProfileHandler := handler.NewUpdateUserProfileHandler(updateUserProfileUseCase)
	logger.Println("Update user profile handler created")

	profileSoftDeleteUseCase := usecase.NewProfileSoftDeleteUseCase(
		userRepository,
		sessionStorage,
		passwordHasher,
		coreLogger,
	)
	logger.Println("Profile soft delete use case created")

	profileSoftDeleteHandler := handler.NewProfileSoftDeleteHandler(profileSoftDeleteUseCase)
	logger.Println("Profile soft delete handler created")

	listUsersUseCase := usecase.NewListUsersUseCase(userRepository)
	logger.Println("List users use case created")

	listUsersHandler := handler.NewListUsersHandler(listUsersUseCase)
	logger.Println("List users handler created")

	switchUserRoleUseCase := usecase.NewSwitchUserRoleUseCase(
		userRepository,
		sessionStorage,
		coreLogger,
	)
	logger.Println("Switch user role use case created")

	switchUserRoleHandler := handler.NewSwitchUserRoleHandler(switchUserRoleUseCase)
	logger.Println("Switch user role handler created")

	publicKeyHandler, err := handler.NewPublicKeyHandler(publicKey)
	if err != nil {
		logger.Fatalf("Failed to create public key handler: %v", err)
	}
	logger.Println("Public key handler created")

	readyzHealthcheck := healthcheck.NewReadyzHealthcheck(db, cache, rmq)
	logger.Println("Readyz healthcheck created")

	readyzHealthcheckHandler := handler.NewHealthcheckHandler(readyzHealthcheck)
	logger.Println("Readyz healthcheck handler created")

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				healthCheckCtx, healthCheckCancel := context.WithTimeout(appCtx, 1*time.Second)

				readyzHealthcheck.Check(healthCheckCtx)

				healthCheckCancel()
				logger.Println(readyzHealthcheck.IsHealthy())
			case <-appCtx.Done():
				return
			}
		}
	}()

	router := chi.NewRouter()
	accessVerificationMiddleware := middlewares.NewAccessVerificationMiddleware(tokenDataExtractor)
	logger.Println("Access verification middleware created")

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
	router.Get("/api/auth/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.Get("/api/auth/readyz", readyzHealthcheckHandler.ServeHTTP)

	router.Group(func(r chi.Router) {
		r.Use(accessVerificationMiddleware.VerifyAccess)
		r.Get("/api/auth/profile/{userID}", handler.WithUserID(getUserHandler.HandlerWithUserID))
		r.Post("/api/auth/logout", handler.WithUserID(logoutUserHandler.HandlerWithUserID))
		r.Post("/api/auth/initiate-email-update", handler.WithUserID(initiateEmailUpdateHandler.HandlerWithUserID))
		r.Post("/api/auth/confirm-email-update", handler.WithUserID(confirmEmailUpdateHandler.HandlerWithUserID))
		r.Patch("/api/auth/password-update", handler.WithUserID(passwordUpdateHandler.HandlerWithUserID))
		r.Patch("/api/auth/update-user-profile", handler.WithUserID(updateUserProfileHandler.HandlerWithUserID))
		r.Patch("/api/auth/profile-soft-delete", handler.WithUserID(profileSoftDeleteHandler.HandlerWithUserID))
		r.Get("/api/auth/list-users", handler.WithUserID(listUsersHandler.HandlerWithUserID))
		r.Patch("/api/auth/switch-user-role/{userID}", handler.WithUserID(switchUserRoleHandler.HandlerWithUserID))
	})

	type Server struct {
		router *chi.Mux
		server *http.Server
	}

	serverConnection := &Server{
		router: router,
		server: &http.Server{
			Addr:              ":8080",
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			BaseContext: func(_ net.Listener) context.Context {
				return reqCtx
			},
		},
	}

	go func() {

		logger.Println("Starting server on port 8080")
		if err := serverConnection.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("Failed to start server: %v", err)
			appCancel()
		}
	}()

	<-appCtx.Done()
	logger.Println("Gracefully Shutting down...")

	shutDownCtx, shutDownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutDownCancel()

	if err := serverConnection.server.Shutdown(shutDownCtx); err != nil {
		logger.Printf("Failed to shutdown server gracefully: %v", err)
		cancelRequests()
		time.Sleep(1 * time.Second)
	} else {
		logger.Println("Server gracefully shut down")
	}

	if err := cache.Close(); err != nil {
		logger.Printf("Failed to close Redis connection: %v", err)
	} else {
		logger.Println("Redis connection closed")
	}

	if err := rmq.Close(); err != nil {
		logger.Printf("Failed to close RabbitMQ connection: %v", err)
	} else {
		logger.Println("RabbitMQ connection closed")
	}

	db.Close()
	logger.Println("Database connection closed")

	logger.Println("Application shut down")
}
