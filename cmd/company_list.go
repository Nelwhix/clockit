package cmd

import (
	"database/sql"
	"fmt"
	"math"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List companies",
	Long:  "List all companies in the database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := getPlatformSpecificDBPath()
		if err != nil {
			return fmt.Errorf("get db path: %w", err)
		}
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		rows, err := db.Query(`SELECT id, name, rate_cents, currency, created_at, updated_at FROM companies ORDER BY name`)
		if err != nil {
			return fmt.Errorf("query companies: %w", err)
		}
		defer rows.Close()

		type item struct {
			id        int64
			name      string
			rateCents sql.NullInt64
			currency  string
			createdAt string
			updatedAt string
		}
		var out []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.name, &it.rateCents, &it.currency, &it.createdAt, &it.updatedAt); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			out = append(out, it)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("rows: %w", err)
		}

		if len(out) == 0 {
			cmd.Println("No companies found.")
			return nil
		}

		for i, it := range out {
			cmd.Printf("ID: %d\n", it.id)
			cmd.Printf("Name: %s\n", it.name)
			cmd.Printf("Rate: %s\n", formatRateNullable(it.currency, it.rateCents))
			cmd.Printf("CreatedAt: %s\n", it.createdAt)
			cmd.Printf("UpdatedAt: %s\n", it.updatedAt)
			if i < len(out)-1 {
				cmd.Println("")
			}
		}
		return nil
	},
}

// formatRate turns cents + currency into "USD20/hr" or "USD20.50/hr"
func formatRate(currency string, cents int64) string {
	dollars := float64(cents) / 100.0
	if math.Abs(dollars-math.Trunc(dollars)) < 1e-9 {
		return fmt.Sprintf("%s%.0f/hr", currency, dollars)
	}
	return fmt.Sprintf("%s%.2f/hr", currency, dollars)
}

// formatRateNullable handles NULL rate_cents
func formatRateNullable(currency string, cents sql.NullInt64) string {
	if !cents.Valid {
		return "N/A"
	}
	return formatRate(currency, cents.Int64)
}

func init() {
	companyCmd.AddCommand(listCmd)
}
