package cmd

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	deleteID   int64
	deleteName string
	deleteYes  bool
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a company by id or name",
	Long:  "Delete a company. You must provide either --id or --name. Requires confirmation unless --yes is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteID == 0 && deleteName == "" {
			return fmt.Errorf("provide --id or --name")
		}
		if deleteID != 0 && deleteName != "" {
			return fmt.Errorf("provide only one of --id or --name, not both")
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

		var id int64
		var name string
		if deleteID != 0 {
			row := db.QueryRow(`SELECT id, name FROM companies WHERE id = ?`, deleteID)
			if err := row.Scan(&id, &name); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("no company found with id %d", deleteID)
				}
				return fmt.Errorf("lookup company by id: %w", err)
			}
		} else {
			row := db.QueryRow(`SELECT id, name FROM companies WHERE name = ?`, deleteName)
			if err := row.Scan(&id, &name); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("no company found with name %q", deleteName)
				}
				return fmt.Errorf("lookup company by name: %w", err)
			}
		}

		if !deleteYes {
			cmd.Printf("About to delete company %q (id=%d). This will set company_id to NULL on related tasks/time_entries. Proceed? (use --yes to skip)\n", name, id)
			return nil
		}

		res, err := db.Exec(`DELETE FROM companies WHERE id = ?`, id)
		if err != nil {
			return fmt.Errorf("delete company: %w", err)
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			cmd.Println("No company deleted.")
			return nil
		}
		cmd.Printf("Deleted company %q (id=%d)\n", name, id)
		return nil
	},
}

func init() {
	companyCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().Int64Var(&deleteID, "id", 0, "ID of the company to delete")
	deleteCmd.Flags().StringVar(&deleteName, "name", "", "Name of the company to delete")
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Confirm deletion without prompt")
}
