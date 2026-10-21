package user

import (
	"context"
	"errors"

	"github.com/drewp1ter/citypark-go/internal/auth"
	"github.com/drewp1ter/citypark-go/internal/store"
	"github.com/drewp1ter/citypark-go/internal/validate"
)

type AdminParams struct{ Email, Password, Phone string }

// EnsureAdmin creates the admin account when no user has this email or this phone.
// It returns true when it created the account.
func EnsureAdmin(ctx context.Context, s *Store, p AdminParams) (created bool, err error) {
	phone, ok := validate.NormalizePhone(p.Phone)
	if !ok {
		return false, errors.New("invalid admin phone")
	}
	email := normalizeEmail(p.Email)

	for _, lookup := range []func() (User, error){
		func() (User, error) { return s.ByEmail(ctx, email) },
		func() (User, error) { return s.ByPhone(ctx, phone) },
	} {
		_, err := lookup()
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return false, err
		}
	}

	hash, err := auth.HashPassword(p.Password)
	if err != nil {
		return false, err
	}
	_, err = s.Create(ctx, CreateParams{
		Name:         "admin",
		Email:        &email,
		Phone:        phone,
		PasswordHash: hash,
		Role:         auth.RoleAdmin,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
