package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/manniwood/iidy/data"
	"github.com/manniwood/iidy/handlers"
	"github.com/manniwood/iidy/migrations"
)

const defaultPort string = "8080"

func main() {
	port := os.Getenv("IIDY_PORT")
	if port == "" {
		port = defaultPort
	}

	ctx, cFunc := context.WithCancel(context.Background())
	defer cFunc()

	poolURL := os.Getenv("IIDY_PG_POOL_URL")
	if poolURL == "" {
		poolURL = data.DefaultPgPoolURL
	}

	migrationURL := os.Getenv("IIDY_PG_MIGRATION_URL")
	if migrationURL == "" {
		migrationURL = data.DefaultPgMigrationURL
	}

	migrationConn, err := data.CreatePGXConnForMigration(ctx, migrationURL)
	if err != nil {
		log.Fatalf("Could not create connection for migration: %v\n", err)
	}
	defer migrationConn.Close(ctx)
	err = data.MigrateDB(ctx, migrationConn, migrations.Migrations, data.TernMigrationTable)
	if err != nil {
		log.Fatalf("Could not migrate db: %v\n", err)
	}
	// Don't need this single connection anymore, so close it.
	migrationConn.Close(ctx)

	data.PgxPool, err = data.CreatePGXPool(ctx, poolURL)
	defer data.PgxPool.Close()

	http.HandleFunc("POST /iidy/v1/lists/{list}/{item}", handlers.InsertOne)
	http.HandleFunc("GET /iidy/v1/lists/{list}/{item}", handlers.GetOne)
	http.HandleFunc("DELETE /iidy/v1/lists/{list}/{item}", handlers.DeleteOne)
	http.HandleFunc("POST /iidy/v1/increment/lists/{list}/{item}", handlers.IncrementOne)
	http.HandleFunc("POST /iidy/v1/batch/lists/{list}", handlers.InsertBatch)
	http.HandleFunc("DELETE /iidy/v1/batch/lists/{list}", handlers.DeleteBatch)
	http.HandleFunc("GET /iidy/v1/batch/lists/{list}", handlers.GetBatch)
	http.HandleFunc("POST /iidy/v1/increment/batch/lists/{list}", handlers.IncrementBatch)

	log.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
