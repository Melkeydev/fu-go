package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

var criticalPaths = []string{
	"/", "/usr", "/bin", "/etc", "/home", "/root", "/var", "/opt",
	"C:\\", "C:\\Windows", "C:\\Program Files", "C:\\Users",
}

type GoInstallation struct {
	Path        string
	Version     string
	Source      string // "official", "gvm", "snap", "brew", "package_manager"
	Size        int64
	Permissions string
	Verified    bool
}

type Logger struct {
	file *os.File
}

func NewLogger() (*Logger, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	logDir := filepath.Join(homeDir, ".fugo")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	logFile := filepath.Join(logDir, fmt.Sprintf("fugo_%s.log", timestamp))

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %v", err)
	}

	return &Logger{file: file}, nil
}

func (l *Logger) Log(level, message string) {
	if l.file == nil {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)
	l.file.WriteString(logEntry)
	l.file.Sync()
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func generateSecurityHash() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])[:8]
}

func checkPermissions() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to determine current user: %v", err)
	}

	if runtime.GOOS != "windows" && currentUser.Uid != "0" {
		testPath := "/usr/local/go"
		if _, err := os.Stat(testPath); err == nil {
			testFile := filepath.Join(testPath, "fugo-permission-test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				return fmt.Errorf("insufficient permissions: run with sudo for system-wide Go installations")
			}
			os.Remove(testFile)
		}
	}

	return nil
}

func createBackup(sourcePath, backupDir string) error {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil
	}

	backupName := fmt.Sprintf("go_backup_%s.tar.gz", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(backupDir, backupName)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(sourcePath), filepath.Base(sourcePath))
	} else {
		cmd = exec.Command("tar", "-czf", backupPath, "-C", filepath.Dir(sourcePath), filepath.Base(sourcePath))
	}

	return cmd.Run()
}

func isCriticalPath(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, critical := range criticalPaths {
		if cleanPath == critical {
			return true
		}
	}
	return false
}

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
	confirmationStep int
	dryRun           bool
	backupPath       string
	logFile          *Logger
	hashConfirmation string
	detectedInstalls []GoInstallation
	permissionCheck  bool
}

func initialModel() model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ti := textinput.New()
	ti.Placeholder = "Type 'CONFIRM' to proceed"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 25

	logger, _ := NewLogger()
	hash := generateSecurityHash()

	homeDir, _ := os.UserHomeDir()
	backupDir := filepath.Join(homeDir, ".fugo", "backups")
	os.MkdirAll(backupDir, 0755)

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
		confirmationStep: 0,
		dryRun:           true,
		backupPath:       backupDir,
		logFile:          logger,
		hashConfirmation: hash,
		detectedInstalls: []GoInstallation{},
		permissionCheck:  false,
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
	installs []GoInstallation
	permOk   bool
	err      error
}

func detectGoInstallations() []GoInstallation {
	var installations []GoInstallation

	// Official Go installation
	var officialPaths []string
	switch runtime.GOOS {
	case "windows":
		officialPaths = []string{
			filepath.Join(os.Getenv("USERPROFILE"), "go"),
			filepath.Join(os.Getenv("ProgramFiles"), "Go"),
			"C:\\Go",
		}
	case "darwin":
		officialPaths = []string{
			"/usr/local/go",
			"/opt/go",
		}
	default:
		officialPaths = []string{
			"/usr/local/go",
			"/opt/go",
			"/usr/lib/go",
		}
	}

	for _, path := range officialPaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			version := getGoVersion(path)
			size := getDirSize(path)
			installations = append(installations, GoInstallation{
				Path:        path,
				Version:     version,
				Source:      "official",
				Size:        size,
				Permissions: getPermissions(path),
				Verified:    true,
			})
		}
	}

	// GVM installations
	homeDir, err := os.UserHomeDir()
	if err == nil {
		gvmPath := filepath.Join(homeDir, ".gvm", "gos")
		if entries, err := os.ReadDir(gvmPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), "go") {
					path := filepath.Join(gvmPath, entry.Name())
					version := getGoVersion(path)
					size := getDirSize(path)
					installations = append(installations, GoInstallation{
						Path:        path,
						Version:     version,
						Source:      "gvm",
						Size:        size,
						Permissions: getPermissions(path),
						Verified:    true,
					})
				}
			}
		}
	}

	// Package manager installations (Linux)
	if runtime.GOOS == "linux" {
		packagePaths := []string{"/usr/lib/golang", "/usr/share/golang"}
		for _, path := range packagePaths {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				version := getGoVersion(path)
				size := getDirSize(path)
				installations = append(installations, GoInstallation{
					Path:        path,
					Version:     version,
					Source:      "package_manager",
					Size:        size,
					Permissions: getPermissions(path),
					Verified:    true,
				})
			}
		}
	}

	// Homebrew installations (macOS)
	if runtime.GOOS == "darwin" {
		brewPaths := []string{"/usr/local/Cellar/go", "/opt/homebrew/Cellar/go"}
		for _, basePath := range brewPaths {
			if entries, err := os.ReadDir(basePath); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						path := filepath.Join(basePath, entry.Name())
						version := getGoVersion(path)
						size := getDirSize(path)
						installations = append(installations, GoInstallation{
							Path:        path,
							Version:     version,
							Source:      "brew",
							Size:        size,
							Permissions: getPermissions(path),
							Verified:    true,
						})
					}
				}
			}
		}
	}

	return installations
}

func getGoVersion(goPath string) string {
	goExec := filepath.Join(goPath, "bin", "go")
	if runtime.GOOS == "windows" {
		goExec += ".exe"
	}

	if _, err := os.Stat(goExec); err == nil {
		cmd := exec.Command(goExec, "version")
		if output, err := cmd.Output(); err == nil {
			return strings.TrimSpace(string(output))
		}
	}

	// Fallback: try to determine version from directory structure
	versionFile := filepath.Join(goPath, "VERSION")
	if data, err := os.ReadFile(versionFile); err == nil {
		return "go version " + strings.TrimSpace(string(data))
	}

	return "unknown version"
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func getPermissions(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	return info.Mode().String()
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
				whichPath := strings.TrimSpace(string(output))
				if strings.HasSuffix(whichPath, "/bin/go") {
					derivedPath := strings.TrimSuffix(whichPath, "/bin/go")

					if isCriticalPath(derivedPath) {
						return foundGoVersions{
							versions: []string{},
							path:     "",
							err:      fmt.Errorf("refusing to operate on critical system directory: %s", derivedPath),
						}
					}

					if !strings.Contains(strings.ToLower(derivedPath), "go") {
						return foundGoVersions{
							versions: []string{},
							path:     "",
							err:      fmt.Errorf("derived path does not appear to be a Go installation: %s", derivedPath),
						}
					}

					goPath = derivedPath
				}
			}
		}
	}

	// GUARD RAIL: Final check before proceeding
	if isCriticalPath(goPath) {
		return foundGoVersions{
			versions: []string{},
			path:     "",
			err:      fmt.Errorf("refusing to operate on critical system directory: %s", goPath),
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
	permOk := checkPermissions() == nil
	installations := detectGoInstallations()

	return foundGoVersions{
		versions: versions,
		path:     goPath,
		installs: installations,
		permOk:   permOk,
		err:      nil,
	}
}

type deleteGoCompleted struct {
	success bool
	err     error
}

type backupCompleted struct {
	success bool
	err     error
	path    string
}

func createBackupCmd(installations []GoInstallation, backupDir string) tea.Cmd {
	return func() tea.Msg {
		for _, install := range installations {
			if err := createBackup(install.Path, backupDir); err != nil {
				return backupCompleted{success: false, err: err, path: backupDir}
			}
		}
		return backupCompleted{success: true, err: nil, path: backupDir}
	}
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
			if m.logFile != nil {
				m.logFile.Log("INFO", "User cancelled operation")
				m.logFile.Close()
			}
			return m, tea.Quit
		case "d":
			if m.state == "confirm" {
				m.dryRun = !m.dryRun
				if m.logFile != nil {
					m.logFile.Log("INFO", fmt.Sprintf("Dry run mode: %v", m.dryRun))
				}
				return m, nil
			}
		case "enter":
			if m.state == "confirm" {
				return m.handleConfirmation()
			} else if m.state == "complete" {
				return m, tea.Quit
			}
		}

	case foundGoVersions:
		if msg.err != nil {
			m.err = msg.err
			if m.logFile != nil {
				m.logFile.Log("ERROR", msg.err.Error())
			}
			return m, tea.Quit
		}
		m.goVersions = msg.versions
		m.goInstallPath = msg.path
		m.detectedInstalls = msg.installs
		m.permissionCheck = msg.permOk

		if m.logFile != nil {
			m.logFile.Log("INFO", fmt.Sprintf("Found %d Go installations", len(msg.installs)))
			for _, install := range msg.installs {
				m.logFile.Log("INFO", fmt.Sprintf("Installation: %s (%s, %s)", install.Path, install.Version, install.Source))
			}
		}

		items := []list.Item{}
		for _, v := range m.goVersions {
			items = append(items, item{title: v, desc: "Will be removed"})
		}

		m.state = "confirm"
		return m, nil

	case backupCompleted:
		if msg.err != nil {
			m.err = msg.err
			m.state = "complete"
			if m.logFile != nil {
				m.logFile.Log("ERROR", fmt.Sprintf("Backup failed: %v", msg.err))
			}
			return m, nil
		}
		if m.logFile != nil {
			m.logFile.Log("SUCCESS", fmt.Sprintf("Backup created at: %s", msg.path))
		}
		m.state = "deleting"
		return m, tea.Batch(
			m.spinner.Tick,
			deleteGoVersionsCmd(m.goInstallPath),
		)

	case deleteGoCompleted:
		m.state = "complete"
		m.deletionComplete = msg.success
		m.err = msg.err
		if m.logFile != nil {
			if msg.success {
				m.logFile.Log("SUCCESS", "Go uninstallation completed successfully")
			} else {
				m.logFile.Log("ERROR", fmt.Sprintf("Go uninstallation failed: %v", msg.err))
			}
			m.logFile.Close()
		}
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

func (m model) handleConfirmation() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())

	switch m.confirmationStep {
	case 0:
		if strings.ToUpper(input) == "CONFIRM" {
			m.confirmationStep = 1
			m.textInput.SetValue("")
			m.textInput.Placeholder = fmt.Sprintf("Type hash: %s", m.hashConfirmation)
			if m.logFile != nil {
				m.logFile.Log("INFO", "First confirmation step passed")
			}
			return m, nil
		}
	case 1:
		if input == m.hashConfirmation {
			m.confirmationStep = 2
			m.textInput.SetValue("")
			m.textInput.Placeholder = "Type 'DESTROY' to proceed"
			if m.logFile != nil {
				m.logFile.Log("INFO", "Second confirmation step passed")
			}
			return m, nil
		}
	case 2:
		if strings.ToUpper(input) == "DESTROY" {
			if m.logFile != nil {
				m.logFile.Log("INFO", "All confirmation steps passed, proceeding with operation")
			}
			if m.dryRun {
				m.state = "dry_run_complete"
				return m, nil
			} else {
				m.state = "creating_backup"
				return m, tea.Batch(
					m.spinner.Tick,
					createBackupCmd(m.detectedInstalls, m.backupPath),
				)
			}
		}
	}

	return m, tea.Quit
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

	s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, subtitleStyle.Render("The Go Uninstaller - Enhanced Security Edition")) + "\n\n"

	switch m.state {
	case "loading":
		loadingMsg := fmt.Sprintf("%s Detecting Go installations...", m.spinner.View())
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, loadingMsg) + "\n"

	case "confirm":
		if len(m.detectedInstalls) == 0 {
			s += warningStyle.Render("No Go installations found!") + "\n"
			s += "If you believe Go is installed but not detected, please run this tool with admin/sudo privileges.\n"
			s += "\nPress q to quit."
			return s
		}

		s += highlightStyle.Render(fmt.Sprintf("🔍 Detected %d Go installation(s):", len(m.detectedInstalls))) + "\n\n"
		for _, install := range m.detectedInstalls {
			sizeStr := fmt.Sprintf("%.1f MB", float64(install.Size)/(1024*1024))
			s += fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCB6B")).Render("📦"),
				install.Version)
			s += fmt.Sprintf("     📍 Path: %s\n", install.Path)
			s += fmt.Sprintf("     🔧 Source: %s | 💾 Size: %s\n", install.Source, sizeStr)
			s += fmt.Sprintf("     🔐 Permissions: %s\n\n", install.Permissions)
		}

		// Security status
		if !m.permissionCheck {
			s += warningStyle.Render("⚠️  WARNING: Insufficient permissions detected!") + "\n"
			s += infoStyle.Render("   Run with sudo/admin privileges for complete removal") + "\n\n"
		} else {
			s += successStyle.Render("✅ Permissions check passed") + "\n\n"
		}

		// Dry run status
		if m.dryRun {
			s += highlightStyle.Render("🔍 DRY RUN MODE ENABLED - No files will be deleted") + "\n"
		} else {
			s += warningStyle.Render("🔥 LIVE MODE - Files WILL be permanently deleted!") + "\n"
		}

		s += "\n" + warningStyle.Render("⚠️  CRITICAL WARNING: This will delete ALL Go installations from your system!") + "\n"
		s += infoStyle.Render(fmt.Sprintf("📂 Backup location: %s", m.backupPath)) + "\n\n"

		// Confirmation steps
		switch m.confirmationStep {
		case 0:
			s += "Step 1/3: " + m.textInput.View() + "\n"
		case 1:
			s += "Step 2/3: " + m.textInput.View() + "\n"
		case 2:
			s += "Step 3/3: " + m.textInput.View() + "\n"
		}

		s += "\n" + confirmButtonStyle.Render("ENTER") + " to continue, " + cancelButtonStyle.Render("d") + " toggle dry-run, " + cancelButtonStyle.Render("q") + " to quit\n"

	case "creating_backup":
		backupMsg := fmt.Sprintf("%s Creating safety backup...", m.spinner.View())
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, backupMsg) + "\n"

	case "deleting":
		deletingMsg := fmt.Sprintf("%s Removing Go installations...", m.spinner.View())
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, deletingMsg) + "\n"

	case "dry_run_complete":
		dryMsg := successStyle.Render("🔍 DRY RUN COMPLETED")
		s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, dryMsg) + "\n\n"
		s += "The following operations would be performed:\n\n"
		for _, install := range m.detectedInstalls {
			s += fmt.Sprintf("  ❌ Remove: %s (%s)\n", install.Path, install.Source)
		}
		s += "\n" + infoStyle.Render("No files were actually deleted in dry-run mode") + "\n"
		s += "\nPress ENTER or Q to exit\n"

	case "complete":
		if m.err != nil {
			errorMsg := warningStyle.Render("❌ Error: " + m.err.Error())
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, errorMsg) + "\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "You may need to run this tool with admin/sudo privileges.") + "\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, fmt.Sprintf("💾 Backup available at: %s", m.backupPath)) + "\n"
		} else if m.deletionComplete {
			successMsg := successStyle.Render("✨ Success! All Go installations have been removed. ✨")
			confirmMsg := warningStyle.Render("Enjoy loneliness")
			backupMsg := infoStyle.Render(fmt.Sprintf("💾 Backup created at: %s", m.backupPath))

			successBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#C3E88D")).
				Padding(1).
				Render(successMsg + "\n\n" + confirmMsg + "\n\n" + backupMsg)

			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, successBox) + "\n\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "📋 Check logs at ~/.fugo/ for detailed information") + "\n"
			s += lipgloss.PlaceHorizontal(m.width, lipgloss.Center, "🔧 You may need to clean up your PATH environment variable manually.") + "\n"
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
	teaModel, err := p.Run()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}

	m, ok := teaModel.(model)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unexpected model type\n")
		os.Exit(1)
	}

	if m.err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", m.err)
		os.Exit(1)
	}
}
