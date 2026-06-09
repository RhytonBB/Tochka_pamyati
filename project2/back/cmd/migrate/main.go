package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = strings.TrimSpace(os.Args[1])
	}

	migrationsPath := filepath.ToSlash(filepath.Join(".", "migrations"))

	// Replace postgresql:// with pgx5:// for golang-migrate
	dbURL = strings.Replace(dbURL, "postgresql://", "pgx5://", 1)
	dbURL = strings.Replace(dbURL, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://"+migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}

	switch command {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "version":
		var v uint
		var dirty bool
		v, dirty, err = m.Version()
		if err == nil {
			fmt.Printf("%d dirty=%v\n", v, dirty)
			return
		}
	case "force":
		if len(os.Args) < 3 {
			log.Fatal("force requires version argument")
		}
		v, convErr := strconv.Atoi(os.Args[2])
		if convErr != nil {
			log.Fatalf("invalid version: %v", convErr)
		}
		err = m.Force(v)
	default:
		log.Fatalf("unknown command: %s (use up|down|version|force)", command)
	}

	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return
		}
		log.Fatalf("migrate %s: %v", command, err)
	}
}
