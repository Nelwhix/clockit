package cmd

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	companyName   string
	companyRate   int64
	companyCurrency string
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a company",
	Long:  "Create a new company with a unique name.",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := companyName
		if name == "" && len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("company name is required (use --name or provide as first argument)")
		}

		dbPath, err := getPlatformSpecificDBPath()
		if err != nil {
			return fmt.Errorf("get db path: %w", err)
		}
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		var q string
		var params []any
		if companyRate != 0 || companyCurrency != "" {
			if companyCurrency == "" { companyCurrency = "USD" }
			q = `INSERT INTO companies(name, rate_cents, currency) VALUES (?,?,?)`
			params = []any{name, companyRate, companyCurrency}
		} else {
			q = `INSERT INTO companies(name) VALUES (?)`
			params = []any{name}
		}
		res, err := db.Exec(q, params...)
		if err != nil {
			// SQLite returns a constraint error on UNIQUE violation
			if errors.Is(err, sql.ErrNoRows) { // unlikely for Exec, but keep for clarity
				return fmt.Errorf("unexpected no rows on insert: %w", err)
			}
			return fmt.Errorf("create company: %w", err)
		}
		id, _ := res.LastInsertId()
		cmd.Printf("Created company %q with id %d\n", name, id)
		return nil
	},
}

func init() {
	companyCmd.AddCommand(createCmd)
	createCmd.Flags().StringVarP(&companyName, "name", "n", "", "Name of the company")
	createCmd.Flags().Int64Var(&companyRate, "rate-cents", 0, "Hourly rate in cents for this company")
	createCmd.Flags().StringVar(&companyCurrency, "currency", "", "Currency code (default USD)")
}
