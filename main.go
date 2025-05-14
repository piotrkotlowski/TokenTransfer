package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/99designs/gqlgen/graphql/playground"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)



func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.AutoMigrate(&Wallet{}, &Transaction{}); err != nil {
		log.Fatalf("auto‑migrate failed: %v", err)
	}

	
	http.Handle("/graphql", NewGraphQLHandler(db))

	
	http.Handle("/", playground.Handler(
		"Token API – GraphQL Playground", 
		"/graphql",                       
	))

	log.Println("🚀  Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/*database connection*/
func connectDB() (*gorm.DB, error) {
	host := getEnv("DB_HOST", "db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "hello")
	password := getEnv("DB_PASSWORD", "hello")
	dbname := getEnv("DB_NAME", "db")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, err2 := db.DB()
			if err2 == nil {
				err = sqlDB.Ping()
			} else {
				err = err2
			}
		}
		if err == nil {
			return db, nil
		}
		log.Printf("waiting for database: %v", err)
		time.Sleep(2 * time.Second)
	}
	return nil, err
}


/* convenience env helper */
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
