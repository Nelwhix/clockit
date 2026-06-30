package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Nelwhix/clockit/pkg"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	force bool
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "This initializes the data stores for clockit",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := pkg.GetPlatformSpecificDBPath()
		if err != nil {
			return fmt.Errorf("failed to get db path: %w", err)
		}

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("failed to open db: %w", err)
		}

		hasMigrations, err := hasMigrationsRun(db)
		if err != nil {
			return fmt.Errorf("failed to check migrations: %w", err)
		}

		// If migrations have already run, clear all data from user tables before re-initializing
		if hasMigrations {
			if !force {
				cmd.Print("Migrations have already been run, use --force to re-initialize (this will clear existing data)\n")
				return nil
			}

			// Single-shot reset: close and delete the database file, then reopen
			if err := db.Close(); err != nil {
				return fmt.Errorf("failed to close db before reset: %w", err)
			}
			if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove db file: %w", err)
			}
			// Reopen a fresh database file
			db, err = sql.Open("sqlite3", dbPath)
			if err != nil {
				return fmt.Errorf("failed to reopen db: %w", err)
			}
		}

		// Load migrations from embedded SQL to avoid filesystem dependency
		bytes := []byte(migrationsSQL)
		if len(bytes) == 0 {
			// Fallback to filesystem for dev scenarios
			var err error
			bytes, err = os.ReadFile("./migrations.sql")
			if err != nil {
				return fmt.Errorf("failed to read migrations: %w", err)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.ExecContext(ctx, string(bytes)); err != nil {
			return fmt.Errorf("failed to execute migrations: %w", err)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit migrations: %w", err)
		}

		cmd.Print("Successfully initialized data stores\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "re-initialize data stores")
}

func hasMigrationsRun(db *sql.DB) (bool, error) {
	var exists int
	err := db.QueryRow(`
		select 1 from sqlite_master where type = 'table' and name = 'schema_migrations'
	`).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil // table is not found -> no migrations ran
	}
	if err != nil {
		return false, fmt.Errorf("check schema_migrations existence: %w", err)
	}

	return true, nil
}
