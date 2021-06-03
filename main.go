package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// addr is the bind address for the web server.
const addr = ":8080"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	// Parse command line flags.
	dsn := flag.String("dsn", "", "datasource name")
	flag.Parse()
	if *dsn == "" {
		flag.Usage()
		return fmt.Errorf("required: -dsn")
	}

	// Open database file.
	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create table for storing page views.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS page_views (id INTEGER PRIMARY KEY, timestamp TEXT);`); err != nil {
		return fmt.Errorf("cannot create table: %w", err)
	}

	// Run web server.
	fmt.Printf("listening on %s\n", addr)
	go http.ListenAndServe(addr,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Store page view.
			if _, err := db.Exec(`INSERT INTO page_views (timestamp) VALUES (?);`, time.Now().Format(time.RFC3339)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Read total page views.
			var n int
			if err := db.QueryRow(`SELECT COUNT(1) FROM page_views;`).Scan(&n); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Print total page views.
			fmt.Fprintf(w, "This server has been visited %d times.\n", n)
		}),
	)

	// Wait for signal.
	<-ctx.Done()
	log.Print("myapp received signal, shutting down")

	return nil
}
