package storage

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/dmitrorezn/feedero/domain"
)

type Storage struct {
	feedbacks *mongo.Collection
	comments  *mongo.Collection
	products  *mongo.Collection
	owners    *mongo.Collection
}

type Cfg struct {
	FeedbacksCollectionName string
	CommentsCollectionName  string
	ProductsCollectionName  string
}

func New(db *mongo.Database, cfg Cfg) *Storage {
	return &Storage{
		feedbacks: db.Collection(cfg.FeedbacksCollectionName),
	}
}

type ID = domain.ID
type Feedback struct {
	ID         ID     `bson:"_id"`
	Anonymous  bool   `bson:"anonymous"`
	UserID     ID     `bson:"userID"`
	Content    string `bson:"content"`
	MediaIDs   []ID   `bson:"medias"`
	ShowStatus int32  `bson:"showStatus"`
}

type Media struct {
	ID   string `bson:"_id"`
	Type string `bson:"type"`
	Link string `bson:"link"`
}

func feedbackToDBFeedback(f domain.Feedback) Feedback {
	return Feedback{
		ID:        f.ID,
		Anonymous: f.Anonymous,
	}
}

func (f Feedback) toDomain() domain.Feedback {
	return domain.Feedback{
		ID:         f.ID,
		Anonymous:  f.Anonymous,
		UserID:     f.UserID,
		Content:    f.Content,
		MediaIDs:   f.MediaIDs,
		ShowStatus: domain.ShowStatus(f.ShowStatus),
	}
}

var ErrNotFound = errors.New("not found")

func handleMongoError(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return errors.Join(err, ErrNotFound)
	}

	return err
}

func (s *Storage) InsertFeedback(ctx context.Context, f domain.Feedback) error {
	_, err := s.feedbacks.InsertOne(ctx, feedbackToDBFeedback(f))
	if err != nil {
		return handleMongoError(err)
	}

	return nil
}

func (s *Storage) GetFeedbackByID(ctx context.Context, id domain.ID) (domain.Feedback, error) {
	var feedback Feedback
	err := s.feedbacks.FindOne(ctx, bson.D{
		{Key: "_id", Value: id},
	}).Decode(&feedback)
	if err != nil {
		return domain.Feedback{}, handleMongoError(err)
	}

	return feedback.toDomain(), nil
}

type Owner struct {
	ID            ID     `bson:"_id"`
	Name          string `bson:"name" `
	Surname       string `bson:"surname"`
	ProductIDs    []ID   `bson:"product_ids"`
	EmailVerified bool   `bson:"email_verified"`
	PhoneVerified bool   `bson:"phone_verified"`
}

func (s *Storage) InsertOwner(ctx context.Context, o domain.Owner) error {
	_, err := s.owners.InsertOne(ctx, ownerToDBModel(o))
	if err != nil {
		return handleMongoError(err)
	}

	return nil
}
func ownerToDBModel(o domain.Owner) Owner {
	return Owner{
		ID:            o.ID,
		Name:          o.Name,
		Surname:       o.Surname,
		ProductIDs:    o.ProductIDs,
		EmailVerified: o.EmailVerified,
		PhoneVerified: o.PhoneVerified,
	}
}

func (o Owner) toDomain() domain.Owner {
	return domain.Owner{
		ID:            o.ID,
		Name:          o.Name,
		Surname:       o.Surname,
		ProductIDs:    o.ProductIDs,
		EmailVerified: o.EmailVerified,
		PhoneVerified: o.PhoneVerified,
	}
}

type Product struct {
	ID      ID     `bson:"_id"`
	OwnerID ID     `bson:"owner_id"`
	Name    string `bson:"name"`
	Domain  string `bson:"domain"`
	APIKey  string `bson:"api_Key"`
}

func (o Product) toDomain() domain.Product {
	return domain.Product{
		ID:      o.ID,
		Name:    o.Name,
		OwnerID: o.OwnerID,
		Domain:  o.Domain,
		APIKey:  o.APIKey,
	}
}

func productToDBModel(p domain.Product) Product {
	return Product{
		ID:      p.ID,
		Name:    p.Name,
		OwnerID: p.OwnerID,
		Domain:  p.Domain,
		APIKey:  p.APIKey,
	}
}

func (s *Storage) GetProductsByOwnerID(ctx context.Context, ownerID domain.ID) (domain.Owner, error) {
	var owner Owner
	err := s.products.FindOne(ctx, bson.D{
		{"ownerID", ownerID},
	}).Decode(&owner)
	if err != nil {
		return domain.Owner{}, handleMongoError(err)
	}

	return owner.toDomain(), nil
}

func (s *Storage) GetProductsByAPIKey(ctx context.Context, key string) (domain.Product, error) {
	var product Product
	err := s.products.FindOne(ctx, bson.D{
		{Key: "api_key", Value: key},
	}).Decode(&product)
	if err != nil {
		return domain.Product{}, handleMongoError(err)
	}

	return product.toDomain(), nil
}

func (s *Storage) GetProductsByID(ctx context.Context, id domain.ID) (domain.Product, error) {
	var product Product
	err := s.products.FindOne(ctx, bson.D{
		{Key: "_id", Value: id},
	}).Decode(&product)
	if err != nil {
		return domain.Product{}, handleMongoError(err)
	}

	return product.toDomain(), nil
}

func (s *Storage) UpdateProduct(
	ctx context.Context,
	id domain.ID,
	upd func(app *domain.Product) error,
) error {
	product, err := s.GetProductsByID(ctx, id)
	if err != nil {
		return err
	}
	if err = upd(&product); err != nil {
		return err
	}
	if _, err = s.feedbacks.UpdateByID(ctx, id, productToDBModel(product)); err != nil {
		return err
	}

	return err
}
