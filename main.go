package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	title, desc, path string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list   list.Model
	choice string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			if i, ok := m.list.SelectedItem().(item); ok {
				m.choice = i.path
				return m, tea.Quit
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	entriesToCheck := []string{"dev", ".config"}
	allDirectories := getDirectories(entriesToCheck)

	var items []list.Item
	for category, entries := range allDirectories {
		for _, entry := range entries {
			if entry.IsDir() {
				home, err := os.UserHomeDir()
				if err != nil {
					log.Fatal(err)
				}
				fullPath := filepath.Join(home, category, entry.Name())

				items = append(items, item{
					title: entry.Name(),
					desc:  category,
					path:  fullPath,
				})
			}
		}
	}

	delegate := list.NewDefaultDelegate()

	// Normal State (Non-Selected)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.
		Foreground(lipgloss.Color("#d3c6aa")) // Foreground
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.
		Foreground(lipgloss.Color("#9da9a0")) // Dimmer

	// Selected State
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#a7c080")). // Green
		BorderLeftForeground(lipgloss.Color("#a7c080"))
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#7fbbb3")). // Blue
		BorderLeftForeground(lipgloss.Color("#a7c080"))

	// Dimmed State (When filtering but empty or no match)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.
		Foreground(lipgloss.Color("#7a8478")) // Greyish Green
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.
		Foreground(lipgloss.Color("#5c6a72")) // Greyish Blue

	// Filter Match State (Characters matching the search)
	delegate.Styles.FilterMatch = lipgloss.NewStyle().
		Underline(true).
		Foreground(lipgloss.Color("#e67e80")) // Red for matches

	l := list.New(items, delegate, 0, 0)
	l.Title = "Sessionizer"

	// Title Styling
	l.Styles.Title = l.Styles.Title.
		Background(lipgloss.Color("#a7c080")). // Green
		Foreground(lipgloss.Color("#272e33")). // Background Dark
		Bold(true)

	// Filter Input Styling
	l.FilterInput.PromptStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a7c080")) // Green Prompt

	l.FilterInput.TextStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d3c6aa")) // Normal Text

	// Cursor Styling
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a7c080")) // Green Cursor

	l.SetFilterText("")
	l.SetFilterState(list.Filtering)

	m := model{list: l}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok {
		if m.choice != "" {
			AttachOrCreateSession(m.choice)
		} else {
			os.Exit(10)
		}
	}
}

func AttachOrCreateSession(path string) {
	sessionName := filepath.Base(path)
	sessionName = strings.ReplaceAll(sessionName, ".", "_")
	sessionName = strings.ReplaceAll(sessionName, ":", "_")

	// Try to create the session if it doesn't exist
	// -d: detached
	// -s: session name
	// -c: start directory
	exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", path).Run()

	// Attach to the session
	// If we are already inside tmux, we should use switch-client
	var cmd *exec.Cmd
	if os.Getenv("TMUX") != "" {
		cmd = exec.Command("tmux", "switch-client", "-t", sessionName)
	} else {
		cmd = exec.Command("tmux", "attach-session", "-t", sessionName)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func getDirectories(entries []string) map[string][]os.DirEntry {
	allDirectories := make(map[string][]os.DirEntry)
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(home, entry)
		directories, err := os.ReadDir(fullPath)
		if err != nil {
			log.Fatal(err)
		}

		allDirectories[entry] = append(allDirectories[entry], directories...)
	}

	return allDirectories
}
