// Package mongo implements database structures for mongodb.
package mongo

import (
	"context"
	"fmt"

	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName   = "selene-bananas-db"
	collectionName = "users"
	usernameField  = "username"
	passwordField  = "password"
	pointsField    = "points"
)

// UserBackend is a backend manager for a users collection.
type UserBackend struct {
	Users *mongo.Collection
	db.Config
}

// NewUserBackend creates a backend manager for the users collection.
func NewUserBackend(ctx context.Context, cfg db.Config, databaseURL string) (*UserBackend, error) {
	clientOptions := options.Client()
	clientOptions.ApplyURI(databaseURL)
	ctx, cancelFunc := context.WithTimeout(ctx, cfg.QueryPeriod)
	defer cancelFunc()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connecting to mongodb: %w", err)
	}
	databaseName := "selene-bananas-db" // TODO: should this be hard-coded?
	database := client.Database(databaseName)
	users := database.Collection("users")
	ub := UserBackend{
		Users:  users,
		Config: cfg,
	}
	return &ub, nil
}

// Setup initializes the backend with appropriate triggers
func (ub *UserBackend) Setup(ctx context.Context) error {
	indexOptions := options.Index()
	indexOptions.SetUnique(true)
	document := d(e(usernameField, 1))
	model := mongo.IndexModel{
		Keys:    document,
		Options: indexOptions,
	}
	indexes := ub.Users.Indexes()
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	_, err := indexes.CreateOne(ctx, model)
	if err != nil {
		return fmt.Errorf("creating unique username index: %w", err)
	}
	return nil
}

// Create adds the username/password pair.
func (ub *UserBackend) Create(ctx context.Context, u user.User) error {
	document := d(
		e(usernameField, u.Username),
		e(passwordField, u.Password),
	)
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	if _, err := ub.Users.InsertOne(ctx, document); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Read validates the username/password pair and gets the points.
func (ub *UserBackend) Read(ctx context.Context, u user.User) (*user.User, error) {
	filter := d(e(usernameField, u.Username))
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	result := ub.Users.FindOne(ctx, filter)
	var u2 user.User
	if err := result.Decode(&u2); err != nil {
		if err == mongo.ErrNoDocuments {
			// TODO: This is sloppy to have a dependency on the user package for an error return.
			//       Could the signature be altered to be more clear? : (User, ok, error)
			return nil, user.ErrIncorrectLogin
		}
		return nil, fmt.Errorf("reading user: %w", err)
	}
	return &u2, nil
}

// UpdatePassword updates the password for user identified by the username.
func (ub *UserBackend) UpdatePassword(ctx context.Context, u user.User) error {
	filter := d(e(usernameField, u.Username))
	update := d(e("$set", d(e(passwordField, u.Password))))
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	if _, err := ub.Users.UpdateOne(ctx, filter, update); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement changes the points for all of the usernames.
func (ub *UserBackend) UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error {
	writeModels := make([]mongo.WriteModel, 0, len(usernamePoints))
	for username, points := range usernamePoints {
		filter := d(e(usernameField, username))
		update := d(e("$inc", d(e(pointsField, points))))
		m := mongo.NewUpdateOneModel()
		m.SetFilter(filter)
		m.SetUpdate(update)
		writeModels = append(writeModels, m)
	}
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	if _, err := ub.Users.BulkWrite(ctx, writeModels); err != nil {
		return fmt.Errorf("updating user points: %w", err)
	}
	return nil
}

// Delete removes the user.
func (ub *UserBackend) Delete(ctx context.Context, u user.User) error {
	filter := d(e(usernameField, u.Username))
	ctx, cancelFunc := context.WithTimeout(ctx, ub.Config.QueryPeriod)
	defer cancelFunc()
	if _, err := ub.Users.DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// d is a helper function to create bson.D elements.
func d(e ...bson.E) bson.D {
	return bson.D(e)
}

// e is a helper function to create bson.E elements.
func e(key string, value interface{}) bson.E {
	return bson.E{Key: key, Value: value}
}
