package postgre

import (
	"authorization/internal/domain"
	"authorization/internal/usecase"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	conn *pgxpool.Pool
}

var userTable = "users"

func NewUserRepository(conn *pgxpool.Pool) *userRepository {
	return &userRepository{conn: conn}
}

func (r *userRepository) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE email=$1)", userTable)
	err := r.conn.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	model := FromDomain(user)
	query := fmt.Sprintf("INSERT INTO %s (id, email, password_hash, name, surname, phone, role, is_verified, is_active) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)", userTable)
	_, err := r.conn.Exec(
		ctx,
		query,
		model.ID,
		model.Email,
		model.PasswordHash,
		model.Name,
		model.Surname,
		model.Phone,
		model.Role,
		model.IsVerified,
		model.IsActive,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return usecase.ErrEmailAlreadyTaken
		}
		return err
	}
	return nil
}

func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	model := FromDomain(user)
	query := fmt.Sprintf("UPDATE %s SET email=$1, password_hash=$2, name=$3, surname=$4, phone=$5, role=$6, is_verified=$7, is_active=$8 WHERE id=$9", userTable)
	cmdTag, err := r.conn.Exec(
		ctx,
		query,
		model.Email,
		model.PasswordHash,
		model.Name,
		model.Surname,
		model.Phone,
		model.Role,
		model.IsVerified,
		model.IsActive,
		model.ID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return usecase.ErrEmailAlreadyTaken
		}
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return usecase.ErrUserIdNotFound
	}
	return nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := fmt.Sprintf("SELECT id, email, password_hash, name, surname, phone, role, is_verified, is_active FROM %s WHERE email=$1", userTable)
	row := r.conn.QueryRow(ctx, query, email)
	var model UserModel
	err := row.Scan(
		&model.ID,
		&model.Email,
		&model.PasswordHash,
		&model.Name,
		&model.Surname,
		&model.Phone,
		&model.Role,
		&model.IsVerified,
		&model.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, usecase.ErrUserEmailNotFound
		}

		return nil, err
	}

	return model.ToDomain(), nil
}

func (r *userRepository) GetUserByID(ctx context.Context, ID string) (*domain.User, error) {
	query := fmt.Sprintf("SELECT id, email, password_hash, name, surname, phone, role, is_verified, is_active FROM %s WHERE id=$1", userTable)
	row := r.conn.QueryRow(ctx, query, ID)
	var model UserModel
	err := row.Scan(
		&model.ID,
		&model.Email,
		&model.PasswordHash,
		&model.Name,
		&model.Surname,
		&model.Phone,
		&model.Role,
		&model.IsVerified,
		&model.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, usecase.ErrUserIdNotFound
		}

		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *userRepository) GetUserList(ctx context.Context, page, quantity int) ([]domain.User, int, error) {
	if page < 1 {
		page = 1
	}
	if quantity < 1 {
		quantity = 1
	}
	offset := (page - 1) * quantity

	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", userTable)
	if err := r.conn.QueryRow(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to fetch total user count: %w", err)
	}

	if totalCount == 0 {
		return []domain.User{}, 0, nil
	}

	query := fmt.Sprintf("SELECT id, email, password_hash, name, surname, phone, role, is_verified, is_active FROM %s ORDER BY id LIMIT $1 OFFSET $2", userTable)
	rows, err := r.conn.Query(ctx, query, quantity, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to select users from database: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, quantity)
	for rows.Next() {
		var model UserModel
		err := rows.Scan(
			&model.ID,
			&model.Email,
			&model.PasswordHash,
			&model.Name,
			&model.Surname,
			&model.Phone,
			&model.Role,
			&model.IsVerified,
			&model.IsActive,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user model: %w", err)
		}

		users = append(users, *model.ToDomain())
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during row iteration: %w", err)
	}

	return users, totalCount, nil
}
