package cmd

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	deleteTaskID   int64
	deleteTaskName string
	deleteTaskYes  bool
	deleteTaskCompanyID int64
	deleteTaskCompanyName string
)

// taskDeleteCmd represents the task delete command
var taskDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a task by id or name",
	Long:  "Delete a task. Provide --id, or --name with a company context (--company-id or --company-name). Requires --yes to perform deletion.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteTaskID == 0 && deleteTaskName == "" {
			return fmt.Errorf("provide --id or --name (with company context)")
		}
		if deleteTaskID == 0 && deleteTaskName != "" && (deleteTaskCompanyID == 0 && deleteTaskCompanyName == "") {
			return fmt.Errorf("when deleting by --name, also provide --company-id or --company-name to disambiguate")
		}

		dbPath, err := getPlatformSpecificDBPath()
		if err != nil { return fmt.Errorf("get db path: %w", err) }
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil { return fmt.Errorf("open db: %w", err) }
		defer db.Close()

		var id int64
		var name string
		if deleteTaskID != 0 {
			if err := db.QueryRow(`SELECT id, name FROM tasks WHERE id = ?`, deleteTaskID).Scan(&id, &name); err != nil {
				if errors.Is(err, sql.ErrNoRows) { return fmt.Errorf("no task found with id %d", deleteTaskID) }
				return fmt.Errorf("lookup task by id: %w", err)
			}
		} else {
			var companyFilter string
			var arg any
			if deleteTaskCompanyID != 0 { companyFilter = "company_id = ?"; arg = deleteTaskCompanyID } else {
				var cid int64
				if err := db.QueryRow(`SELECT id FROM companies WHERE name = ?`, deleteTaskCompanyName).Scan(&cid); err != nil {
					if errors.Is(err, sql.ErrNoRows) { return fmt.Errorf("no company found with name %q", deleteTaskCompanyName) }
					return fmt.Errorf("lookup company: %w", err)
				}
				companyFilter = "company_id = ?"; arg = cid
			}
			q := "SELECT id, name FROM tasks WHERE name = ? AND " + companyFilter + " LIMIT 1"
			if err := db.QueryRow(q, deleteTaskName, arg).Scan(&id, &name); err != nil {
				if errors.Is(err, sql.ErrNoRows) { return fmt.Errorf("no task found with name %q for the specified company", deleteTaskName) }
				return fmt.Errorf("lookup task by name: %w", err)
			}
		}

		if !deleteTaskYes {
			cmd.Printf("About to delete task %q (id=%d). This will cascade delete related time entries. Proceed? (use --yes to skip)\n", name, id)
			return nil
		}

		res, err := db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
		if err != nil { return fmt.Errorf("delete task: %w", err) }
		affected, _ := res.RowsAffected()
		if affected == 0 { cmd.Println("No task deleted."); return nil }
		cmd.Printf("Deleted task %q (id=%d)\n", name, id)
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskDeleteCmd)
	taskDeleteCmd.Flags().Int64Var(&deleteTaskID, "id", 0, "ID of the task to delete")
	taskDeleteCmd.Flags().StringVar(&deleteTaskName, "name", "", "Name of the task to delete")
	taskDeleteCmd.Flags().BoolVarP(&deleteTaskYes, "yes", "y", false, "Confirm deletion without prompt")
	taskDeleteCmd.Flags().Int64Var(&deleteTaskCompanyID, "company-id", 0, "Company ID (required when deleting by name)")
	taskDeleteCmd.Flags().StringVar(&deleteTaskCompanyName, "company-name", "", "Company name (required when deleting by name)")
}
