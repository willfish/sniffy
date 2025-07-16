package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tokyo Night theme colors
var (
	bgColor     = lipgloss.Color("#1a1b26")
	fgColor     = lipgloss.Color("#a9b1d6")
	blueColor   = lipgloss.Color("#7aa2f7")
	purpleColor = lipgloss.Color("#bb9af7")
	greenColor  = lipgloss.Color("#9ece6a")
	redColor    = lipgloss.Color("#f7768e")
	yellowColor = lipgloss.Color("#e0af68")
	dimColor    = lipgloss.Color("#565f89")

	titleStyle = lipgloss.NewStyle().
			Foreground(purpleColor).
			Bold(true).
			Padding(1, 2)

	bannerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(blueColor).
			Padding(1, 2).
			Align(lipgloss.Center)

	uiStyle = lipgloss.NewStyle().
		Foreground(blueColor).
		Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(greenColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(redColor).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	yellowStyle = lipgloss.NewStyle().
			Foreground(yellowColor)
)

// AWS integration
type AWSSecretsManager struct {
	client *secretsmanager.Client
}

func NewAWSSecretsManager() (*AWSSecretsManager, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &AWSSecretsManager{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

type SecretEntry struct {
	Name             string
	CreatedDate      *time.Time
	LastAccessedDate *time.Time
}

func (sm *AWSSecretsManager) ListSecrets(ctx context.Context) ([]SecretEntry, error) {
	var secrets []SecretEntry

	paginator := secretsmanager.NewListSecretsPaginator(sm.client, &secretsmanager.ListSecretsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range page.SecretList {
			if secret.Name != nil && !strings.HasSuffix(*secret.Name, "-configuration") {
				if secret.CreatedDate == nil {
					continue // Skip if no creation date
				}
				secrets = append(secrets, SecretEntry{
					Name:             *secret.Name,
					CreatedDate:      secret.CreatedDate,
					LastAccessedDate: secret.LastAccessedDate,
				})
			}
		}
	}

	return secrets, nil
}

func (sm *AWSSecretsManager) ListSecretVersions(ctx context.Context, secretName string) ([]types.SecretVersionsListEntry, error) {
	var versions []types.SecretVersionsListEntry

	input := &secretsmanager.ListSecretVersionIdsInput{
		SecretId:          aws.String(secretName),
		IncludeDeprecated: aws.Bool(true),
		MaxResults:        aws.Int32(100),
	}

	paginator := secretsmanager.NewListSecretVersionIdsPaginator(sm.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secret versions: %w", err)
		}

		versions = append(versions, page.Versions...)
	}

	return versions, nil
}

func (sm *AWSSecretsManager) GetSecretValue(ctx context.Context, secretName, versionId string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId:  aws.String(secretName),
		VersionId: aws.String(versionId),
	}

	output, err := sm.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret value: %w", err)
	}

	return *output.SecretString, nil
}

func (sm *AWSSecretsManager) DeleteSecret(ctx context.Context, secretName string) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(secretName),
	}

	_, err := sm.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", secretName, err)
	}

	return nil
}

// Enhanced secret analysis
type SecretAnalyzer struct {
	awsManager *AWSSecretsManager
}

func NewSecretAnalyzer() (*SecretAnalyzer, error) {
	awsManager, err := NewAWSSecretsManager()
	if err != nil {
		return nil, err
	}

	return &SecretAnalyzer{
		awsManager: awsManager,
	}, nil
}

const recentThresholdDays = 14

func (sa *SecretAnalyzer) AnalyzeSecrets(ctx context.Context, applyFilter bool) ([]SecretResult, error) {
	// Step 1: Get AWS secrets
	secrets, err := sa.awsManager.ListSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch AWS secrets: %w", err)
	}

	if len(secrets) == 0 {
		return []SecretResult{}, nil
	}

	var results []SecretResult

	// Step 2: Analyze each secret
	for _, entry := range secrets {
		// Calculate days since access
		var daysSinceAccess int = 9999
		var lastAccessedStr string = "Never"

		createdDays := int(time.Since(*entry.CreatedDate).Hours() / 24)

		if entry.LastAccessedDate != nil {
			lastAccessedStr = entry.LastAccessedDate.Format("2006-01-02")
			daysSinceAccess = int(time.Since(*entry.LastAccessedDate).Hours() / 24)
		} else {
			// Never accessed, use creation date as proxy
			daysSinceAccess = createdDays
		}

		if applyFilter && daysSinceAccess <= recentThresholdDays {
			continue
		}

		results = append(results, SecretResult{
			Name:         entry.Name,
			LastAccessed: lastAccessedStr,
		})
	}

	return results, nil
}

// Fuzzy match function
func isFuzzyMatch(query, target string) bool {
	query = strings.ToLower(query)
	target = strings.ToLower(target)
	i := 0
	for _, c := range query {
		found := false
		for ; i < len(target); i++ {
			if rune(target[i]) == c {
				found = true
				i++
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Model structures
type VersionInfo struct {
	VersionId    string
	CreatedDate  string
	LastAccessed string
	Stages       string
	Value        string
	Revealed     bool
}

type SecretResult struct {
	Name         string
	LastAccessed string
}

type model struct {
	state            string
	spinner          spinner.Model
	progress         progress.Model
	table            table.Model
	versionTable     table.Model
	filterInput      textinput.Model
	scanning         bool
	results          []SecretResult
	baseResults      []SecretResult
	originalResults  []SecretResult
	selected         []bool
	originalSelected []bool
	analyzer         *SecretAnalyzer
	currentScanStep  string
	err              error
	viewingSecret    string
	versions         []VersionInfo
	confirmDelete    bool
	deleteError      string
	copiedMessage    string
	lastCursorPos    int
	filterMode       string
	filtered         bool
	hasFilter        bool
}

type analysisCompleteMsg struct {
	results []SecretResult
	err     error
}

type versionsFetchedMsg struct {
	versions []VersionInfo
	err      error
}

type valueRevealedMsg struct {
	index int
	value string
	err   error
}

type startScanMsg struct{}

type deleteCompleteMsg struct {
	err error
}

type clearCopiedMsg struct{}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = uiStyle

	p := progress.New(progress.WithDefaultGradient())

	// Main table columns: checkbox, Secret, Last Accessed
	columns := []table.Column{
		{Title: "", Width: 3},
		{Title: "Secret", Width: 40},
		{Title: "Last Accessed", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(dimColor).
		BorderBottom(true).
		Bold(false)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(yellowColor).
		Background(bgColor).
		Bold(false)
	t.SetStyles(tableStyle)

	// Version table columns: Version ID, Created Date, Last Accessed, Stages, Value
	versionColumns := []table.Column{
		{Title: "Version ID", Width: 36},
		{Title: "Created Date", Width: 20},
		{Title: "Last Accessed", Width: 15},
		{Title: "Stages", Width: 20},
		{Title: "Value", Width: 40},
	}

	vt := table.New(
		table.WithColumns(versionColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	vt.SetStyles(tableStyle)

	fi := textinput.New()
	fi.Placeholder = "Filter..."

	// Initialize analyzer
	analyzer, err := NewSecretAnalyzer()
	if err != nil {
		analyzer = nil
	}

	return model{
		state:           "banner",
		spinner:         s,
		progress:        p,
		table:           t,
		versionTable:    vt,
		filterInput:     fi,
		scanning:        false,
		analyzer:        analyzer,
		currentScanStep: "Ready to scan",
		err:             err,
		selected:        []bool{},
		copiedMessage:   "",
		lastCursorPos:   0,
		filtered:        true,
		hasFilter:       false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return startScanMsg{}
	}))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()

		if key == "q" || key == "ctrl+c" {
			return m, tea.Quit
		}

		if m.state == "confirm_delete" {
			if key == "y" {
				m.confirmDelete = false
				return m, m.performDelete()
			} else if key == "n" || key == "esc" {
				m.confirmDelete = false
				m.state = "results"
				m.deleteError = ""
			}
			return m, nil
		}

		if m.state == "filter_include" || m.state == "filter_exclude" {
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)

			filter := m.filterInput.Value()

			// Preview filter
			var tempResults []SecretResult
			for _, res := range m.originalResults {
				match := isFuzzyMatch(filter, res.Name)
				if (m.state == "filter_include" && match) || (m.state == "filter_exclude" && !match) {
					tempResults = append(tempResults, res)
				}
			}
			m.results = tempResults
			m.selected = make([]bool, len(m.results))
			m.table.SetRows(m.formatResults())

			if key == "esc" {
				m.results = m.originalResults
				m.selected = m.originalSelected
				m.table.SetRows(m.formatResults())
				m.state = "results"
				return m, nil
			}
			if key == "enter" {
				m.hasFilter = true
				m.state = "results"
				return m, nil
			}
			return m, cmd
		}

		if m.state == "view_secret" {
			if key == "esc" {
				m.state = "results"
				m.viewingSecret = ""
				m.versions = nil
				m.table.SetCursor(m.lastCursorPos)
				return m, nil
			}
			if key == "r" {
				cursor := m.versionTable.Cursor()
				if cursor >= 0 && cursor < len(m.versions) && !m.versions[cursor].Revealed {
					return m, m.revealValue(cursor)
				}
			}
			if key == "y" {
				err := clipboard.WriteAll(m.viewingSecret)
				if err == nil {
					m.copiedMessage = "Copied secret name to clipboard"
					return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return clearCopiedMsg{}
					})
				}
			}
			var cmd tea.Cmd
			m.versionTable, cmd = m.versionTable.Update(msg)
			return m, cmd
		}

		if m.state == "results" {
			if key == "r" {
				m.filtered = true
				m.state = "banner"
				m.scanning = false
				return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
					return startScanMsg{}
				})
			}
			if key == "R" {
				m.filtered = false
				m.state = "banner"
				m.scanning = false
				return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
					return startScanMsg{}
				})
			}
			if key == "enter" {
				cursor := m.table.Cursor()
				if cursor >= 0 && cursor < len(m.results) {
					m.lastCursorPos = cursor
					m.viewingSecret = m.results[cursor].Name
					m.state = "view_secret"
					return m, m.fetchVersions()
				}
			}
			if key == " " {
				cursor := m.table.Cursor()
				if cursor >= 0 && cursor < len(m.selected) {
					m.selected[cursor] = !m.selected[cursor]
					rows := m.table.Rows() // get copy
					check := " "
					if m.selected[cursor] {
						check = "✔"
					}
					rows[cursor][0] = check
					m.table.SetRows(rows)
					m.table.SetCursor(cursor)
				}
			}
			if key == "D" { // Shift-D
				hasSelected := false
				for _, sel := range m.selected {
					if sel {
						hasSelected = true
						break
					}
				}
				if hasSelected {
					m.confirmDelete = true
					m.state = "confirm_delete"
				}
			}
			if key == "y" {
				cursor := m.table.Cursor()
				if cursor >= 0 && cursor < len(m.results) {
					err := clipboard.WriteAll(m.results[cursor].Name)
					if err == nil {
						m.copiedMessage = "Copied secret name to clipboard"
						return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
							return clearCopiedMsg{}
						})
					}
				}
			}
			if key == "/" {
				m.state = "filter_include"
				m.originalResults = append([]SecretResult(nil), m.results...)
				m.originalSelected = append([]bool(nil), m.selected...)
				m.filterInput.Reset()
				m.filterInput.Focus()
				return m, nil
			}
			if key == "?" {
				m.state = "filter_exclude"
				m.originalResults = append([]SecretResult(nil), m.results...)
				m.originalSelected = append([]bool(nil), m.selected...)
				m.filterInput.Reset()
				m.filterInput.Focus()
				return m, nil
			}
			if key == "esc" {
				if m.hasFilter {
					m.results = m.baseResults
					m.selected = make([]bool, len(m.results))
					m.table.SetRows(m.formatResults())
					m.hasFilter = false
				}
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}

	case startScanMsg:
		if m.analyzer == nil {
			m.state = "error"
			return m, nil
		}
		m.state = "scanning"
		m.scanning = true
		m.currentScanStep = "Connecting to AWS and analyzing secrets..."
		return m, tea.Batch(
			m.spinner.Tick,
			m.startRealScan(),
		)

	case analysisCompleteMsg:
		m.scanning = false
		m.state = "results"
		m.baseResults = msg.results
		m.results = msg.results
		m.err = msg.err
		if m.err == nil {
			m.selected = make([]bool, len(m.results))
			m.table.SetRows(m.formatResults())
			m.hasFilter = false
		}
		return m, nil

	case versionsFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = "results"
		} else {
			m.versions = msg.versions
			m.versionTable.SetRows(m.formatVersions())
		}
		return m, nil

	case valueRevealedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.versions[msg.index].Value = msg.value
			m.versions[msg.index].Revealed = true
			m.versionTable.SetRows(m.formatVersions())
		}
		return m, nil

	case deleteCompleteMsg:
		if msg.err != nil {
			m.deleteError = msg.err.Error()
		} else {
			// Remove deleted secrets from results
			var newResults []SecretResult
			var newSelected []bool
			for i, sel := range m.selected {
				if !sel {
					newResults = append(newResults, m.results[i])
					newSelected = append(newSelected, false)
				}
			}
			m.results = newResults
			m.baseResults = newResults // Update base if deleted
			m.selected = newSelected
			m.table.SetRows(m.formatResults())
		}
		m.state = "results"
		return m, nil

	case clearCopiedMsg:
		m.copiedMessage = ""
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m model) startRealScan() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results, err := m.analyzer.AnalyzeSecrets(ctx, m.filtered)
		return analysisCompleteMsg{results: results, err: err}
	}
}

func (m model) fetchVersions() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		versionsRaw, err := m.analyzer.awsManager.ListSecretVersions(ctx, m.viewingSecret)
		if err != nil {
			return versionsFetchedMsg{err: err}
		}

		var versions []VersionInfo
		for _, v := range versionsRaw {
			createdStr := ""
			if v.CreatedDate != nil {
				createdStr = v.CreatedDate.Format("2006-01-02 15:04")
			}
			lastAccessedStr := "Never"
			if v.LastAccessedDate != nil {
				lastAccessedStr = v.LastAccessedDate.Format("2006-01-02")
			}
			stagesStr := strings.Join(v.VersionStages, ", ")
			versions = append(versions, VersionInfo{
				VersionId:    *v.VersionId,
				CreatedDate:  createdStr,
				LastAccessed: lastAccessedStr,
				Stages:       stagesStr,
				Value:        "********",
				Revealed:     false,
			})
		}

		return versionsFetchedMsg{versions: versions}
	}
}

func (m model) revealValue(index int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		value, err := m.analyzer.awsManager.GetSecretValue(ctx, m.viewingSecret, m.versions[index].VersionId)
		if err != nil {
			return valueRevealedMsg{index: index, err: err}
		}
		return valueRevealedMsg{index: index, value: value}
	}
}

func (m model) performDelete() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var errStr strings.Builder
		for i, sel := range m.selected {
			if sel {
				err := m.analyzer.awsManager.DeleteSecret(ctx, m.results[i].Name)
				if err != nil {
					errStr.WriteString(fmt.Sprintf("%s: %v\n", m.results[i].Name, err))
				}
			}
		}
		if errStr.Len() > 0 {
			return deleteCompleteMsg{err: fmt.Errorf(errStr.String())}
		}
		return deleteCompleteMsg{}
	}
}

func (m model) View() string {
	var s strings.Builder

	switch m.state {
	case "banner":
		s.WriteString(m.renderBanner())
		s.WriteString("\n\n")
		if m.err != nil {
			s.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
			s.WriteString("\n\n")
			s.WriteString(dimStyle.Render("Make sure AWS credentials are configured"))
		} else {
			s.WriteString(uiStyle.Render("Starting scan..."))
		}

	case "scanning":
		s.WriteString(m.renderBanner())
		s.WriteString("\n\n")
		s.WriteString(m.renderScanningProgress())

	case "results":
		if m.err != nil {
			s.WriteString(errorStyle.Render(fmt.Sprintf("Scan failed: %v", m.err)))
			s.WriteString("\n\n")
			s.WriteString(dimStyle.Render("Check AWS credentials and permissions"))
		} else {
			s.WriteString(m.renderResults())
		}

	case "filter_include", "filter_exclude":
		s.WriteString(m.renderResults())
		s.WriteString("\n\n")
		if m.state == "filter_include" {
			s.WriteString("Include secrets containing: " + m.filterInput.View())
		} else {
			s.WriteString("Exclude secrets containing: " + m.filterInput.View())
		}

	case "view_secret":
		s.WriteString(titleStyle.Render(fmt.Sprintf("Versions for %s", m.viewingSecret)))
		s.WriteString("\n")
		s.WriteString(m.versionTable.View())

	case "confirm_delete":
		s.WriteString(errorStyle.Render("Confirm delete selected secrets? (y/n)"))

	case "error":
		s.WriteString(errorStyle.Render("Failed to initialize AWS connection"))
		s.WriteString("\n\n")
		s.WriteString(dimStyle.Render("Make sure AWS credentials are configured"))
	}

	if m.deleteError != "" {
		s.WriteString("\n")
		s.WriteString(errorStyle.Render(m.deleteError))
	}

	if m.copiedMessage != "" {
		s.WriteString("\n")
		s.WriteString(successStyle.Render(m.copiedMessage))
	}

	s.WriteString("\n\n")
	switch m.state {
	case "results":
		tooltip := "Enter: View secret • Space: Select • y: Copy name • /: Filter in • ?: Filter out • Shift+D: Delete selected • r: Rescan • R: Rescan all • q: Quit"
		if m.hasFilter {
			tooltip += " • esc: Clear filter"
		}
		s.WriteString(dimStyle.Render(tooltip))
	case "view_secret":
		s.WriteString(dimStyle.Render("r: Reveal value • y: Copy name • esc: Back • q: Quit"))
	case "confirm_delete":
		s.WriteString(dimStyle.Render("y: Yes • n: No • q: Quit"))
	case "filter_include", "filter_exclude":
		s.WriteString(dimStyle.Render("enter: Apply • esc: Cancel • q: Quit"))
	default:
		s.WriteString(dimStyle.Render("q: Quit"))
	}

	return s.String()
}

func (m model) renderBanner() string {
	banner := `
    ╔═══════════════════════════════════════════════════════════════╗
    ║                                                               ║
    ║             SNIFFY SCAN - Secret Analysis Tool                ║
    ║                                                               ║
    ║              Analyzing potentially unused secrets...          ║
    ║                                                               ║
    ╚═══════════════════════════════════════════════════════════════╝
    `
	return bannerStyle.Render(banner)
}

func (m model) renderScanningProgress() string {
	var s strings.Builder

	s.WriteString(fmt.Sprintf("%s\n\n", m.currentScanStep))

	s.WriteString(m.spinner.View())
	s.WriteString("\n\n")
	s.WriteString(m.progress.View())

	return s.String()
}

func (m model) renderResults() string {
	var s strings.Builder

	secretCount := len(m.results)

	if m.filtered {
		if secretCount > 0 {
			s.WriteString(yellowStyle.Render(fmt.Sprintf("Found %d potentially unused secrets", secretCount)))
		} else {
			s.WriteString(successStyle.Render("No potentially unused secrets found"))
		}
	} else {
		if secretCount > 0 {
			s.WriteString(uiStyle.Render(fmt.Sprintf("Listed %d secrets", secretCount)))
		} else {
			s.WriteString(successStyle.Render("No secrets found"))
		}
	}

	s.WriteString("\n\n")
	s.WriteString(titleStyle.Render("Secret Analysis"))
	s.WriteString("\n")
	s.WriteString(m.table.View())

	if m.filtered && secretCount > 0 {
		s.WriteString("\n\n")
		s.WriteString(yellowStyle.Render("Recommendations:"))
		s.WriteString("\n")
		s.WriteString(dimStyle.Render("Review and consider deleting unused secrets to improve security."))
	} else if !m.filtered && secretCount > 0 {
		s.WriteString("\n\n")
		s.WriteString(successStyle.Render("All secrets listed."))
	} else if secretCount == 0 {
		s.WriteString("\n\n")
		s.WriteString(successStyle.Render("All secrets appear to be in use."))
	}

	s.WriteString("\n\n")
	s.WriteString(uiStyle.Render("Scan complete."))

	return s.String()
}

func (m model) formatResults() []table.Row {
	var rows []table.Row
	for i, result := range m.results {
		check := " "
		if i < len(m.selected) && m.selected[i] {
			check = "✔"
		}
		rows = append(rows, table.Row{
			check,
			result.Name,
			result.LastAccessed,
		})
	}
	return rows
}

func (m model) formatVersions() []table.Row {
	var rows []table.Row
	for _, v := range m.versions {
		rows = append(rows, table.Row{
			v.VersionId,
			v.CreatedDate,
			v.LastAccessed,
			v.Stages,
			v.Value,
		})
	}
	return rows
}

func main() {
	m := initialModel()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
	}
}
