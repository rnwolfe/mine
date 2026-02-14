package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var statusJSON bool
var statusPrompt bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mine status (for prompt integration)",
	Long:  `Output current mine status as JSON or a compact prompt segment.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	statusCmd.Flags().BoolVar(&statusPrompt, "prompt", false, "Output compact prompt segment")
}

// StatusData holds the status snapshot.
type StatusData struct {
	OpenTodos    int    `json:"open_todos"`
	TotalTodos   int    `json:"total_todos"`
	OverdueTodos int    `json:"overdue_todos"`
	DigStreak    int    `json:"dig_streak"`
	DigTotalMins int    `json:"dig_total_mins"`
	Version      string `json:"version"`
}

func runStatus(_ *cobra.Command, _ []string) error {
	data := gatherStatus()

	if statusPrompt {
		seg := formatPromptSegment(data)
		if seg != "" {
			fmt.Print(seg)
		}
		return nil
	}

	if statusJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(data)
	}

	// Default: human-readable
	fmt.Printf("Todos: %d open", data.OpenTodos)
	if data.OverdueTodos > 0 {
		fmt.Printf(" (%d overdue)", data.OverdueTodos)
	}
	fmt.Println()
	if data.DigStreak > 0 {
		fmt.Printf("Dig streak: %d days\n", data.DigStreak)
	}
	return nil
}

func gatherStatus() StatusData {
	data := StatusData{
		Version: version.Short(),
	}

	db, err := store.Open()
	if err != nil {
		return data
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	open, total, overdue, err := ts.Count()
	if err == nil {
		data.OpenTodos = open
		data.TotalTodos = total
		data.OverdueTodos = overdue
	}

	var current int
	if err := db.Conn().QueryRow(`SELECT current FROM streaks WHERE name = 'dig'`).Scan(&current); err == nil {
		data.DigStreak = current
	}

	var totalMins int
	if err := db.Conn().QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&totalMins); err == nil {
		data.DigTotalMins = totalMins
	}

	return data
}

func formatPromptSegment(data StatusData) string {
	seg := ""
	if data.OpenTodos > 0 {
		seg += fmt.Sprintf("%dt", data.OpenTodos)
	}
	if data.DigStreak > 0 {
		if seg != "" {
			seg += "|"
		}
		seg += fmt.Sprintf("%dd", data.DigStreak)
	}
	if seg != "" {
		return "[" + seg + "]"
	}
	return ""
}
