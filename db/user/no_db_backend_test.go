package user

import (
	"context"
	"reflect"
	"testing"
)

func TestNoDatabaseBackendCreate(t *testing.T) {
	u := User{
		Username: "john",
		Password: "Doe12345",
	}
	ctx := context.Background()
	var b NoDatabaseBackend
	if err := b.Create(ctx, u); err == nil {
		t.Errorf("wanted error")
	}
}

func TestNoDatabaseBackendRead(t *testing.T) {
	u := User{
		Username: "john",
	}
	ctx := context.Background()
	var b NoDatabaseBackend
	got, err := b.Read(ctx, u)
	if err != nil {
		t.Errorf("unwanted error: %v", err)
	}
	if want := &u; !reflect.DeepEqual(want, got) {
		t.Errorf("wanted %v, got %v", want, got)
	}
}

func TestNoDatabaseBackendUpdatePassword(t *testing.T) {
	u := User{
		Username: "john",
		Password: "Doe12345",
	}
	ctx := context.Background()
	var b NoDatabaseBackend
	if err := b.UpdatePassword(ctx, u); err == nil {
		t.Errorf("wanted error")
	}
}

func TestNoDatabaseBackendUpdatePointsIncrement(t *testing.T) {
	usernamePoints := map[string]int{
		"john": 10,
	}
	ctx := context.Background()
	var b NoDatabaseBackend
	if err := b.UpdatePointsIncrement(ctx, usernamePoints); err == nil {
		t.Errorf("wanted error")
	}
}

// Delete returns an error.
func TestNoDatabaseBackendDelete(t *testing.T) {
	u := User{
		Username: "john",
		Password: "Doe12345",
	}
	ctx := context.Background()
	var b NoDatabaseBackend
	if err := b.Delete(ctx, u); err == nil {
		t.Errorf("wanted error")
	}
}
