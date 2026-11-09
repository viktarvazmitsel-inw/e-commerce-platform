package mongo

import (
	"context"
	"errors"
	"fmt"

	"github.com/shop/services/notification/internal/domain"
	"github.com/shop/services/notification/internal/usecase"
	"go.mongodb.org/mongo-driver/v2/bson"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const notificationsCollection = "notifications"

type NotificationRepository struct {
	coll *drivermongo.Collection
}

func NewNotificationRepository(db *drivermongo.Database) (*NotificationRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("mongo database is nil")
	}

	repo := &NotificationRepository{coll: db.Collection(notificationsCollection)}
	if err := repo.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *NotificationRepository) ensureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []drivermongo.IndexModel{
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
		{Keys: bson.D{
			{Key: "status", Value: 1},
			{Key: "claimed_until", Value: 1},
			{Key: "updated_at", Value: 1},
		}},
	})
	return err
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, notification *domain.Notification) error {
	if notification == nil {
		return fmt.Errorf("notification is nil")
	}
	_, err := r.coll.InsertOne(ctx, FromDomain(notification))
	return err
}

func (r *NotificationRepository) GetNotificationByID(ctx context.Context, id string) (*domain.Notification, error) {
	var doc notificationDocument
	err := r.coll.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	if errors.Is(err, drivermongo.ErrNoDocuments) {
		return nil, usecase.ErrNotificationNotFound
	}
	if err != nil {
		return nil, err
	}
	notification := doc.ToDomain()
	return &notification, nil
}

func (r *NotificationRepository) GetNotifications(ctx context.Context, page, quantity int) ([]domain.Notification, int, error) {
	total64, err := r.coll.CountDocuments(ctx, bson.D{})
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if quantity <= 0 {
		return []domain.Notification{}, int(total64), nil
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64(page-1) * int64(quantity)).
		SetLimit(int64(quantity))
	cursor, err := r.coll.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var docs []notificationDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, 0, err
	}
	return documentsToDomain(docs), int(total64), nil
}

func (r *NotificationRepository) UpdateNotification(ctx context.Context, notification *domain.Notification) error {
	if notification == nil {
		return fmt.Errorf("notification is nil")
	}

	result, err := r.coll.ReplaceOne(ctx, bson.D{{Key: "_id", Value: notification.ID}}, FromDomain(notification))
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return usecase.ErrNotificationNotFound
	}
	return nil
}

func documentsToDomain(docs []notificationDocument) []domain.Notification {
	notifications := make([]domain.Notification, 0, len(docs))
	for _, doc := range docs {
		notifications = append(notifications, doc.ToDomain())
	}
	return notifications
}

var _ usecase.NotificationRepository = (*NotificationRepository)(nil)
