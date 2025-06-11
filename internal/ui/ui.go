// Package main provides different user interface implementations for ezbp.
// It includes a fuzzy finder UI and a terminal-based UI.
package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/driquet/ezbp/internal/boilerplate"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

// TODO: WTF is this
var execCommand = exec.Command // Public for tests to modify

// ErrUserAborted is returned when the user cancels an input/selection operation.
// Mimicking huh.ErrUserAborted for consistency in error handling if needed.
var ErrUserAborted = huh.ErrUserAborted

// UI defines the interface for user interactions.
// It abstracts the methods for selecting boilerplates, choosing from a list, and prompting for input.
type UI interface {
	// SelectBoilerplate asks the user to choose a boilerplate from a map of available boilerplates.
	// It returns the name of the selected boilerplate or an error if the selection fails.
	SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error)

	// Select asks the user to choose among a list of possible string choices.
	// It takes a prompt message and a slice of choices.
	// It returns the selected choice or an error if the selection fails.
	Select(prompt string, choices []string) (string, error)

	// Prompt expects an answer from the user for a given prompt message.
	// It returns the user's input as a string or an error if reading input fails.
	Prompt(prompt string) (string, error)
}

// FuzzyConfig holds the configuration for the Fuzzy UI.
// Currently, it's an empty struct, but it can be extended with configuration options in the future.
type FuzzyConfig struct{}

// Fuzzy implements the UI interface using a fuzzy finder for selections.
type Fuzzy struct {
	// config holds the configuration for the Fuzzy UI.
	config FuzzyConfig
}

// NewFuzzy creates a new Fuzzy UI instance with the given configuration.
// It returns a UI interface or an error if initialization fails.
func NewFuzzy(config FuzzyConfig) (UI, error) {
	return &Fuzzy{
		config: config,
	}, nil
}

// SelectBoilerplate implements the UI interface method for selecting a boilerplate using a fuzzy finder.
// It sorts the boilerplates by usage count in descending order before presenting them to the user.
// It displays the usage count and name of each boilerplate in the fuzzy finder.
// A preview window shows the value of the currently selected boilerplate.
func (u *Fuzzy) SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error) {
	// Convert the map of boilerplates to a slice for sorting and fuzzy finding.
	// TODO: Could be done using slices/maps utils?
	var bps []*boilerplate.Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}

	// Sort the boilerplates by usage count in descending order.
	// This ensures that frequently used boilerplates appear at the top of the list.
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	// Use the fuzzyfinder library to let the user select a boilerplate.
	idx, err := fuzzyfinder.Find(
		bps, // The slice of boilerplates to choose from.
		func(i int) string { // Function to display each boilerplate in the list.
			return fmt.Sprintf("%5d %s", bps[i].Count, bps[i].Name) // Format: "  123 boilerplate_name"
		},
		fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string { // Function to display a preview for the selected boilerplate.
			if i == -1 { // If no item is selected (e.g., during initial display or empty list).
				return ""
			}
			return bps[i].Value // Show the boilerplate's template value in the preview.
		}),
	)
	if err != nil {
		return "", fmt.Errorf("failed to find boilerplate: %w", err)
	}

	return bps[idx].Name, nil
}

// Select implements the UI interface method for selecting from a list of choices using a fuzzy finder.
// It takes a prompt (though not used in the current fuzzy finder implementation) and a slice of string choices.
func (u *Fuzzy) Select(prompt string, choices []string) (string, error) {
	// Use the fuzzyfinder library to let the user select a choice.
	idx, err := fuzzyfinder.Find(
		choices, // The slice of strings to choose from.
		func(i int) string { // Function to display each choice in the list.
			return choices[i]
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to select choice: %w", err)
	}
	return choices[idx], nil
}

// Prompt implements the UI interface method for prompting the user for input using standard input.
// It displays the prompt message and reads a line of text from the user.
func (u *Fuzzy) Prompt(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s> ", prompt)            // Display the prompt message.
	input, err := reader.ReadString('\n') // Read input until a newline character.
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return input, nil
}

// TermUI implements the UI interface using the charmbracelet/huh library for terminal-based interactions.
type TermUI struct{}

// NewTermUI creates a new TermUI instance.
// It returns a UI interface.
func NewTermUI() UI {
	return &TermUI{}
}

// SelectBoilerplate implements the UI interface method for selecting a boilerplate using a terminal select prompt.
// It sorts the boilerplates by usage count in descending order.
// It uses huh.NewSelect to present the options to the user.
func (u *TermUI) SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error) {
	// Convert the map of boilerplates to a slice for sorting and display.
	// TODO: Could be done using slices/maps utils?
	var bps []*boilerplate.Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}

	// Sort the boilerplates by usage count in descending order.
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	// Create huh.Option items for each boilerplate.
	// The label shows the count and name, while the value is the boilerplate name.
	var opts []huh.Option[string]
	for _, bp := range bps {
		opts = append(opts, huh.NewOption[string](fmt.Sprintf("%4d %s", bp.Count, bp.Name), bp.Name))
	}

	var name string
	form := huh.NewForm(
		huh.NewGroup( // Group UI elements.
			huh.NewSelect[string](). // Create a select prompt.
							Title("Boilerplate to expland").
							Options(opts...).
							Value(&name),
		),
	)

	// Run the form to get user input.
	err := form.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run select form: %w", err)
	}

	return name, nil
}

// Select implements the UI interface method for selecting from a list of choices using a terminal select prompt.
// It uses huh.NewSelect to present the options to the user.
func (u *TermUI) Select(prompt string, choices []string) (string, error) {
	var value string

	err := huh.NewSelect[string]().
		Title(prompt).
		Options(huh.NewOptions[string](choices...)...).
		Value(&value).
		Run()
	if err != nil {
		return "", fmt.Errorf("failed to run select prompt: %w", err)
	}

	return value, nil
}

// Prompt implements the UI interface method for prompting the user for input using a terminal input field.
// It uses huh.NewInput to get input from the user.
func (u *TermUI) Prompt(prompt string) (string, error) {
	var value string

	// Create and run a new input prompt using the huh library.
	err := huh.NewInput().
		Title(prompt). // Set the title of the input field.
		Value(&value). // Store the user's input in the 'value' variable.
		Run()
	if err != nil {
		return "", fmt.Errorf("failed to run input prompt: %w", err)
	}

	return value, nil
}

// RofiConfig holds configuration specific to the Rofi user interface.
type RofiConfig struct {
	// Path is the command or path to the Rofi executable.
	Path string `toml:"path"`
	// Theme specifies the Rofi theme to use. If empty, Rofi's default theme is used.
	Theme string `toml:"theme,omitempty"`
	// SelectArgs are extra arguments to pass to Rofi when used for selections (e.g., boilerplate choice, multiple choice prompts).
	SelectArgs []string `toml:"select_args,omitempty"`
	// InputArgs are extra arguments to pass to Rofi when used for free-form text input.
	InputArgs []string `toml:"input_args,omitempty"`
}

// RofiUI implements the UI interface using Rofi for user interactions.
type RofiUI struct {
	config RofiConfig // Holds Rofi-specific configuration from main Config
}

// NewRofiUI creates a new RofiUI instance with the given Rofi configuration.
func NewRofiUI(config RofiConfig) UI {
	return &RofiUI{config: config}
}

// runRofi executes a Rofi command with the given arguments and input string.
// It returns the selected string or an error.
func (u *RofiUI) runRofi(prompt string, input string, args []string) (string, error) {
	cmdArgs := []string{"-dmenu"}
	if prompt != "" {
		cmdArgs = append(cmdArgs, "-p", prompt)
	}

	if u.config.Theme != "" {
		cmdArgs = append(cmdArgs, "-theme", u.config.Theme)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(u.config.Path, cmdArgs...)

	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	} else {
		// For Rofi's input mode when no stdin is provided for selection.
		// Some versions might need specific flags like -input /dev/null or -l 0.
		// However, many Rofi versions automatically switch to input mode if stdin is not a TTY and no lines are provided.
		// If issues arise, flags like "-l", "0" or "-input", "/dev/null" might be needed here.
		// For now, we rely on Rofi's behavior of switching to input mode when Stdin is not set or is not a TTY.
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error is due to user cancellation (e.g., pressing Esc).
		// Rofi typically exits with status 1 on Esc. Other non-zero exits might be actual errors.
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 is often user cancellation.
			// Exit code 10, 11, 12, 13 are for custom keybindings in Rofi.
			// We'll treat exit code 1 as cancellation.
			if exitErr.ExitCode() == 1 {
				return "", ErrUserAborted
			}
		}
		return "", fmt.Errorf("rofi command failed: %w\nStderr: %s", err, stderr.String())
	}

	selected := strings.TrimSpace(stdout.String())
	// If Rofi was cancelled in a way that results in a 0 exit code but empty output (less common),
	// also treat as cancellation.
	if selected == "" && input != "" { // only consider empty output as cancellation if there was input to select from
		return "", ErrUserAborted
	}

	return selected, nil
}

// SelectBoilerplate implements the UI interface method for selecting a boilerplate using Rofi.
func (u *RofiUI) SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error) {
	var bps []*boilerplate.Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	var rofiInput strings.Builder
	nameOnlyMap := make(map[string]string) // To map display string back to original name

	for _, bp := range bps {
		// Format: "  123 boilerplate_name" - Rofi will display this.
		// We need to extract the actual name after selection.
		displayString := fmt.Sprintf("%5d %s", bp.Count, bp.Name)
		rofiInput.WriteString(displayString + "\n")
		nameOnlyMap[displayString] = bp.Name
	}

	selectedDisplayString, err := u.runRofi("Select Boilerplate:", rofiInput.String(), u.config.SelectArgs)
	if err != nil {
		return "", err
	}

	// Map the selected display string (which includes count) back to the actual boilerplate name.
	actualName, found := nameOnlyMap[selectedDisplayString]
	if !found {
		// This case should ideally not happen if Rofi returns a string that was in the input.
		// Could occur if Rofi somehow altered the string or if selection was empty and not caught as ErrUserAborted.
		if selectedDisplayString == "" { // If runRofi didn't return ErrUserAborted for empty selection
			return "", ErrUserAborted
		}
		return "", fmt.Errorf("selected boilerplate display string %q not found in original list", selectedDisplayString)
	}

	return actualName, nil
}

// Select implements the UI interface method for selecting from a list of choices using Rofi.
func (u *RofiUI) Select(prompt string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices provided for selection")
	}
	rofiInput := strings.Join(choices, "\n")
	return u.runRofi(prompt, rofiInput, u.config.SelectArgs)
}

// Prompt implements the UI interface method for prompting the user for input using Rofi.
func (u *RofiUI) Prompt(prompt string) (string, error) {
	// For text input, Rofi's dmenu typically expects no stdin, or specific flags.
	// We pass an empty input string and rely on runRofi's handling for input mode.
	// Additional args for input mode are taken from u.config.InputArgs.
	response, err := u.runRofi(prompt, "", u.config.InputArgs)
	if err != nil {
		return "", err
	}
	// If Rofi input is cancelled (e.g. Esc), runRofi should return ErrUserAborted.
	// If it returns empty string for other reasons (e.g. user just hits enter),
	// it's still a valid (empty) input.
	return response, nil
}

// TerminalUI implements the UI interface using Bubble Tea and Huh
type TerminalUI struct{}

// NewTerminalUI creates a new TerminalUI instance
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{}
}

// SelectBoilerplate displays boilerplates with preview using a custom Bubble Tea model
func (t *TerminalUI) SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error) {
	if len(boilerplates) == 0 {
		return "", fmt.Errorf("no boilerplates available")
	}

	// Convert map to sorted slice
	var sortedBoilerplates []*boilerplate.Boilerplate
	for _, bp := range boilerplates {
		sortedBoilerplates = append(sortedBoilerplates, bp)
	}

	// Sort by count (descending)
	sort.Slice(sortedBoilerplates, func(i, j int) bool {
		return sortedBoilerplates[i].Count > sortedBoilerplates[j].Count
	})

	// Create the selection model
	model := newBoilerplateSelector(sortedBoilerplates)

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run selection: %w", err)
	}

	result := finalModel.(*boilerplateSelectorModel)
	if result.cancelled {
		return "", fmt.Errorf("selection cancelled")
	}

	return result.selectedName, nil
}

// Select uses huh.Form for simple selection
func (t *TerminalUI) Select(prompt string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices available")
	}

	var selected string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(prompt).
				Options(huh.NewOptions(choices...)...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("selection failed: %w", err)
	}

	return selected, nil
}

// Prompt uses huh.Form for text input
func (t *TerminalUI) Prompt(prompt string) (string, error) {
	var input string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(prompt).
				Value(&input),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	return input, nil
}

// boilerplateSelectorModel is the Bubble Tea model for boilerplate selection with preview
type boilerplateSelectorModel struct {
	boilerplates  []*boilerplate.Boilerplate
	selectedIndex int
	selectedName  string
	cancelled     bool
	viewport      viewport.Model
	ready         bool
	width         int
	height        int
}

func newBoilerplateSelector(boilerplates []*boilerplate.Boilerplate) *boilerplateSelectorModel {
	vp := viewport.New(0, 0)

	return &boilerplateSelectorModel{
		boilerplates:  boilerplates,
		selectedIndex: 0,
		viewport:      vp,
	}
}

func (m *boilerplateSelectorModel) Init() tea.Cmd {
	return nil
}

func (m *boilerplateSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for the list (left side, 40% of width)
		listWidth := int(float64(msg.Width) * 0.4)
		previewWidth := msg.Width - listWidth - 3 // 3 for borders and spacing

		if previewWidth < 20 {
			previewWidth = 20
		}

		m.viewport.Width = previewWidth
		m.viewport.Height = msg.Height - 4 // Reserve space for title and instructions

		if !m.ready {
			m.ready = true
			m.updatePreview()
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if len(m.boilerplates) > 0 {
				m.selectedName = m.boilerplates[m.selectedIndex].Name
			}
			return m, tea.Quit

		case "up", "k":
			if m.selectedIndex > 0 {
				m.selectedIndex--
				m.updatePreview()
			}

		case "down", "j":
			if m.selectedIndex < len(m.boilerplates)-1 {
				m.selectedIndex++
				m.updatePreview()
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *boilerplateSelectorModel) updatePreview() {
	if len(m.boilerplates) == 0 {
		return
	}

	selected := m.boilerplates[m.selectedIndex]
	content := fmt.Sprintf("Name: %s\nUsage Count: %d\n\n%s",
		selected.Name, selected.Count, selected.Value)
	m.viewport.SetContent(content)
}

func (m *boilerplateSelectorModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Styles
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("57")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	// Build the list
	var listItems []string
	listItems = append(listItems, titleStyle.Render("Select Boilerplate:"))
	listItems = append(listItems, "")

	for i, bp := range m.boilerplates {
		line := fmt.Sprintf("  %s (used %d times)", bp.Name, bp.Count)
		if i == m.selectedIndex {
			line = selectedStyle.Render("▶ " + line)
		} else {
			line = normalStyle.Render("  " + line)
		}
		listItems = append(listItems, line)
	}

	// Calculate dimensions
	listWidth := int(float64(m.width) * 0.4)
	if listWidth < 30 {
		listWidth = 30
	}

	// Build the list view
	listView := strings.Join(listItems, "\n")
	if len(listView) > 0 {
		listView = lipgloss.NewStyle().
			Width(listWidth).
			Height(m.height - 4).
			Render(listView)
	}

	// Build preview
	previewTitle := titleStyle.Render("Preview:")
	previewContent := borderStyle.Render(m.viewport.View())
	preview := lipgloss.JoinVertical(lipgloss.Left, previewTitle, previewContent)

	// Join list and preview horizontally
	main := lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", preview)

	// Add instructions
	instructions := normalStyle.Render("↑/↓: navigate • enter: select • q/esc: quit")

	return lipgloss.JoinVertical(lipgloss.Left, main, "", instructions)
}
