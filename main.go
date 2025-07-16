package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type Secret struct {
	Name         string
	Description  string
	LastAccessed *time.Time
	CreatedDate  *time.Time
	Selected     bool
}

type model struct {
	secrets   []Secret
	table     table.Model
	loading   bool
	err       error
	confirmed bool
	deleting  bool
	deleted   []string
	client    *secretsmanager.Client
}

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("211")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
)

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Delete  key.Binding
	Confirm key.Binding
	Quit    key.Binding
	Help    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Delete, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select},
		{k.Delete, k.Confirm, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Select: key.NewBinding(
		key.WithKeys(" ", "enter"),
		key.WithHelp("space", "toggle selection"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete selected"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "confirm deletion"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	client := secretsmanager.NewFromConfig(cfg)

	m := model{
		client:  client,
		loading: true,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

func (m model) Init() tea.Cmd {
	return loadSecrets(m.client)
}

func loadSecrets(client *secretsmanager.Client) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()

		input := &secretsmanager.ListSecretsInput{
			MaxResults: aws.Int32(100),
		}

		var secrets []Secret

		for {
			result, err := client.ListSecrets(ctx, input)
			if err != nil {
				return errMsg{err}
			}

			for _, secret := range result.SecretList {
				s := Secret{
					Name:         aws.ToString(secret.Name),
					Description:  aws.ToString(secret.Description),
					LastAccessed: secret.LastAccessedDate,
					CreatedDate:  secret.CreatedDate,
				}
				secrets = append(secrets, s)
			}

			if result.NextToken == nil {
				break
			}
			input.NextToken = result.NextToken
		}

		// Sort by last accessed date (oldest first, then never accessed)
		sort.Slice(secrets, func(i, j int) bool {
			if secrets[i].LastAccessed == nil && secrets[j].LastAccessed == nil {
				return secrets[i].CreatedDate.Before(*secrets[j].CreatedDate)
			}
			if secrets[i].LastAccessed == nil {
				return false
			}
			if secrets[j].LastAccessed == nil {
				return true
			}
			return secrets[i].LastAccessed.Before(*secrets[j].LastAccessed)
		})

		return secretsLoadedMsg{secrets}
	})
}

type secretsLoadedMsg struct {
	secrets []Secret
}

type errMsg struct {
	err error
}

func (e errMsg) Error() string {
	return e.err.Error()
}

type deleteCompleteMsg struct {
	deleted []string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		if m.confirmed && !m.deleting {
			switch msg.String() {
			case "y":
				m.deleting = true
				return m, m.deleteSelected()
			case "n", "q", "ctrl+c":
				m.confirmed = false
				return m, nil
			}
		}

		if m.deleting {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Select):
			if len(m.secrets) > 0 {
				idx := m.table.Cursor()
				m.secrets[idx].Selected = !m.secrets[idx].Selected
				m.updateTable()
			}

		case key.Matches(msg, keys.Delete):
			selected := m.getSelectedSecrets()
			if len(selected) > 0 {
				m.confirmed = true
			}

		case key.Matches(msg, keys.Up):
			m.table.MoveUp(1)

		case key.Matches(msg, keys.Down):
			m.table.MoveDown(1)
		}

	case secretsLoadedMsg:
		m.secrets = msg.secrets
		m.loading = false
		m.setupTable()

	case deleteCompleteMsg:
		m.deleted = msg.deleted
		m.deleting = false
		m.confirmed = false

	case errMsg:
		m.err = msg.err
		m.loading = false
	}

	return m, cmd
}

func (m *model) setupTable() {
	columns := []table.Column{
		{Title: "Selected", Width: 8},
		{Title: "Name", Width: 40},
		{Title: "Description", Width: 50},
		{Title: "Last Accessed", Width: 20},
		{Title: "Created", Width: 20},
	}

	m.table = table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	m.table.SetStyles(s)

	m.updateTable()
}

func (m *model) updateTable() {
	rows := make([]table.Row, len(m.secrets))
	for i, secret := range m.secrets {
		selected := "[ ]"
		if secret.Selected {
			selected = "[×]"
		}

		lastAccessed := "Never"
		if secret.LastAccessed != nil {
			lastAccessed = secret.LastAccessed.Format("2006-01-02")
		}

		created := ""
		if secret.CreatedDate != nil {
			created = secret.CreatedDate.Format("2006-01-02")
		}

		description := secret.Description
		if len(description) > 47 {
			description = description[:47] + "..."
		}

		rows[i] = table.Row{
			selected,
			secret.Name,
			description,
			lastAccessed,
			created,
		}
	}
	m.table.SetRows(rows)
}

func (m model) getSelectedSecrets() []Secret {
	var selected []Secret
	for _, secret := range m.secrets {
		if secret.Selected {
			selected = append(selected, secret)
		}
	}
	return selected
}

func (m model) deleteSelected() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		ctx := context.Background()
		var deleted []string

		for _, secret := range m.secrets {
			if secret.Selected {
				_, err := m.client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
					SecretId:                   aws.String(secret.Name),
					ForceDeleteWithoutRecovery: aws.Bool(true),
				})
				if err != nil {
					// Log error but continue with other deletions
					fmt.Printf("Error deleting %s: %v\n", secret.Name, err)
				} else {
					deleted = append(deleted, secret.Name)
				}
			}
		}

		return deleteCompleteMsg{deleted}
	})
}

func (m model) View() string {
	if m.loading {
		return "\n  Loading secrets from AWS...\n\n"
	}

	if m.err != nil {
		return fmt.Sprintf("\n  %s\n\n", errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	if len(m.deleted) > 0 {
		var b strings.Builder
		b.WriteString(successStyle.Render("✓ Successfully deleted secrets:"))
		b.WriteString("\n\n")
		for _, name := range m.deleted {
			b.WriteString(fmt.Sprintf("  • %s\n", name))
		}
		b.WriteString("\n" + "Press 'q' to quit")
		return b.String()
	}

	if len(m.secrets) == 0 {
		return "\n  No secrets found in AWS Secrets Manager.\n\n"
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("AWS Secrets Manager Cleanup Tool"))
	b.WriteString("\n\n")

	// Instructions
	if !m.confirmed {
		b.WriteString("Navigate with ↑/↓, select with space, delete with 'd', quit with 'q'\n")
		b.WriteString(fmt.Sprintf("Found %d secrets (sorted by last accessed, oldest first):\n\n", len(m.secrets)))
	}

	// Table
	if !m.confirmed {
		b.WriteString(baseStyle.Render(m.table.View()))
		b.WriteString("\n\n")

		selected := m.getSelectedSecrets()
		if len(selected) > 0 {
			b.WriteString(fmt.Sprintf("Selected %d secrets for deletion", len(selected)))
		}
	}

	// Confirmation dialog
	if m.confirmed {
		b.WriteString(errorStyle.Render("⚠ CONFIRM DELETION"))
		b.WriteString("\n\n")
		selected := m.getSelectedSecrets()
		b.WriteString(fmt.Sprintf("You are about to permanently delete %d secrets:\n\n", len(selected)))
		for _, secret := range selected {
			b.WriteString(fmt.Sprintf("  • %s\n", secret.Name))
		}
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("This action cannot be undone!"))
		b.WriteString("\n\n")
		b.WriteString("Press 'y' to confirm, 'n' to cancel")
	}

	// Deleting state
	if m.deleting {
		b.WriteString("Deleting selected secrets...\n")
	}

	return b.String()
}
