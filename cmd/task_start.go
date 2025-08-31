package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// taskStartCmd starts a stopwatch for a task and writes a time entry on Ctrl+C
var taskStartCmd = &cobra.Command{
	Use:   "start [task_id]",
	Short: "Start tracking time for a task (Ctrl+C to stop and save)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// parse task id
		taskID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || taskID <= 0 {
			return fmt.Errorf("invalid task id: %q", args[0])
		}

		// open db
		dbPath, err := getPlatformSpecificDBPath()
		if err != nil {
			return fmt.Errorf("get db path: %w", err)
		}
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		// ensure task exists and fetch optional company_id
		var (
			existsID int64
			companyID sql.NullInt64
		)
		row := db.QueryRow(`SELECT id, company_id FROM tasks WHERE id = ?`, taskID)
		if err := row.Scan(&existsID, &companyID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("task with id %d not found", taskID)
			}
			return fmt.Errorf("lookup task: %w", err)
		}

		// model for stopwatch and persistence
		m := swModel{
			stopwatch: stopwatch.NewWithInterval(time.Millisecond * 100),
			keymap: keymap{
				start: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start")),
				stop:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "stop")),
				reset: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset")),
				quit:  key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("q", "quit & save")),
			},
			help:      help.New(),
			taskID:    taskID,
			companyID: func() *int64 { if companyID.Valid { v := companyID.Int64; return &v }; return nil }(),
			db:        db,
		}
		m.keymap.start.SetEnabled(false)

		// trap SIGINT to let Bubble Tea handle Ctrl+C cleanly
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigc)

		p := tea.NewProgram(&m)
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("program run: %w", err)
		}

		return nil
	},
}

func init() {
	// Add as subcommand of task
	taskCmd.AddCommand(taskStartCmd)
}

// swModel implements a minimal stopwatch Bubble Tea model

type swModel struct {
	stopwatch stopwatch.Model
	keymap    keymap
	help      help.Model
	quitting  bool

	// persistence
	db        *sql.DB
	saved     bool
	taskID    int64
	companyID *int64
}

type keymap struct {
	start key.Binding
	stop  key.Binding
	reset key.Binding
	quit  key.Binding
}

func (m swModel) Init() tea.Cmd {
	return tea.Batch(m.stopwatch.Init(), m.stopwatch.Start())
}

func (m swModel) View() string {
	s := m.stopwatch.View() + "\n"
	if !m.quitting {
		s = "Elapsed: " + s
		s += m.helpView()
	}
	return s
}

func (m swModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.reset,
		m.keymap.quit,
	})
}

func (m *swModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			// persist entry on quit with current time
			_ = m.persistTimeEntry(time.Now())
			return m, tea.Quit
		case key.Matches(msg, m.keymap.reset):
			m.saved = false
			return m, m.stopwatch.Reset()
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			m.keymap.stop.SetEnabled(!m.stopwatch.Running())
			m.keymap.start.SetEnabled(m.stopwatch.Running())
			return m, m.stopwatch.Toggle()
		}
	}
	var cmd tea.Cmd
	m.stopwatch, cmd = m.stopwatch.Update(msg)
	return m, cmd
}

func (m *swModel) persistTimeEntry(end time.Time) error {
	if m.saved {
		return nil
	}
	m.saved = true
	// Derive start time from stopwatch elapsed to ensure accuracy across resets/pauses
	elapsed := m.stopwatch.Elapsed()
	start := end.Add(-elapsed)
	// Insert time entry (company_id optional)
	var (
		q  string
		ar []any
	)
	if m.companyID != nil {
		q = `INSERT INTO time_entries(task_id, company_id, start_time, end_time, notes) VALUES (?,?,?,?,?)`
		ar = []any{m.taskID, *m.companyID, start, end, nil}
	} else {
		q = `INSERT INTO time_entries(task_id, company_id, start_time, end_time, notes) VALUES (?,?,?, ?,?)`
		ar = []any{m.taskID, nil, start, end, nil}
	}
	if _, err := m.db.Exec(q, ar...); err != nil {
		return fmt.Errorf("insert time entry: %w", err)
	}
	fmt.Printf("\nSaved time entry for task %d: %s -> %s (%s)\n", m.taskID, start.Format(time.RFC3339), end.Format(time.RFC3339), elapsed.Truncate(time.Second))
	return nil
}
