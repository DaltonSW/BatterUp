package styles

import (
	"github.com/charmbracelet/lipgloss/v2"
)

// Colors
var (
	BallColor   = lipgloss.Blue
	StrikeColor = lipgloss.Red
	OnBaseColor = lipgloss.Yellow
	OutColor    = lipgloss.BrightRed

	WalkColor        = lipgloss.Green
	StrikeOutColor   = lipgloss.Red
	InPlayOutColor   = lipgloss.Green
	InPlayNoOutColor = lipgloss.BrightGreen
	OtherEventColor  = lipgloss.White
)

// Styles
var (
	AppHeaderStyle = lipgloss.NewStyle().Background(lipgloss.White).Foreground(lipgloss.Black).Bold(true).Italic(true).AlignHorizontal(lipgloss.Center)

	ScheduleListItem = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Left).Border(lipgloss.HiddenBorder())
	ScheduleListCurr = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Left).Border(lipgloss.RoundedBorder())

	HelpTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Magenta).Italic(true).Padding(1).AlignHorizontal(lipgloss.Center)
)
