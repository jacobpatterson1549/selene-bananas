// Package firestore use a google cloud firestore database.
package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/jacobpatterson1549/selene-bananas/db"
	"github.com/jacobpatterson1549/selene-bananas/db/user"
)

const (
	collectionName = "users"
	usernameField  = "username"
	passwordField  = "password"
	pointsField    = "points"
)

// UserBackend is a backend manager for a users collection.
type UserBackend struct {
	client *firestore.Client
	db.Config
}

func (ub *UserBackend) usersCollection() *firestore.CollectionRef {
	return ub.client.Collection("services").Doc("selene-bananas").Collection("users")
}

// NewUserBackend creates a backend manager for users.
func NewUserBackend(ctx context.Context, cfg db.Config, projectID string) (*UserBackend, error) {
	ub := UserBackend{
		Config: cfg,
	}
	client, err := firestore.NewClient(ctx, projectID) // do not timeout context - the client is used by the backend
	if err != nil {
		return nil, fmt.Errorf("creating firestore client: %w", err)
	}
	ub.client = client
	return &ub, nil
}

// withTimeoutContext configures the context to timeout when running the function.
func (ub *UserBackend) withTimeoutContext(ctx context.Context, f func(ctx context.Context) error) error {
	ctx, cancelFunc := context.WithTimeout(ctx, ub.QueryPeriod)
	defer cancelFunc()
	return f(ctx)
}

// Create adds the username/password pair.
func (ub *UserBackend) Create(ctx context.Context, u user.User) error {
	if err := ub.withTimeoutContext(ctx, func(ctx context.Context) error {
		users := ub.usersCollection()
		docRef := users.Doc(u.Username)
		m := map[string]interface{}{
			passwordField: u.Password,
		}
		_, err := docRef.Create(ctx, m) // returns an error if user already exists
		return err
	}); err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// Get validates the username/password pair and gets the points.
func (ub *UserBackend) Read(ctx context.Context, u user.User) (*user.User, error) {
	if err := ub.withTimeoutContext(ctx, func(ctx context.Context) error {
		users := ub.usersCollection()
		docRef := users.Doc(u.Username)
		snapshot, err := docRef.Get(ctx)
		if err != nil {
			if snapshot != nil && !snapshot.Exists() {
				return user.ErrIncorrectLogin
			}
			return err
		}
		if err := snapshot.DataTo(&u); err != nil {
			return err
		}
		return err
	}); err != nil {
		if err == user.ErrIncorrectLogin {
			return nil, err
		}
		return nil, fmt.Errorf("reading user: %w", err)
	}
	return &u, nil
}

// UpdatePassword updates the password for user identified by the username.
func (ub *UserBackend) UpdatePassword(ctx context.Context, u user.User) error {
	if err := ub.withTimeoutContext(ctx, func(ctx context.Context) error {
		users := ub.usersCollection()
		docRef := users.Doc(u.Username)
		u := []firestore.Update{
			{
				Path:  passwordField,
				Value: u.Password,
			},
		}
		_, err := docRef.Update(ctx, u)
		return err
	}); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}

// UpdatePointsIncrement increments the points for all of the usernames.
func (ub *UserBackend) UpdatePointsIncrement(ctx context.Context, usernamePoints map[string]int) error {
	if err := ub.withTimeoutContext(ctx, func(ctx context.Context) error {
		users := ub.usersCollection()
		b := ub.client.Batch()
		for username, points := range usernamePoints {
			d := users.Doc(username)
			u := []firestore.Update{
				{
					Path:  pointsField,
					Value: firestore.FieldTransformIncrement(points),
				},
			}
			b.Update(d, u)
		}
		_, err := b.Commit(ctx)
		return err
	}); err != nil {
		return fmt.Errorf("incrementing user points: %w", err)
	}
	return nil
}

// Delete removes the user.
func (ub *UserBackend) Delete(ctx context.Context, u user.User) error {
	if err := ub.withTimeoutContext(ctx, func(ctx context.Context) error {
		users := ub.usersCollection()
		docRef := users.Doc(u.Username)
		_, err := docRef.Delete(ctx, firestore.Exists)
		return err
	}); err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}
	return nil
}
