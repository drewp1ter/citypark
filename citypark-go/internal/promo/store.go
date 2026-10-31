package promo

import (
	"context"
	"github.com/drewp1ter/citypark-go/internal/store"
)

type Store struct{ db store.DBTX }

func NewStore(db store.DBTX) *Store { return &Store{db: db} }

const columns = `id,code,percentage,enabled,target,minimum_eligible_subtotal_kopecks,maximum_discount_kopecks,starts_at,ends_at,created_at,updated_at`

// List returns newest campaigns first, including disabled campaigns.
func (s *Store) List(ctx context.Context) ([]Campaign, error) {
	out := []Campaign{}
	if err := s.db.SelectContext(ctx, &out, `SELECT `+columns+` FROM promo_codes ORDER BY id DESC`); err != nil {
		return nil, err
	}
	for i := range out {
		if err := s.targets(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ByID and ByCode acquire a shared row lock. Pricing callers must supply a
// transaction so the lock remains held while selections are read and used.
func (s *Store) ByID(ctx context.Context, id int64) (Campaign, error) {
	return s.read(ctx, `id=$1`, id)
}
func (s *Store) ByCode(ctx context.Context, code string) (Campaign, error) {
	return s.read(ctx, `code=$1`, normalizeCode(code))
}
func (s *Store) read(ctx context.Context, where string, arg any) (Campaign, error) {
	var c Campaign
	if err := s.db.GetContext(ctx, &c, `SELECT `+columns+` FROM promo_codes WHERE `+where+` FOR SHARE`, arg); err != nil {
		return c, store.NotFound(err)
	}
	err := s.targets(ctx, &c)
	return c, err
}
func (s *Store) targets(ctx context.Context, c *Campaign) error {
	c.ProductIDs = []int64{}
	c.CategoryIDs = []int64{}
	if err := s.db.SelectContext(ctx, &c.ProductIDs, `SELECT product_id FROM promo_code_products WHERE promo_code_id=$1 ORDER BY product_id`, c.ID); err != nil {
		return err
	}
	return s.db.SelectContext(ctx, &c.CategoryIDs, `SELECT category_id FROM promo_code_categories WHERE promo_code_id=$1 ORDER BY category_id`, c.ID)
}

// Create requires a caller-owned transaction. The caller must roll back on error.
func (s *Store) Create(ctx context.Context, in Input) (Campaign, error) {
	in, err := normalize(in)
	if err != nil {
		return Campaign{}, err
	}
	var id int64
	err = s.db.GetContext(ctx, &id, `INSERT INTO promo_codes(code,percentage,enabled,target,minimum_eligible_subtotal_kopecks,maximum_discount_kopecks,starts_at,ends_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, in.Code, in.Percentage, in.Enabled, in.Target, in.MinimumEligibleSubtotal, in.MaximumDiscount, in.StartsAt, in.EndsAt)
	if err != nil {
		return Campaign{}, err
	}
	if err = s.replace(ctx, id, in); err != nil {
		return Campaign{}, err
	}
	return s.ByID(ctx, id)
}

// Update requires a caller-owned transaction and preserves the immutable code.
// Its exclusive lock serializes replacement with shared pricing reads.
func (s *Store) Update(ctx context.Context, id int64, in Input) (Campaign, error) {
	in, err := normalize(in)
	if err != nil {
		return Campaign{}, err
	}
	var code string
	if err = s.db.GetContext(ctx, &code, `SELECT code FROM promo_codes WHERE id=$1 FOR UPDATE`, id); err != nil {
		return Campaign{}, store.NotFound(err)
	}
	if code != in.Code {
		return Campaign{}, &ValidationError{"code", "immutable"}
	}
	_, err = s.db.ExecContext(ctx, `UPDATE promo_codes SET percentage=$2,enabled=$3,target=$4,minimum_eligible_subtotal_kopecks=$5,maximum_discount_kopecks=$6,starts_at=$7,ends_at=$8,updated_at=now() WHERE id=$1`, id, in.Percentage, in.Enabled, in.Target, in.MinimumEligibleSubtotal, in.MaximumDiscount, in.StartsAt, in.EndsAt)
	if err != nil {
		return Campaign{}, err
	}
	if err = s.replace(ctx, id, in); err != nil {
		return Campaign{}, err
	}
	return s.ByID(ctx, id)
}
func (s *Store) replace(ctx context.Context, id int64, in Input) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM promo_code_products WHERE promo_code_id=$1`, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM promo_code_categories WHERE promo_code_id=$1`, id); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO promo_code_products(promo_code_id,product_id) SELECT $1,unnest($2::bigint[])`, id, in.ProductIDs); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO promo_code_categories(promo_code_id,category_id) SELECT $1,unnest($2::bigint[])`, id, in.CategoryIDs)
	return err
}
