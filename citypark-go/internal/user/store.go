package user

import (
	"context"
	"time"

	"github.com/drewp1ter/citypark-go/internal/store"
)

const columns = `id, created_at, updated_at, name, email, phone, password_hash, legacy_password_hash, role`

type User struct {
	ID                 int64     `db:"id"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
	Name               string    `db:"name"`
	Email              *string   `db:"email"`
	Phone              string    `db:"phone"`
	PasswordHash       string    `db:"password_hash"`
	LegacyPasswordHash string    `db:"legacy_password_hash"`
	Role               string    `db:"role"`
}

// Response is the JSON shape of a user.
type Response struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"` // "" when the user has no email
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

func (u User) Response() Response {
	r := Response{ID: u.ID, Name: u.Name, Phone: u.Phone, Role: u.Role}
	if u.Email != nil {
		r.Email = *u.Email
	}
	return r
}

type CreateParams struct {
	Name         string
	Email        *string // nil = no email
	Phone        string
	PasswordHash string
	Role         string
}

type UpdateParams struct {
	Name  string
	Email *string // nil = remove the email
	Phone string
}

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, p CreateParams) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u,
		`INSERT INTO users (name, email, phone, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+columns,
		p.Name, p.Email, p.Phone, p.PasswordHash, p.Role)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Store) ByID(ctx context.Context, id int64) (User, error) {
	return s.one(ctx, `SELECT `+columns+` FROM users WHERE id = $1`, id)
}

func (s *Store) ByEmail(ctx context.Context, email string) (User, error) {
	return s.one(ctx, `SELECT `+columns+` FROM users WHERE email = $1`, email)
}

func (s *Store) ByPhone(ctx context.Context, phone string) (User, error) {
	return s.one(ctx, `SELECT `+columns+` FROM users WHERE phone = $1`, phone)
}

func (s *Store) Update(ctx context.Context, id int64, p UpdateParams) (User, error) {
	return s.one(ctx,
		`UPDATE users SET name = $2, email = $3, phone = $4, updated_at = now()
		 WHERE id = $1
		 RETURNING `+columns,
		id, p.Name, p.Email, p.Phone)
}

// SetPasswordHash stores a bcrypt hash and clears legacy_password_hash.
func (s *Store) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = $2, legacy_password_hash = '', updated_at = now()
		 WHERE id = $1`,
		id, hash)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// UpsertCustomer returns the user with this phone, creating it (role "user", no email,
// no password) when missing. An existing user gets the new name.
func (s *Store) UpsertCustomer(ctx context.Context, name, phone string) (User, error) {
	var u User
	err := s.db.GetContext(ctx, &u,
		`INSERT INTO users (name, phone) VALUES ($1, $2)
		 ON CONFLICT (phone) DO UPDATE SET name = EXCLUDED.name, updated_at = now()
		 RETURNING `+columns,
		name, phone)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// one runs a query that returns at most one user row and maps "no rows" to store.ErrNotFound.
func (s *Store) one(ctx context.Context, query string, args ...any) (User, error) {
	var u User
	if err := s.db.GetContext(ctx, &u, query, args...); err != nil {
		return User{}, store.NotFound(err)
	}
	return u, nil
}
