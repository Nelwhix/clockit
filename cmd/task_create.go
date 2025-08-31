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
	taskName        string
	taskCompanyID   int64
	taskCompanyName string
	taskDesc        string
	taskActive      bool
)

// taskCreateCmd represents the task create command
var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a task",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.TrimSpace(taskName)
		if name == "" && len(args) > 0 {
			name = strings.TrimSpace(args[0])
		}
		if name == "" {
			return fmt.Errorf("task name is required (use --name or provide as first argument)")
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

		var companyID *int64
		if taskCompanyID != 0 || taskCompanyName != "" {
			var id int64
			if taskCompanyID != 0 {
				id = taskCompanyID
				// verify company exists
				row := db.QueryRow(`SELECT id FROM companies WHERE id = ?`, id)
				if err := row.Scan(&id); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return fmt.Errorf("no company found with id %d", taskCompanyID)
					}
					return fmt.Errorf("lookup company: %w", err)
				}
			} else {
				row := db.QueryRow(`SELECT id FROM companies WHERE name = ?`, taskCompanyName)
				if err := row.Scan(&id); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return fmt.Errorf("no company found with name %q", taskCompanyName)
					}
					return fmt.Errorf("lookup company by name: %w", err)
				}
			}
			companyID = &id
		}

		// Build INSERT with optional company_id
		if companyID != nil {
			res, err := db.Exec(`INSERT INTO tasks(company_id, name, description, is_active) VALUES (?,?,?,?)`, *companyID, name, nullIfEmpty(taskDesc), boolToInt(taskActive || true))
			if err != nil {
				return fmt.Errorf("create task: %w", err)
			}
			id, _ := res.LastInsertId()
			cmd.Printf("Created task %q with id %d\n", name, id)
			return nil
		}

		res, err := db.Exec(`INSERT INTO tasks(company_id, name, description, is_active) VALUES (NULL,?,?,?)`, name, nullIfEmpty(taskDesc), boolToInt(taskActive || true))
		if err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		id, _ := res.LastInsertId()
		cmd.Printf("Created task %q with id %d\n", name, id)
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskCreateCmd)
	taskCreateCmd.Flags().StringVarP(&taskName, "name", "n", "", "Name of the task")
	taskCreateCmd.Flags().StringVarP(&taskCompanyName, "company", "c", "", "Name of the company to associate")
	taskCreateCmd.Flags().StringVarP(&taskDesc, "description", "d", "", "Description of the task")
	taskCreateCmd.Flags().BoolVar(&taskActive, "active", true, "Whether the task is active")
}

// helpers
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
