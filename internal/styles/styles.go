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

	MainContentWrapperStyle = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Top).Padding(1)

	ScheduleListItem = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Border(lipgloss.HiddenBorder())
	ScheduleListCurr = lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center).Border(lipgloss.RoundedBorder())

	HelpTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Magenta).Italic(true).Padding(1).AlignHorizontal(lipgloss.Center)

	ScheduleStatusUpcoming  = lipgloss.NewStyle().Foreground(lipgloss.Blue).Bold(true)
	ScheduleStatusLive      = lipgloss.NewStyle().Foreground(lipgloss.Red).Bold(true)
	ScheduleStatusFinal     = lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(true)
	ScheduleStatusPostponed = lipgloss.NewStyle().Foreground(lipgloss.Yellow).Bold(true)

	ScheduleWinnerTeam  = lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true)
	ScheduleLoserTeam   = lipgloss.NewStyle()
	ScheduleNeutralTeam = lipgloss.NewStyle()
	ScheduleTeamRecord  = lipgloss.NewStyle().Foreground(lipgloss.Magenta)

	ScheduleTeamCell    = lipgloss.NewStyle().Align(lipgloss.Left, lipgloss.Center)
	ScheduleTableHeader = lipgloss.NewStyle().Foreground(lipgloss.Yellow).AlignHorizontal(lipgloss.Center).Bold(true)
	ScheduleTableStat   = lipgloss.NewStyle().Foreground(lipgloss.BrightWhite)

	LiveGameSectionWrapper  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
	LiveGamePlayDescription = lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Cyan).Italic(true)
)
