package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const fugoASCII = `
███████╗██╗   ██╗      ██████╗  ██████╗ 
██╔════╝██║   ██║     ██╔════╝ ██╔═══██╗
█████╗  ██║   ██║     ██║  ███╗██║   ██║
██╔══╝  ██║   ██║     ██║   ██║██║   ██║
██║     ╚██████╔╝     ╚██████╔╝╚██████╔╝
╚═╝      ╚═════╝       ╚═════╝  ╚═════╝ 
`

var (
	logoGradient = []string{
		"#FF5370", "#F78C6C", "#FFCB6B", "#C3E88D", "#89DDFF", "#82AAFF", "#C792EA",
	}

	bigTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			MarginBottom(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#C792EA")).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#888888"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5370"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C3E88D"))

	confirmButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#FF5370")).
				PaddingLeft(1).
				PaddingRight(1)

	cancelButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#555555")).
				PaddingLeft(1).
				PaddingRight(1)

	highlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#82AAFF"))
)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	state            string
	goVersions       []string
	goInstallPath    string
	list             list.Model
	spinner          spinner.Model
	textInput        textinput.Model
	deletionComplete bool
	width            int
	height           int
	err              error
}

func initialModel() model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Type 'yes' to confirm"
	ti.Focus()
	ti.CharLimit = 3
	ti.Width = 20

	return model{
		state:            "loading",
		goVersions:       []string{},
		goInstallPath:    "",
		spinner:          sp,
		textInput:        ti,
		deletionComplete: false,
		width:            80,
		height:           24,
		err:              nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		findGoVersionsCmd,
	)
}

type foundGoVersions struct {
	versions []string
	path     string
	err      error
}

func findGoVersionsCmd() tea.Msg {
	var goPath string
	var versions []string

	switch runtime.GOOS {
	case "windows":
		goPath = filepath.Join(os.Getenv("USERPROFILE"), "go")
		if _, err := os.Stat(goPath); os.IsNotExist(err) {
			goPath = filepath.Join(os.Getenv("ProgramFiles"), "Go")
		}
	case "darwin":
		goPath = "/usr/local/go"
		brewGoPath := "/usr/local/Cellar/go"
		if _, err := os.Stat(brewGoPath); err == nil {
			goPath = brewGoPath
		}
	default:
		goPath = "/usr/local/go"
		if _, err := os.Stat("/usr/bin/go"); err == nil {
			cmd := exec.Command("which", "go")
			if output, err := cmd.Output(); err == nil {
				goPath = strings.TrimSpace(string(output))
				goPath = strings.TrimSuffix(goPath, "/bin/go")
			}
		}
	}

	if _, err := os.Stat(goPath); err == nil {
		cmd := exec.Command("go", "version")
		if output, err := cmd.Output(); err == nil {
			versionStr := strings.TrimSpace(string(output))
			versions = append(versions, versionStr)
		}

		homeDir, err := os.UserHomeDir()
		if err == nil {
			gvmPath := filepath.Join(homeDir, ".gvm", "gos")
			if _, err := os.Stat(gvmPath); err == nil {
				entries, _ := os.ReadDir(gvmPath)
				for _, entry := range entries {
					if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
						versions = append(versions, "go "+entry.Name())
					}
				}
			}
		}
	}

	if len(versions) == 0 {
		cmd := exec.Command("go", "version")
		if output, err := cmd.Output(); err == nil {
			versionStr := strings.TrimSpace(string(output))
			versions = append(versions, versionStr)
		}
	}

	return foundGoVersions{
		versions: versions,
		path:     goPath,
		err:      nil,
	}
}

type deleteGoCompleted struct {
	success bool
	err     error
}

func deleteGoVersionsCmd(path string) tea.Cmd {
	return func() tea.Msg {
		var err error

		tempFile := filepath.Join(path, "fugo-test-file")
		if err = os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
			return deleteGoCompleted{success: false, err: fmt.Errorf("no write permission: %v", err)}
		}
		os.Remove(tempFile)

		if err = os.RemoveAll(path); err != nil {
			return deleteGoCompleted{success: false, err: err}
		}

		homeDir, err := os.UserHomeDir()
		if err == nil {
			gvmPath := filepath.Join(homeDir, ".gvm", "gos")
			if _, err := os.Stat(gvmPath); err == nil {
				entries, _ := os.ReadDir(gvmPath)
				for _, entry := range entries {
					if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
						versionPath := filepath.Join(gvmPath, entry.Name())
						os.RemoveAll(versionPath)
					}
				}
			}
		}

		return deleteGoCompleted{success: true, err: nil}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == "complete" {
				return m, tea.Quit
			}
			return m, tea.Quit
		case "enter":
			if m.state == "confirm" {
				if strings.ToLower(m.textInput.Value()) == "yes" {
					m.state = "deleting"
					return m, tea.Batch(
						m.spinner.Tick,
						deleteGoVersionsCmd(m.goInstallPath),
					)
				}
				return m, tea.Quit
			} else if m.state == "complete" {
				return m, tea.Quit
			}
		}

	case foundGoVersions:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.goVersions = msg.versions
		m.goInstallPath = msg.path

		items := []list.Item{}
		for _, v := range m.goVersions {
			items = append(items, item{title: v, desc: "Will be removed"})
		}

		m.state = "confirm"
		return m, nil

	case deleteGoCompleted:
		m.state = "complete"
		m.deletionComplete = msg.success
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.list.Items() != nil {
			top, right, bottom, left := lipgloss.NewStyle().Margin(2).GetMargin()
			m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom-10)
		}
	}

	if m.state == "confirm" {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func renderFuGoLogo(width int) string {
	lines := strings.Split(fugoASCII, "\n")
	coloredLines := make([]string, len(lines))

	for i, line := range lines {
		if len(line) == 0 {
			coloredLines[i] = line
			continue
		}

		var coloredLine strings.Builder
		for j, char := range line {
			colorIndex := j % len(logoGradient)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(logoGradient[colorIndex]))
			coloredLine.WriteString(style.Render(string(char)))
		}
		coloredLines[i] = coloredLine.String()
	}

	logo := strings.Join(coloredLines, "\n")
	styledLogo := bigTitleStyle.Render(logo)

	if width > 0 {
		styledLogo = lipgloss.PlaceHorizontal(width, lipgloss.Center, styledLogo)
	}

	return styledLogo
}

func (m model) View() string {
	var s string

	s = renderFuGoLogo(m.width) + "\n"

	s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, subtitleStyle.Render("The Go Version Uninstaller")) + "\n\n"

	switch m.state {
	case "loading":
		loadingMsg := fmt.Sprintf("%s Searching for Go installations...", m.spinner.View())
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loadingMsg) + "\n"

	case "confirm":
		if len(m.goVersions) == 0 {
			s += warningStyle.Render("No Go installations found!") + "\n"
			s += "If you believe Go is installed but not detected, please run this tool with admin/sudo privileges.\n"
			s += "\nPress q to quit."
			return s
		}

		s += highlightStyle.Render(fmt.Sprintf("Found %d Go installation(s):", len(m.goVersions))) + "\n\n"
		for _, v := range m.goVersions {
			s += fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCB6B")).Render("•"),
				v)
		}
		s += "\n"
		s += warningStyle.Render("WARNING: This will delete ALL Go installations from your system!") + "\n"
		s += infoStyle.Render("Installation path: "+m.goInstallPath) + "\n\n"
		s += "To confirm deletion, " + m.textInput.View() + "\n\n"
		s += confirmButtonStyle.Render("ENTER") + " to continue, " + cancelButtonStyle.Render("q") + " to quit\n"

	case "deleting":
		deletingMsg := fmt.Sprintf("%s Removing Go installations...", m.spinner.View())
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, deletingMsg) + "\n"

	case "complete":
		if m.err != nil {
			errorMsg := warningStyle.Render("Error: " + m.err.Error())
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, errorMsg) + "\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "You may need to run this tool with admin/sudo privileges.") + "\n"
		} else if m.deletionComplete {
			successMsg := successStyle.Render("✨ Success! All Go installations have been removed. ✨")
			confirmMsg := warningStyle.Render("Enjoy loneliness")

			successBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#C3E88D")).
				Padding(1).
				Render(successMsg + "\n\n" + confirmMsg)

			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, successBox) + "\n\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "You may need to clean up your PATH environment variable manually.") + "\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "Press ENTER or Q to exit") + "\n"
		}
	}

	if m.err != nil && m.state != "complete" {
		s += warningStyle.Render("Error: "+m.err.Error()) + "\n"
	}

	return s
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
