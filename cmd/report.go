package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Nelwhix/clockit/pkg"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
)

var (
	reportFrom     string
	reportTo       string
	reportTaskID   int64
	reportTaskName string
)

type reportFlags struct {
	From     string `validate:"required,date_yyyy_mm_dd"`
	To       string `validate:"required,date_yyyy_mm_dd"`
	TaskID   int64
	TaskName string
}

type reportRow struct {
	name  string
	hours float64
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generates a date range breakdown of tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		flags := reportFlags{
			From:     reportFrom,
			To:       reportTo,
			TaskID:   reportTaskID,
			TaskName: reportTaskName,
		}

		if err := validateReportFlags(flags); err != nil {
			return err
		}

		from, err := time.Parse("2006-01-02", flags.From)
		if err != nil {
			return fmt.Errorf("parse --from: %w", err)
		}

		to, err := time.Parse("2006-01-02", flags.To)
		if err != nil {
			return fmt.Errorf("parse --to: %w", err)
		}

		toExclusive := to.AddDate(0, 0, 1)
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

		query := strings.Builder{}
		query.WriteString(`
SELECT
  t.name,
  ROUND(COALESCE(SUM(te.duration_seconds), 0) / 3600.0, 2) AS hours
FROM time_entries te
JOIN tasks t ON t.id = te.task_id
WHERE te.end_time IS NOT NULL
  AND te.start_time >= ?
  AND te.start_time < ?
`)
		queryArgs := []any{
			from.Format("2006-01-02"),
			toExclusive.Format("2006-01-02"),
		}

		if flags.TaskID != 0 {
			query.WriteString(`
  AND te.task_id = ?
`)
			queryArgs = append(queryArgs, flags.TaskID)
		}

		if flags.TaskName != "" {
			query.WriteString(`AND t.name = ?`)
			queryArgs = append(queryArgs, flags.TaskName)
		}

		query.WriteString(`GROUP BY t.name ORDER BY t.name`)

		rows, err := db.Query(query.String(), queryArgs...)
		if err != nil {
			return fmt.Errorf("query report: %w", err)
		}
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {
				os.Exit(1)
			}
		}(rows)

		var reportRows []reportRow
		for rows.Next() {
			var row reportRow
			if err := rows.Scan(&row.name, &row.hours); err != nil {
				return fmt.Errorf("scan report row: %w", err)
			}
			reportRows = append(reportRows, row)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("read report rows: %w", err)
		}

		if len(reportRows) == 0 {
			cmd.Println("No time entries for the selected date range.")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TASK NAME\tTOTAL HOURS")
		fmt.Fprintln(w, "---------\t-----------")
		for _, row := range reportRows {
			fmt.Fprintf(w, "%s\t%.2f\n", row.name, row.hours)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().StringVar(&reportFrom, "from", "", "Start date YYYY-MM-DD (required)")
	reportCmd.Flags().StringVar(&reportTo, "to", "", "End date YYYY-MM-DD (required)")
	reportCmd.Flags().Int64Var(&reportTaskID, "task-id", 0, "Filter by task ID")
	reportCmd.Flags().StringVar(&reportTaskName, "task-name", "", "Filter by task name")
}

func validateReportFlags(flags reportFlags) error {
	validate := validator.New()

	if err := validate.RegisterValidation("date_yyyy_mm_dd", validateDateYYYYMMDD); err != nil {
		return fmt.Errorf("register date validator: %w", err)
	}

	if err := validate.Struct(flags); err != nil {
		return formatReportValidationError(err)
	}

	from, _ := time.Parse("2006-01-02", flags.From)
	to, _ := time.Parse("2006-01-02", flags.To)

	if from.After(to) {
		return fmt.Errorf("--from must be before or equal to --to")
	}

	if flags.TaskID != 0 && flags.TaskName != "" {
		return fmt.Errorf("use either --task-id or --task-name, not both")
	}

	return nil
}

func validateDateYYYYMMDD(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func formatReportValidationError(err error) error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	for _, fieldErr := range validationErrors {
		switch fieldErr.Field() {
		case "From":
			if fieldErr.Tag() == "required" {
				return fmt.Errorf("--from is required")
			}
			if fieldErr.Tag() == "date_yyyy_mm_dd" {
				return fmt.Errorf("invalid --from %q: expected YYYY-MM-DD", fieldErr.Value())
			}
		case "To":
			if fieldErr.Tag() == "required" {
				return fmt.Errorf("--to is required")
			}
			if fieldErr.Tag() == "date_yyyy_mm_dd" {
				return fmt.Errorf("invalid --to %q: expected YYYY-MM-DD", fieldErr.Value())
			}
		}
	}

	return err
}
