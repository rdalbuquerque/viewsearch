package viewsearch

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type LogMsg string

type searchResult struct {
	Line  int
	Index int
}

type Model struct {
	SelectedResultStyle lipgloss.Style
	ResultStyle         lipgloss.Style
	Viewport            viewport.Model
	searchResults       []searchResult
	searchMode          bool
	ta                  textarea.Model
	originalContent     string
	currentResultIndex  int
	navigationMode      bool
	HelpBindings        []key.Binding
	showHelp            bool
	height              int
	width               int
}

var (
	noResultsStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#b22222")) // red

	searchPromptStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#cd00cd")) // bright magenta
	navigationPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")) // gray
)

func defaultStyles() (lipgloss.Style, lipgloss.Style) {
	selectedResultStyle := lipgloss.NewStyle().Background(lipgloss.Color("#00FF00"))
	resultStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF00FF"))
	return selectedResultStyle, resultStyle
}

func New() Model {
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "find")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "navigate results")),
		key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "exit find")),
		key.NewBinding(key.WithKeys("n", "N"), key.WithHelp("n/N", "forward/backward")),
	}
	vp := viewport.New()
	ta := textarea.New()
	ta.ShowLineNumbers = false
	ta.Prompt = "/"

	selectedResultStyle, resultStyle := defaultStyles()

	m := Model{
		SelectedResultStyle: selectedResultStyle,
		ResultStyle:         resultStyle,
		Viewport:            vp,
		ta:                  ta,
		HelpBindings:        bindings,
	}
	m.SetShowHelp(true)
	return m
}

const searchCounterReservedWidth = 10

func (m *Model) setTextAreaWidth(viewportWidth int) {
	taWidth := min(80, viewportWidth-searchCounterReservedWidth)
	if taWidth < 1 {
		taWidth = 1
	}
	m.ta.SetWidth(taWidth)
}

func (m *Model) SetDimensions(width, height int) {
	m.height = height
	m.width = width
	m.Viewport.SetWidth(width)
	m.setTextAreaWidth(width)
	m.setHeights()
}

func (m *Model) GotoBottom() {
	m.Viewport.GotoBottom()
}

func (m *Model) SetContent(content string) {
	m.originalContent = content
	m.Viewport.SetContent(content)
	if m.searchMode {
		m.highlightMatches()
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.searchMode {
				m.ExitSearch()
				return m, nil
			}
		case "/", "alt+/", "alt+ctrl+_":
			if m.navigationMode {
				m.handleDeactivations()
				return m, nil
			}
			return m, m.handleSearchActivation(msg)
		case "backspace":
			if m.searchMode && m.ta.Length() == 0 {
				return m, m.handleDeactivations()
			}
		case "enter":
			return m, m.handleNavigationActivation()
		case "n":
			return m, m.handleNavigationForward(msg)
		case "N":
			return m, m.handleNavigationBackwards(msg)
		}
	}
	var tacmd tea.Cmd
	if m.searchMode && !m.navigationMode {
		tacmd = m.updateTextArea(msg)
		if m.searchMode {
			m.Viewport.SetContent(m.originalContent)
			m.highlightMatches()
		}
	}
	vpcmd := m.updateViewPort(msg)
	return m, tea.Batch(vpcmd, tacmd)
}

func (m *Model) updateTextArea(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return cmd
}

func (m *Model) updateViewPort(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return cmd
}

func (m *Model) handleNavigationForward(msg tea.Msg) tea.Cmd {
	if m.navigationMode {
		m.navigateToNextResult()
		return m.updateViewPort(msg)
	}
	return m.updateTextArea(msg)
}

func (m *Model) handleNavigationBackwards(msg tea.Msg) tea.Cmd {
	if m.navigationMode {
		m.navigateToPreviousResult()
		return m.updateViewPort(msg)
	}
	return m.updateTextArea(msg)
}

func (m *Model) handleNavigationActivation() tea.Cmd {
	if m.searchMode {
		m.navigationMode = true
		m.updatePromptStyle()
		return nil
	}
	return nil
}

func (m *Model) handleSearchActivation(msg tea.Msg) tea.Cmd {
	if m.searchMode {
		return m.updateTextArea(msg)
	}
	if !m.searchMode {
		m.setShowSearch(true)
		return nil
	}
	return nil
}

func (m *Model) setShowSearch(v bool) {
	m.searchMode = v
	if v {
		m.ta.Focus()
		m.updatePromptStyle()
	}
	m.setHeights()
}

func (m *Model) updatePromptStyle() {
	s := m.ta.Styles()
	if m.navigationMode {
		s.Focused.Prompt = navigationPromptStyle
	} else {
		s.Focused.Prompt = searchPromptStyle
	}
	m.ta.SetStyles(s)
}

func (m *Model) ExitSearch() {
	m.navigationMode = false
	m.searchResults = nil
	m.currentResultIndex = 0
	m.ta.Reset()
	m.setShowSearch(false)
	m.Viewport.SetContent(m.originalContent)
	m.Viewport.SetXOffset(0)
}

func (m *Model) handleDeactivations() tea.Cmd {
	if m.navigationMode {
		m.navigationMode = false
		m.updatePromptStyle()
		m.ta.Focus()
		return nil
	}
	if m.searchMode {
		m.setShowSearch(false)
		m.Viewport.SetXOffset(0)
		return nil
	}
	return nil
}

func (m *Model) SearchActive() bool {
	return m.searchMode
}

func (m *Model) SetShowHelp(v bool) {
	m.showHelp = v
	m.setHeights()
}

func (m *Model) setHeights() {
	viewportHeight := m.height
	if m.showHelp {
		viewportHeight -= 1
	}
	if m.searchMode {
		m.ta.SetHeight(1)
		viewportHeight -= 1
	}
	m.Viewport.SetHeight(viewportHeight)
}

func (m *Model) View() string {
	searchCounter := fmt.Sprintf(" %d/%d", m.currentResultIndex+1, len(m.searchResults))
	if !m.hasSearchResults() {
		searchCounter = noResultsStyle.Render(" 0!")
	}
	taView := m.ta.View()
	renderedViewPort := m.Viewport.View()
	viewsearchView := renderedViewPort
	if m.searchMode {
		viewsearchView = lipgloss.JoinVertical(lipgloss.Top, lipgloss.JoinHorizontal(lipgloss.Left, taView, searchCounter), viewsearchView)
	}
	if m.showHelp {
		viewsearchView = lipgloss.JoinVertical(lipgloss.Top, viewsearchView, help.New().ShortHelpView(m.HelpBindings))
	}
	return viewsearchView
}

func (m *Model) highlightMatches() {
	searchQuery := m.ta.Value()
	if searchQuery == "" {
		return
	}

	m.resetSearchResults()
	m.findAndHighlightMatches(searchQuery)
}

func (m *Model) resetSearchResults() {
	m.searchResults = []searchResult{}
}

func (m *Model) findAndHighlightMatches(searchQuery string) {
	lines := strings.Split(m.originalContent, "\n")
	var processedLines []string
	for i, line := range lines {
		processedLines = append(processedLines, m.processLineForCaseInsensitiveMatches(i, line, searchQuery))
	}
	m.Viewport.SetContent(strings.Join(processedLines, "\n"))
}

func (m *Model) processLineForCaseInsensitiveMatches(lineIndex int, line, searchQuery string) string {
	var highlightedLine string
	var startPos int

	lowercaseline := strings.ToLower(line)
	lowercasesearchQuery := strings.ToLower(searchQuery)

	for {
		index := strings.Index(lowercaseline[startPos:], lowercasesearchQuery)
		if index < 0 {
			highlightedLine += line[startPos:]
			break
		}

		m.storeSearchResult(lineIndex, startPos+index)
		highlightedLine += m.highlightMatch(lineIndex, startPos, index, lowercasesearchQuery, line)
		startPos += index + len(lowercasesearchQuery)
	}

	return highlightedLine
}

func (m *Model) highlightMatch(lineIndex, startPos, index int, searchQuery, line string) string {
	styleToUse := m.setHighlightStyle(lineIndex, startPos+index)
	matchedPart := line[startPos+index : startPos+index+len(searchQuery)]
	return line[startPos:startPos+index] + styleToUse.Render(matchedPart)
}

func (m *Model) storeSearchResult(line, index int) {
	m.searchResults = append(m.searchResults, searchResult{Line: line, Index: index})
}

func (m *Model) setHighlightStyle(lineIndex, index int) lipgloss.Style {
	if m.currentResultIndex >= 0 && m.currentResultIndex < len(m.searchResults) {
		if lineIndex == m.searchResults[m.currentResultIndex].Line && index == m.searchResults[m.currentResultIndex].Index {
			return m.SelectedResultStyle
		}
	}
	return m.ResultStyle
}

func (m *Model) navigateToNextResult() {
	if !m.hasSearchResults() {
		return
	}
	m.incrementSearchIndex()
	m.scrollToCurrentResult()
	m.highlightMatches()
}

func (m *Model) navigateToPreviousResult() {
	if !m.hasSearchResults() {
		return
	}
	m.decrementSearchIndex()
	m.scrollToCurrentResult()
	m.highlightMatches()
}

func (m *Model) hasSearchResults() bool {
	return len(m.searchResults) > 0
}

func (m *Model) incrementSearchIndex() {
	m.currentResultIndex = (m.currentResultIndex + 1) % len(m.searchResults)
}

func (m *Model) decrementSearchIndex() {
	m.currentResultIndex = m.currentResultIndex - 1
	if m.currentResultIndex < 0 {
		m.currentResultIndex = len(m.searchResults) - 1
	}
}

func (m *Model) scrollToCurrentResult() {
	nextResult := m.searchResults[m.currentResultIndex]
	m.scrollViewportToLine(nextResult.Line)
	searchQuery := m.ta.Value()
	m.scrollViewportToColumn(nextResult.Line, nextResult.Index, len(searchQuery))
}

func (m *Model) scrollViewportToColumn(lineIndex, byteIndex, matchLen int) {
	lines := strings.Split(m.originalContent, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return
	}
	line := lines[lineIndex]
	if byteIndex > len(line) {
		byteIndex = len(line)
	}
	matchEnd := min(byteIndex+matchLen, len(line))

	// Convert byte offsets to display columns
	colStart := lipgloss.Width(line[:byteIndex])
	colEnd := lipgloss.Width(line[:matchEnd])

	leftEdge := m.Viewport.XOffset()
	rightEdge := leftEdge + m.Viewport.Width()

	// Ensure the full match is visible
	if colEnd > rightEdge {
		m.Viewport.SetXOffset(colEnd - m.Viewport.Width())
	}
	if colStart < m.Viewport.XOffset() {
		m.Viewport.SetXOffset(colStart)
	}
}

func (m *Model) scrollViewportToLine(line int) {
	// Check if the resultLine is currently visible
	topLine := m.Viewport.YOffset()
	bottomLine := topLine + m.Viewport.Height() - 1 // -1 because it's zero-based index
	for line < topLine || line > bottomLine {
		if line < topLine {
			m.Viewport.ScrollUp(m.Viewport.Height())
		} else {
			m.Viewport.ScrollDown(m.Viewport.Height())
		}

		// Update topLine and bottomLine after scrolling
		topLine = m.Viewport.YOffset()
		bottomLine = topLine + m.Viewport.Height() - 1
	}
}
