package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/shop/services/notification/internal/config"
	"github.com/shop/services/notification/internal/infrastructure/broker"
	"github.com/shop/services/notification/internal/infrastructure/db/mongo"
	"github.com/shop/services/notification/internal/infrastructure/mail"
	"github.com/shop/services/notification/internal/infrastructure/token"
	brokerHandler "github.com/shop/services/notification/internal/interface/broker/handler"
	httpHandler "github.com/shop/services/notification/internal/interface/http/handler"
	httpMiddleware "github.com/shop/services/notification/internal/interface/http/middleware"
	"github.com/shop/services/notification/internal/usecase"
)

func main() {
	logger := log.New(os.Stdout, "notification-back: ", log.LstdFlags)

	err := godotenv.Load()
	if err != nil {
		logger.Println("Error loading .env file: %v", err)
	}

	config := config.Load()
	logger.Println("Config loaded")

	ctx := context.Background()

	reqCtx, cancelRequests := context.WithCancel(ctx)
	defer cancelRequests()

	appCtx, appCancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer appCancel()

	initCtx, initCancel := context.WithTimeout(ctx, 10*time.Second)
	defer initCancel()

	brokerConfig := &broker.RMQConfig{
		Host:     config.RMQHost,
		Port:     config.RMQPort,
		User:     config.RMQUser,
		Password: config.RMQPassword,
	}

	services := broker.BrokerServiceList{
		List: []string{"auth"},
	}

	/*RMQ load*/
	messageBroker, err := broker.NewConnection(initCtx, brokerConfig, &services)
	if err != nil {
		logger.Printf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	logger.Println("RabbitMQ connected")

	dbConfig := &mongo.Config{
		Host:     config.MongoHost,
		Port:     config.MongoPort,
		User:     config.MongoUser,
		Password: config.MongoPassword,
		Database: config.MongoDatabase,
	}

	client, err := mongo.NewClient(initCtx, *dbConfig)
	if err != nil {
		logger.Printf("Failed to connect to MongoDB: %v", err)
		return
	}
	defer client.Disconnect(ctx)
	logger.Println("MongoDB loaded")

	/*MongoDB load*/
	db := client.Database(dbConfig.Database)
	if db == nil {
		logger.Printf("Failed to get database: %v", err)
		return
	}
	logger.Println("Database loaded")

	notificationRepository, err := mongo.NewNotificationRepository(db)
	if err != nil {
		logger.Printf("Failed to create notification repository: %v", err)
		return
	}
	logger.Println("Notification repository created")

	/*Email sender load*/
	emailSender, err := mail.NewEmailSender(config.SMTPHost, config.SMTPPort, config.SMTPFrom)
	if err != nil {
		logger.Printf("Failed to create email sender: %v", err)
		return
	}
	logger.Println("Email sender created")

	/*JWT load*/
	publicKey, err := token.FetchPublicKey(initCtx, config.AuthPublicKeyURL, 3, 1*time.Second, nil)
	if err != nil {
		logger.Printf("Failed to fetch public key: %v", err)
		return
	}
	logger.Println("Public key fetched")

	keyParser, err := token.NewParser(publicKey)
	if err != nil {
		logger.Printf("Failed to create key parser: %v", err)
		return
	}
	logger.Println("Key parser created")

	/*Use cases load*/
	getNotificationUseCase := usecase.NewGetNotificationUseCase(notificationRepository)
	logger.Println("Get notification use case created")

	listNotificationsUseCase := usecase.NewListNotificationsUseCase(notificationRepository)
	logger.Println("List notifications use case created")

	sendNotificationUseCase := usecase.NewSendNotificationUseCase(notificationRepository, emailSender)
	logger.Println("Send notification use case created")

	/*Handlers load*/
	getNotificationHandler := httpHandler.NewGetNotificationHandler(getNotificationUseCase)
	logger.Println("Get notification handler created")

	listNotificationsHandler := httpHandler.NewListNotificationsHandler(listNotificationsUseCase)
	logger.Println("List notifications handler created")

	sendVerificationEmailHandler := brokerHandler.NewSendVerificationEmailHandler(sendNotificationUseCase, messageBroker)
	logger.Println("Send verification email handler created")

	/*HTTP router load*/
	router := chi.NewRouter()
	accessVerificationMiddleware, err := httpMiddleware.NewAccessVerificationMiddleware(keyParser)
	if err != nil {
		logger.Printf("Failed to create access verification middleware: %v", err)
		return
	}
	logger.Println("Access verification middleware created")

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Use(accessVerificationMiddleware.VerifyAccess)
	router.Get("/api/note/notifications/{id}", getNotificationHandler.ServeHTTP)
	router.Get("/api/note/notifications", listNotificationsHandler.ServeHTTP)

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

	messageHandlerMap := map[string]broker.MessageHandler{
		"auth.user.created":        sendVerificationEmailHandler.Handle,
		"auth.user.password.reset": sendVerificationEmailHandler.Handle,
	}

	messageRouter := broker.NewBrokerRouter(messageHandlerMap)
	logger.Println("Message router created")

	go func() {
		logger.Println("Starting message broker")
		if err := messageBroker.Serve(appCtx, messageRouter, &services); err != nil {
			logger.Printf("Failed to start message broker: %v", err)
			appCancel()
		}
	}()

	<-appCtx.Done()
}
