package main

import (
	"authorization/internal/config"
	"authorization/internal/infrastructure/broker"
	messageBroker "authorization/internal/infrastructure/broker"
	"authorization/internal/infrastructure/db/postgre"
	"authorization/internal/infrastructure/db/redis"
	tokenStorage "authorization/internal/infrastructure/db/redis"
	"authorization/internal/infrastructure/hasher"
	coreLogger "authorization/internal/infrastructure/logger"
	"authorization/internal/interface/http/handler"
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
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	}

	cache, err := redis.NewRedisClient(ctx, redisConfig)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	logger.Println("Redis connected")

	verificationTokenRepository := tokenStorage.NewVerificationTokenRepository(cache, config.VerificationTokenTTL)
	logger.Println("Verification token storage created")

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

	coreLogger := coreLogger.NewLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	logger.Println("Core logger created")

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

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Post("/api/auth/register", registerUserHandler.ServeHTTP)
	router.Post("/api/auth/verify", verifyUserHandler.ServeHTTP)
	router.Post("/api/auth/resend-verification-email", resendVerificationEmailHandler.ServeHTTP)

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

	// router.Group(func(r chi.Router) {
	// 	r.Use(accessVerificationMiddleware.VerifyAccess)
	// 	r.Get("/api/auth/mwcheck", func(w http.ResponseWriter, r *http.Request) {
	// 		response.RespondJSON(w, http.StatusOK, &response.Response{
	// 			Success: true,
	// 			Data: map[string]string{
	// 				"message": "Middleware check passed",
	// 			},
	// 		})
	// 	})
	// })

	http.ListenAndServe(":8080", router)
}
