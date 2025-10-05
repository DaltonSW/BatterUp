package cmd

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"go.dalton.dog/batterup/internal/mlb"
	"go.dalton.dog/batterup/internal/ui"
)

var rootCmd = cobra.Command{
	Use:   "batterup",
	Short: "Monitor MLB games in your terminal",
	Run: func(cmd *cobra.Command, args []string) {
		client := mlb.NewClient()
		model := ui.NewAppModel(client)

		program := tea.NewProgram(model, tea.WithAltScreen())

		if _, err := program.Run(); err != nil {
			log.Fatalf("Error running BatterUp: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
