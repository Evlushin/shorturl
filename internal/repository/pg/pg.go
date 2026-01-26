package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Evlushin/shorturl/internal/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/models"
	"github.com/Evlushin/shorturl/internal/myerrors"
	"github.com/Evlushin/shorturl/internal/repository"
	"github.com/Evlushin/shorturl/internal/repository/pg/migrator"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// generate:reset
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

	migrationsPath := "./migrations"

	_, err = os.Stat(migrationsPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Log.Info("directory with migrations does not exist")
			return store, nil
		}
		return nil, fmt.Errorf("error accessing migrations directory: %w", err)
	}

	err = migrator.ApplyMigrations(conn, fmt.Sprintf("file://%s", migrationsPath))
	if err != nil {
		return nil, err
	}

	return store, nil
}

func newErrGetShortenerNotFound(id string) error {
	return fmt.Errorf("%w for id = %s", myerrors.ErrGetShortenerNotFound, id)
}

func generator(doneCh chan struct{}, input []models.RequestIDBatch) chan models.RequestIDBatch {
	inputCh := make(chan models.RequestIDBatch)

	go func() {
		defer close(inputCh)

		for _, data := range input {
			select {
			case <-doneCh:
				return
			case inputCh <- data:
			}
		}
	}()

	return inputCh
}

func update(ctx context.Context, stmt *sql.Stmt, doneCh chan struct{}, inputCh chan models.RequestIDBatch, userID string) chan error {
	updateErr := make(chan error)

	go func() {
		defer close(updateErr)

		for id := range inputCh {
			_, err := stmt.ExecContext(ctx, id, userID)

			select {
			case <-doneCh:
				return
			case updateErr <- err:
			}
		}
	}()
	return updateErr
}

// fanOut принимает канал данных, порождает 10 горутин
func fanOut(ctx context.Context, stmt *sql.Stmt, doneCh chan struct{}, inputCh chan models.RequestIDBatch, userID string) []chan error {
	// количество горутин add
	numWorkers := 10
	// каналы, в которые отправляются результаты
	channelsErr := make([]chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		// получаем канал из горутины add
		err := update(ctx, stmt, doneCh, inputCh, userID)
		// отправляем его в слайс каналов
		channelsErr[i] = err
	}

	// возвращаем слайс каналов
	return channelsErr
}

// fanIn объединяет несколько каналов resultChs в один.
func fanIn(doneCh chan struct{}, resultChs ...chan error) chan error {
	// конечный выходной канал в который отправляем данные из всех каналов из слайса, назовём его результирующим
	finalCh := make(chan error)

	// понадобится для ожидания всех горутин
	var wg sync.WaitGroup

	// перебираем все входящие каналы
	for _, ch := range resultChs {
		// в горутину передавать переменную цикла нельзя, поэтому делаем так
		chClosure := ch

		// инкрементируем счётчик горутин, которые нужно подождать
		wg.Add(1)

		go func() {
			// откладываем сообщение о том, что горутина завершилась
			defer wg.Done()

			// получаем данные из канала
			for data := range chClosure {
				select {
				// выходим из горутины, если канал закрылся
				case <-doneCh:
					return
				// если не закрылся, отправляем данные в конечный выходной канал
				case finalCh <- data:
				}
			}
		}()
	}

	go func() {
		// ждём завершения всех горутин
		wg.Wait()
		// когда все горутины завершились, закрываем результирующий канал
		close(finalCh)
	}()

	// возвращаем результирующий канал
	return finalCh
}

func (st *Store) DeleteShortenerUrls(ctx context.Context, req []models.RequestIDBatch, userID string) error {
	tx, err := st.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, "UPDATE shorteners SET IS_DELETED = TRUE WHERE ID = $1 AND USER_ID = $2")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// сигнальный канал для завершения горутин
	doneCh := make(chan struct{})
	// закрываем его при завершении программы
	defer close(doneCh)

	// канал с данными
	inputCh := generator(doneCh, req)

	// получаем слайс каналов из 10 рабочих add
	channels := fanOut(ctx, stmt, doneCh, inputCh, userID)

	// а теперь объединяем десять каналов в один
	channelErr := fanIn(doneCh, channels...)

	for err = range channelErr {
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (st *Store) GetShortener(ctx context.Context, req *models.GetShortenerRequest) (*models.GetShortenerResponse, error) {
	var res models.GetShortenerResponse
	err := st.conn.QueryRowContext(ctx, `SELECT URL, IS_DELETED FROM shorteners WHERE ID = $1 LIMIT 1`, req.ID).Scan(&res.URL, &res.IsDeleted)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, newErrGetShortenerNotFound(req.ID)
		}
		return nil, err
	}

	return &res, nil
}

func (st *Store) GetStats(ctx context.Context) (*models.ResponseStats, error) {
	var res models.ResponseStats
	err := st.conn.QueryRowContext(ctx, `SELECT 
    COUNT(DISTINCT URL) AS urls,
    COUNT(DISTINCT USER_ID) AS users
	FROM shorteners
	WHERE IS_DELETED = FALSE;`).Scan(&res.URLs, &res.Users)

	if err != nil {
		return nil, fmt.Errorf("get stats request error: %w", err)
	}

	return &res, nil
}

func (st *Store) GetShortenerUrls(ctx context.Context, userID string) ([]models.GetShortenerUrls, error) {
	rows, err := st.conn.QueryContext(ctx, `SELECT ID, URL FROM shorteners WHERE USER_ID = $1 AND IS_DELETED = FALSE`, userID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, myerrors.ErrGetShortenerNotFound
		}
		return nil, err
	}

	var res []models.GetShortenerUrls
	for rows.Next() {
		var url models.GetShortenerUrls
		err = rows.Scan(&url.ID, &url.URL)
		if err != nil {
			return nil, err
		}

		res = append(res, url)
	}

	err = rows.Err()
	if err != nil {
		fmt.Println(2)
		return nil, err
	}
	fmt.Println(res)
	return res, nil
}

func (st *Store) SetShortener(ctx context.Context, req *models.SetShortenerRequest) error {
	var returnedID string
	_, err := st.conn.ExecContext(ctx, `
        INSERT INTO shorteners
        (ID, URL, CREATED_AT, USER_ID)
        VALUES
        ($1, $2, $3, $4);
    `, req.ID, req.URL, time.Now(), req.UserID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			err = st.conn.QueryRowContext(ctx, `
				SELECT ID FROM shorteners WHERE URL = $1 AND USER_ID = $2 LIMIT 1
			`, req.URL, req.UserID).Scan(&returnedID)

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
				(ID, URL, CREATED_AT, USER_ID)
				VALUES
				($1, $2, $3, $4)
				ON CONFLICT (URL, USER_ID) DO NOTHING
				RETURNING ID
			   `)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	var (
		returnedID   string
		errUniqueURL error
	)
	for key, r := range req {
		err = stmt.QueryRowContext(ctx, r.ID, r.URL, time.Now(), r.UserID).Scan(&returnedID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				err = st.conn.QueryRowContext(ctx, `
					SELECT ID FROM shorteners WHERE URL = $1 AND USER_ID = $2 LIMIT 1
				`, r.URL, r.UserID).Scan(&returnedID)

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
