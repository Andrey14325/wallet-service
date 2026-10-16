package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"wallet-service/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	transfers     = 400
	startBalance  = 100000
	deadlockCode  = "40P01"
	defaultDBConn = "postgres://wallet:wallet@localhost:5432/wallet?sslmode=disable"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDBConn
	}

	pool, err := storage.NewPostgresStorage(ctx, dsn)
	if err != nil {
		fmt.Println("db error:", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := storage.NewAccountStorage(pool)

	// Два свежих кошелька по 100000 копеек.
	a, _ := st.CreateNewAccount(ctx, "load-a")
	b, _ := st.CreateNewAccount(ctx, "load-b")
	_ = st.Deposit(ctx, a.ID, uuid.NewString(), startBalance)
	_ = st.Deposit(ctx, b.ID, uuid.NewString(), startBalance)

	var wg sync.WaitGroup
	var okCount, deadlocks, otherErrs atomic.Int64

	for i := 0; i < transfers; i++ {
		from, to := a.ID, b.ID
		if i%2 == 1 { // половина переводов идёт в обратную сторону
			from, to = b.ID, a.ID
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			err := st.Transfer(ctx, from, to, uuid.NewString(), 1)

			var pgErr *pgconn.PgError
			switch {
			case err == nil:
				okCount.Add(1)
			case errors.As(err, &pgErr) && pgErr.Code == deadlockCode:
				deadlocks.Add(1)
			default:
				otherErrs.Add(1)
				fmt.Println("error:", err)
			}
		}()
	}
	wg.Wait()

	accA, _ := st.GetAccount(ctx, a.ID)
	accB, _ := st.GetAccount(ctx, b.ID)

	fmt.Println("успешных переводов:", okCount.Load())
	fmt.Println("дедлоков (40P01):  ", deadlocks.Load())
	fmt.Println("других ошибок:     ", otherErrs.Load())
	fmt.Println("баланс A:", accA.Balance, " баланс B:", accB.Balance)
	fmt.Println("сумма:", accA.Balance+accB.Balance, " ожидалось:", 2*startBalance)
}
