package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	updateTaskID      int64
	updateName        string
	updateDesc        string
	updateActiveSet   bool
	updateActive      bool
	updateCompanyID   int64
	updateCompanyName string
)

// taskUpdateCmd represents the task update command
var taskUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a task by id",
	RunE: func(cmd *cobra.Command, args []string) error {
		if updateTaskID == 0 {
			return fmt.Errorf("--id is required")
		}
		dbPath, err := getPlatformSpecificDBPath()
		if err != nil { return fmt.Errorf("get db path: %w", err) }
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil { return fmt.Errorf("open db: %w", err) }
		defer db.Close()

		// ensure task exists
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM tasks WHERE id = ?`, updateTaskID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) { return fmt.Errorf("no task found with id %d", updateTaskID) }
			return fmt.Errorf("lookup task: %w", err)
		}

		sets := make([]string, 0)
		vals := make([]any, 0)

		if s := strings.TrimSpace(updateName); s != "" { sets = append(sets, "name = ?"); vals = append(vals, s) }
		if updateDesc != "" { sets = append(sets, "description = ?"); vals = append(vals, updateDesc) }
		if updateActiveSet { sets = append(sets, "is_active = ?"); if updateActive { vals = append(vals, 1) } else { vals = append(vals, 0) } }

		if updateCompanyID != 0 || strings.TrimSpace(updateCompanyName) != "" {
			var cid int64
			if updateCompanyID != 0 { cid = updateCompanyID } else {
				if err := db.QueryRow(`SELECT id FROM companies WHERE name = ?`, updateCompanyName).Scan(&cid); err != nil {
					if errors.Is(err, sql.ErrNoRows) { return fmt.Errorf("no company found with name %q", updateCompanyName) }
					return fmt.Errorf("lookup company: %w", err)
				}
			}
			sets = append(sets, "company_id = ?"); vals = append(vals, cid)
		}

		if len(sets) == 0 {
			cmd.Println("Nothing to update. Provide at least one field flag.")
			return nil
		}

		vals = append(vals, updateTaskID)
		q := "UPDATE tasks SET " + strings.Join(sets, ", ") + " WHERE id = ?"
		if _, err := db.Exec(q, vals...); err != nil { return fmt.Errorf("update task: %w", err) }
		cmd.Printf("Updated task id %d\n", updateTaskID)
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskUpdateCmd)
	taskUpdateCmd.Flags().Int64Var(&updateTaskID, "id", 0, "ID of the task to update")
	taskUpdateCmd.Flags().StringVar(&updateName, "name", "", "New name")
	taskUpdateCmd.Flags().StringVar(&updateDesc, "description", "", "New description (empty string allowed)")
	taskUpdateCmd.Flags().BoolVar(&updateActive, "active", true, "Set active to true/false (requires --set-active)")
	taskUpdateCmd.Flags().BoolVar(&updateActiveSet, "set-active", false, "Whether to update active flag")
	taskUpdateCmd.Flags().Int64Var(&updateCompanyID, "company-id", 0, "New company id")
	taskUpdateCmd.Flags().StringVar(&updateCompanyName, "company-name", "", "New company name")
}
