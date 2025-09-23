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
	_, err := st.conn.ExecContext(ctx, `
        INSERT INTO shorteners
        (ID, URL, created_at)
        VALUES
        ($1, $2, $3);
    `, req.ID, req.URL, time.Now())

	return err
}

func (st *Store) Ping(ctx context.Context) error {
	return st.conn.PingContext(ctx)
}

func (st *Store) Close() error {
	return st.conn.Close()
}
