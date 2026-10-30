package customer

import "context"

// Method is a public description of a linked customer login method.
type Method struct {
	Provider string `json:"provider"`
	Verified bool   `json:"verified"`
}

func (s *Store) Methods(ctx context.Context, userID int64) ([]Method, error) {
	methods := make([]Method, 0, 3)
	err := s.db.SelectContext(ctx, &methods, `SELECT i.provider,CASE WHEN i.provider='email' THEN u.is_email_confirmed ELSE true END AS verified
FROM customer_identities i JOIN users u ON u.id=i.user_id WHERE i.user_id=$1
ORDER BY CASE i.provider WHEN 'email' THEN 1 WHEN 'google' THEN 2 ELSE 3 END`, userID)
	return methods, err
}
