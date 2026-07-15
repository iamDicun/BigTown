package repository

import (
	"context"
	"database/sql"

	"backend/internal/module/user/entity"
	"backend/internal/module/user/port"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

var _ port.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM app_user WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (r *UserRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, fullName string, email string) (*entity.User, error) {
	var u entity.User
	err := tx.QueryRowContext(
		ctx,
		`INSERT INTO app_user (full_name, email)
		 VALUES ($1, $2)
		 RETURNING id::text, full_name, email, role`,
		fullName,
		email,
	).Scan(&u.ID, &u.FullName, &u.Email, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var u entity.User
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id::text, full_name, email, role FROM app_user WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.FullName, &u.Email, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
	var u entity.User
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id::text, full_name, email, role FROM app_user WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.FullName, &u.Email, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]entity.User, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id::text, full_name, email, role FROM app_user`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		var u entity.User
		if err := rows.Scan(&u.ID, &u.FullName, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
