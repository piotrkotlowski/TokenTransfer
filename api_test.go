
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors"`
}

/* Test setup */
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()


	if dsn := os.Getenv("TEST_DB_DSN"); dsn != "" {
		db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
		require.NoError(t, err, "connect db")
		return db, func() {}
	}


	ctx := context.Background()
	pgC, err := pgcontainer.Run(
		ctx,
		"postgres:16-alpine",
		pgcontainer.WithDatabase("tokenapi"),
		pgcontainer.WithUsername("user"),
		pgcontainer.WithPassword("pass"),
		pgcontainer.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start pg container")

	dsn, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "dsn")

	db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open gorm")

	cleanup := func() {
		assert.NoError(t, pgC.Terminate(ctx), "terminate container")
	}

	return db, cleanup
}


func seedWallets(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.AutoMigrate(&Wallet{}, &Transaction{}))

	wallets := []Wallet{
		{Address: "SENDER", Balance: 10},
		{Address: "R1", Balance: 0},
		{Address: "R2", Balance: 0},
		{Address: "R3", Balance: 0},
	}
	require.NoError(t, db.Create(&wallets).Error)
}

func postMutation(t *testing.T, srv *httptest.Server, mutation string) *graphQLResponse {
	t.Helper()

	body, _ := json.Marshal(graphQLRequest{Query: mutation})
	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var gqlResp graphQLResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&gqlResp))

	return &gqlResp
}

/* TransferSuccess */
func TestTransferSuccess(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	seedWallets(t, db)
	srv := httptest.NewServer(NewGraphQLHandler(db))
	defer srv.Close()

	mutation := `mutation { makeTransaction(sender:"SENDER", receiver:"R1", amount:5){ transaction_id } }`
	resp := postMutation(t, srv, mutation)

	require.Empty(t, resp.Errors, "unexpected GraphQL errors")

	var sender Wallet
	require.NoError(t, db.First(&sender, "address = ?", "SENDER").Error)
	assert.Equal(t, float64(5), sender.Balance)
}


/* InsufficientBalance */
func TestTransferInsufficientBalance(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	seedWallets(t, db)
	srv := httptest.NewServer(NewGraphQLHandler(db))
	defer srv.Close()

	mutation := `mutation { makeTransaction(sender:"SENDER", receiver:"R1", amount:50){ transaction_id } }`
	resp := postMutation(t, srv, mutation)

	require.NotEmpty(t, resp.Errors)
	assert.Equal(t, "insufficient amount", resp.Errors[0].Message)
}


/* Concurrency */
func TestConcurrentTransfers(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	seedWallets(t, db)
	srv := httptest.NewServer(NewGraphQLHandler(db))
	defer srv.Close()

	amounts := []float64{1, 4, 7} 

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(amounts))

	for _, amt := range amounts {
		amtCopy := amt
		go func() {
			defer wg.Done()
			<-start
			mut := fmt.Sprintf(
				`mutation { makeTransaction(sender:"SENDER", receiver:"R1", amount:%v){ transaction_id } }`, amtCopy,
			)
			_ = postMutation(t, srv, mut) 
		}()
	}

	time.Sleep(100 * time.Millisecond) 
	close(start)
	wg.Wait()

	var sender Wallet
	require.NoError(t, db.First(&sender, "address = ?", "SENDER").Error)
	assert.GreaterOrEqual(t, sender.Balance, float64(0))
	assert.LessOrEqual(t, sender.Balance, float64(10))
}


/* Sql injection */
func TestSQLInjectionAttempt(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	seedWallets(t, db)
	srv := httptest.NewServer(NewGraphQLHandler(db))
	defer srv.Close()

	inj := `SENDER'; DROP TABLE wallet; --`
	mutation := fmt.Sprintf(
		`mutation { makeTransaction(sender:%q, receiver:"R1", amount:1){ transaction_id } }`, inj,
	)

	resp := postMutation(t, srv, mutation)
	require.NotEmpty(t, resp.Errors, "expected GraphQL validation error")

	var cnt int64
	require.NoError(t, db.Model(&Wallet{}).Count(&cnt).Error)
	assert.Equal(t, int64(4), cnt, "wallet count should remain unchanged")
}
func TestCreateWalletDuplicate(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    
    require.NoError(t, db.AutoMigrate(&Wallet{}, &Transaction{}))

    srv := httptest.NewServer(NewGraphQLHandler(db))
    defer srv.Close()

    
    create := `mutation { createWallet(address:"DUP", balance:50) { address balance } }`
    resp1 := postMutation(t, srv, create)
    require.Empty(t, resp1.Errors, "first creation should succeed")

    
    resp2 := postMutation(t, srv, create)
    require.NotEmpty(t, resp2.Errors, "second creation should error")
    assert.Contains(t, resp2.Errors[0].Message, "already exists")

   
    var cnt int64
    require.NoError(t, db.Model(&Wallet{}).
        Where("address = ?", "DUP").
        Count(&cnt).Error)
    assert.Equal(t, int64(1), cnt)
}
/* MakeTransactionNegativeAmount */
func TestMakeTransactionNegativeAmount(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    seedWallets(t, db)
    srv := httptest.NewServer(NewGraphQLHandler(db))
    defer srv.Close()

    mutation := `mutation { makeTransaction(sender:"SENDER", receiver:"R1", amount:-5){ transaction_id } }`
    resp := postMutation(t, srv, mutation)

    require.NotEmpty(t, resp.Errors, "negative-amount mutation should fail")
    assert.Equal(t, "amount must be positive", resp.Errors[0].Message)

    var txCount int64
    require.NoError(t, db.Model(&Transaction{}).Count(&txCount).Error)
    assert.Equal(t, int64(0), txCount)

    var sender Wallet
    require.NoError(t, db.First(&sender, "address = ?", "SENDER").Error)
    assert.Equal(t, float64(10), sender.Balance)
}

/* MakeTransactionReceiverNotFound */
func TestMakeTransactionReceiverNotFound(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    seedWallets(t, db)
    srv := httptest.NewServer(NewGraphQLHandler(db))
    defer srv.Close()

    
    mutation := `mutation { makeTransaction(sender:"SENDER", receiver:"NOPE", amount:5){ transaction_id } }`
    resp := postMutation(t, srv, mutation)

    require.NotEmpty(t, resp.Errors, "missing-receiver mutation should fail")
    assert.Contains(t, resp.Errors[0].Message, "not in database")

    var txCount int64
    require.NoError(t, db.Model(&Transaction{}).Count(&txCount).Error)
    assert.Equal(t, int64(0), txCount)

    var sender Wallet
    require.NoError(t, db.First(&sender, "address = ?", "SENDER").Error)
    assert.Equal(t, float64(10), sender.Balance)
}
