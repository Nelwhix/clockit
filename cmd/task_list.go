package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/Nelwhix/clockit/pkg"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	listCompanyID   int64
	listCompanyName string
	listAll         bool
)

// taskListCmd represents the task list command
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := pkg.GetPlatformSpecificDBPath()
		if err != nil {
			return fmt.Errorf("get db path: %w", err)
		}
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer func(db *sql.DB) {
			err := db.Close()
			if err != nil {
				os.Exit(1)
			}
		}(db)

		var where []string
		var params []any
		if !listAll {
			where = append(where, "is_active = 1")
		}
		if listCompanyID != 0 {
			where = append(where, "company_id = ?")
			params = append(params, listCompanyID)
		} else if strings.TrimSpace(listCompanyName) != "" {
			where = append(where, "company_id = (SELECT id FROM companies WHERE name = ?)")
			params = append(params, listCompanyName)
		}
		q := "SELECT id, name, company_id, description, is_active, created_at, updated_at FROM tasks"
		if len(where) > 0 {
			q += " WHERE " + strings.Join(where, " AND ")
		}
		q += " ORDER BY name"

		rows, err := db.Query(q, params...)
		if err != nil {
			return fmt.Errorf("query tasks: %w", err)
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				os.Exit(1)
			}
		}(rows)

		type item struct {
			id          int64
			name        string
			companyID   sql.NullInt64
			description sql.NullString
			isActive    int
			createdAt   string
			updatedAt   string
		}
		var out []item
		for rows.Next() {
			var it item
			if err := rows.Scan(&it.id, &it.name, &it.companyID, &it.description, &it.isActive, &it.createdAt, &it.updatedAt); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
			out = append(out, it)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("rows: %w", err)
		}

		if len(out) == 0 {
			cmd.Println("No tasks found.")
			return nil
		}

		for i, it := range out {
			cmd.Printf("ID: %d\n", it.id)
			cmd.Printf("Name: %s\n", it.name)
			if it.companyID.Valid {
				cmd.Printf("CompanyID: %d\n", it.companyID.Int64)
			} else {
				cmd.Printf("CompanyID: \n")
			}
			if it.description.Valid {
				cmd.Printf("Description: %s\n", it.description.String)
			} else {
				cmd.Printf("Description: \n")
			}
			cmd.Printf("IsActive: %t\n", it.isActive == 1)
			cmd.Printf("CreatedAt: %s\n", it.createdAt)
			cmd.Printf("UpdatedAt: %s\n", it.updatedAt)
			if i < len(out)-1 {
				cmd.Println("")
			}
		}
		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskListCmd.Flags().Int64Var(&listCompanyID, "company-id", 0, "Filter by company ID")
	taskListCmd.Flags().StringVar(&listCompanyName, "company-name", "", "Filter by company name")
	taskListCmd.Flags().BoolVar(&listAll, "all", false, "Include inactive tasks")
}
