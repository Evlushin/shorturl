package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/repository/pg/migrator"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type Store struct {
	cfg  *config.Config
	conn *sql.DB
}

func NewStore(cfg *config.Config) (repository.Repository, error) {
	conn, err := sql.Open("pgx", cfg.DatabaseDsn)
	if err != nil {
		return nil, err
	}

	store := &Store{
		cfg:  cfg,
		conn: conn,
	}

	err = migrator.ApplyMigrations(conn, "file://./migrations")
	if err != nil {
		return nil, err
	}

	return store, nil
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func (st *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	var res models.GetShortenerResponse
	err := st.conn.QueryRowContext(ctx, `SELECT URL FROM shorteners WHERE ID = $1 LIMIT 1`, req.ID).Scan(&res.URL)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newErrGetShortenerNotFound(req.ID)
		}
		return nil, err
	}

	return &res, nil
}

func (st *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	var returnedID string
	_, err := st.conn.ExecContext(ctx, `
        INSERT INTO shorteners
        (ID, URL, created_at)
        VALUES
        ($1, $2, $3);
    `, req.ID, req.URL, time.Now())

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			err = st.conn.QueryRowContext(ctx, `
				SELECT ID FROM shorteners WHERE URL = $1
			`, req.URL).Scan(&returnedID)

			if err != nil {
				return err
			}

			err = myerrors.ErrConflictURL
			req.ID = returnedID
		} else {
			return err
		}
	}

	return err
}

func (st *Store) insertShortenerBatch(ctx context.Context, req []*models.SetShortenerBatchRequest) error {
	tx, err := st.conn.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO shorteners
				(ID, URL, created_at)
				VALUES
				($1, $2, $3)
				ON CONFLICT (URL) DO NOTHING
				RETURNING ID
			   `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var (
		returnedID   string
		errUniqueURL error
	)
	for key, r := range req {
		err = stmt.QueryRowContext(ctx, r.ID, r.URL, time.Now()).Scan(&returnedID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = st.conn.QueryRowContext(ctx, `
					SELECT ID FROM shorteners WHERE URL = $1
				`, r.URL).Scan(&returnedID)

				if err != nil {
					return err
				}
				req[key].ID = returnedID
				errUniqueURL = myerrors.ErrConflictURL
			} else {
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return errUniqueURL
}

func (st *Store) SetShortenerBatch(ctx context.Context, req []models.SetShortenerBatchRequest) error {
	const countBatch = 1000

	buf := make([]*models.SetShortenerBatchRequest, 0, countBatch)
	var errUniqueURL error
	for key := range req {
		buf = append(buf, &req[key])

		if len(buf) >= countBatch {
			err := st.insertShortenerBatch(ctx, buf)
			if err != nil {
				if errors.Is(err, myerrors.ErrConflictURL) {
					errUniqueURL = myerrors.ErrConflictURL
				} else {
					return err
				}
			}
			buf = buf[:0]
		}
	}
	err := st.insertShortenerBatch(ctx, buf)
	if err != nil {
		return err
	}

	return errUniqueURL
}

func (st *Store) Ping(ctx context.Context) error {
	return st.conn.PingContext(ctx)
}

func (st *Store) Close() error {
	return st.conn.Close()
}
