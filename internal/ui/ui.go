package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Zappo-II/mastermind/internal/model"
)

const (
	ESC     = "\033"
	CSI     = ESC + "["
	CUU     = "A"
	CUD     = "B"
	CUF     = "C"
	CUB     = "D"
	ED      = "J"
	EL      = "K"
	Save    = "s"
	Restore = "r"
	Bold    = "1m"
	Reset   = "0m"
)

func MoveCursorUp(n int) string    { return CSI + strconv.Itoa(n) + CUU }
func MoveCursorDown(n int) string  { return CSI + strconv.Itoa(n) + CUD }
func MoveCursorRight(n int) string { return CSI + strconv.Itoa(n) + CUF }
func MoveCursorLeft(n int) string  { return CSI + strconv.Itoa(n) + CUB }
func ClearScreen() string          { return CSI + "2" + ED + CSI + "H" }
func ClearLine() string            { return CSI + EL }
func SaveCursor() string           { return CSI + Save }
func RestoreCursor() string        { return CSI + Restore }

func SetForeground(r, g, b int) string {
	return fmt.Sprintf(CSI+"38;2;%d;%d;%dm", r, g, b)
}
func SetBackground(r, g, b int) string {
	return fmt.Sprintf(CSI+"48;2;%d;%d;%dm", r, g, b)
}

func HexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func ColorBlock(bgR, bgG, bgB, fgR, fgG, fgB int, text string) string {
	return SetForeground(fgR, fgG, fgB) + SetBackground(bgR, bgG, bgB) + text + CSI + Reset
}

func boolToOnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

func divider(width int) string {
	return CSI + "2m" + strings.Repeat("═", width) + CSI + "0m" + "\n"
}

func centerText(text string, width int) string {
	textLen := utf8.RuneCountInString(text)
	if textLen >= width {
		return text
	}
	left := (width - textLen) / 2
	right := width - textLen - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

// ComputeMaxWidth determines the dynamic width for all UI elements.
// Uses worst-case rules text and legend/guess block area width.
func ComputeMaxWidth(level model.LevelConfig) int {
	turnsStr := "\u221e" // ∞
	if level.MaxTurns > 0 {
		turnsStr = fmt.Sprintf("%d/%d", level.MaxTurns, level.MaxTurns)
	}
	rulesLine := fmt.Sprintf("RULES: Code=%d, Colors=%d, Duplicates=%s, Ordered=%s, Turns=%s",
		level.CodeLength, level.MaxColors, boolToOnOff(level.Duplicates),
		boolToOnOff(level.OrderedFeedback), turnsStr)
	rulesWidth := utf8.RuneCountInString(rulesLine)

	// Legend and guess blocks: 4 visible chars each, no separators
	legendWidth := level.MaxColors * 4
	guessWidth := level.CodeLength * 4

	w := rulesWidth
	if legendWidth > w {
		w = legendWidth
	}
	if guessWidth > w {
		w = guessWidth
	}
	if w < 40 {
		w = 40
	}
	return w
}

// centerPad computes left padding to center numBlocks blocks (each 4 chars) within maxWidth.
func centerPad(numBlocks int, maxWidth int) int {
	pad := (maxWidth - numBlocks*4) / 2
	if pad < 0 {
		return 0
	}
	return pad
}

func RenderRulesHeader(state *model.GameState, maxWidth int) string {
	var b strings.Builder
	b.WriteString(divider(maxWidth))

	level := state.Level
	turnsStr := "\u221e"
	if level.MaxTurns > 0 {
		turnsStr = fmt.Sprintf("%d/%d", state.CurrentTurn, level.MaxTurns)
	}

	fmt.Fprintf(&b, "RULES: Code=%d, Colors=%d, Duplicates=%s, Ordered=%s, Turns=%s\n",
		level.CodeLength, level.MaxColors, boolToOnOff(level.Duplicates),
		boolToOnOff(level.OrderedFeedback), turnsStr)

	if level.Hint.Enabled {
		hintParts := []string{"Hints=ON"}
		if level.Hint.Cost == 0 {
			hintParts = append(hintParts, "Cost=free")
		} else {
			hintParts = append(hintParts, fmt.Sprintf("Cost=%d", level.Hint.Cost))
		}
		if level.Hint.CostFactor > 1 {
			hintParts = append(hintParts, fmt.Sprintf("Factor=x%d", level.Hint.CostFactor))
		}
		if level.Hint.MaxHints > 0 {
			hintParts = append(hintParts, fmt.Sprintf("Max=%d", level.Hint.MaxHints))
		}
		if level.Hint.SortByOrder {
			hintParts = append(hintParts, "Mode=positional")
		} else {
			hintParts = append(hintParts, "Mode=colors-first")
		}
		b.WriteString("HINTS: " + strings.Join(hintParts, ", ") + "\n")
	} else {
		b.WriteString("HINTS: OFF\n")
	}

	b.WriteString(divider(maxWidth))
	return b.String()
}

func RenderColorPalette(state *model.GameState, maxWidth int) string {
	var b strings.Builder
	palette := state.Palette
	level := state.Level

	legendPad := centerPad(level.MaxColors, maxWidth)
	b.WriteString(strings.Repeat(" ", legendPad))
	for i := 1; i <= level.MaxColors; i++ {
		if s := renderColorNum(palette, i, ""); s != "" {
			b.WriteString(s)
		}
	}
	b.WriteString("\n")
	b.WriteString(divider(maxWidth))
	return b.String()
}

func RenderSecretShield(state *model.GameState, maxWidth int) string {
	var b strings.Builder
	level := state.Level
	palette := state.Palette
	codePad := centerPad(level.CodeLength, maxWidth)
	b.WriteString(strings.Repeat(" ", codePad))

	if state.Won || state.Lost {
		// Reveal secret code as colored blocks
		for _, colorNum := range state.SecretCode {
			b.WriteString(renderColorNum(palette, colorNum, fmt.Sprintf(" %02d ", colorNum)))
		}
	} else {
		// Hidden shield
		for i := 0; i < level.CodeLength; i++ {
			b.WriteString(CSI + "2m" + "[??]" + CSI + "0m")
		}
	}
	b.WriteString("\n")
	b.WriteString(strings.Repeat("\u2500", maxWidth) + "\n")
	return b.String()
}

// renderColorNum renders a color number as a colored block using the palette.
// Returns fallback if the color is not found.
func renderColorNum(palette model.PaletteConfig, colorNum int, fallback string) string {
	c := palette.GetColor(colorNum)
	if c == nil {
		return fallback
	}
	bgR, bgG, bgB := HexToRGB(c.RGB)
	fgR, fgG, fgB := HexToRGB(c.FG)
	return ColorBlock(bgR, bgG, bgB, fgR, fgG, fgB, fmt.Sprintf(" %02d ", colorNum))
}

// feedbackSymbol renders a feedback peg symbol with the given palette color.
func feedbackSymbol(palette model.PaletteConfig, colorNum int, symbol string) string {
	c := palette.GetColor(colorNum)
	if c == nil {
		return ""
	}
	bgR, bgG, bgB := HexToRGB(c.RGB)
	fgR, fgG, fgB := HexToRGB(c.FG)
	return ColorBlock(bgR, bgG, bgB, fgR, fgG, fgB, symbol)
}

func renderHintRow(b *strings.Builder, turn model.Turn, palette model.PaletteConfig, codeLength int, row int, turns []model.Turn) int {
	if turn.HintData != nil {
		onlyIdx := 0
		for i := 0; i < codeLength; i++ {
			if i < len(turn.HintData.Slots) && turn.HintData.Slots[i] > 0 {
				b.WriteString(renderColorNum(palette, turn.HintData.Slots[i], "[  ]"))
			} else if onlyIdx < len(turn.HintData.OnlyColors) {
				colorNum := turn.HintData.OnlyColors[onlyIdx]
				b.WriteString(CSI + "2m" + fmt.Sprintf("[%02d]", colorNum) + CSI + "0m")
				onlyIdx++
			} else {
				b.WriteString("[  ]")
			}
		}
		label := fmt.Sprintf("  Hint #%d", turn.HintData.Number)
		if turn.HintData.Cost > 0 {
			label += fmt.Sprintf(" Cost %d", turn.HintData.Cost)
		}
		b.WriteString(CSI + "2m" + label + CSI + "0m")
		return row
	}
	// Cost-only row — count consecutive cost rows and consolidate
	costCount := 1
	for row+1 < len(turns) && turns[row+1].IsHint && turns[row+1].HintData == nil {
		costCount++
		row++
	}
	rowWidth := codeLength * 4
	text := fmt.Sprintf(" cost x%d ", costCount)
	fillWidth := rowWidth - 2 - len(text)
	left := fillWidth / 2
	right := fillWidth - left
	bar := strings.Repeat("\u2500", left) + text + strings.Repeat("\u2500", right)
	b.WriteString(CSI + "2m" + "[" + bar + "]" + CSI + "0m")
	return row
}

func renderGuessRow(b *strings.Builder, turn model.Turn, palette model.PaletteConfig) {
	for _, colorNum := range turn.Guess {
		b.WriteString(renderColorNum(palette, colorNum, "    "))
	}
	b.WriteString(" ")

	if turn.Feedback.Positions != nil {
		exactSym := feedbackSymbol(palette, palette.Feedback.Exact.ColorNumber, palette.Feedback.Exact.Symbol)
		partialSym := feedbackSymbol(palette, palette.Feedback.Partial.ColorNumber, palette.Feedback.Partial.Symbol)
		emptySym := feedbackSymbol(palette, palette.Feedback.Empty.ColorNumber, palette.Feedback.Empty.Symbol)
		for _, pos := range turn.Feedback.Positions {
			switch pos {
			case model.FeedbackExact:
				b.WriteString(exactSym)
			case model.FeedbackPartial:
				b.WriteString(partialSym)
			default:
				b.WriteString(emptySym)
			}
		}
	} else {
		b.WriteString(strings.Repeat(feedbackSymbol(palette, palette.Feedback.Exact.ColorNumber, palette.Feedback.Exact.Symbol), turn.Feedback.Exact))
		b.WriteString(strings.Repeat(feedbackSymbol(palette, palette.Feedback.Partial.ColorNumber, palette.Feedback.Partial.Symbol), turn.Feedback.Partial))
	}
}

func renderInputRow(b *strings.Builder, currentGuess []int, palette model.PaletteConfig, codeLength int) {
	for i := 0; i < codeLength; i++ {
		if i < len(currentGuess) && currentGuess[i] > 0 {
			b.WriteString(renderColorNum(palette, currentGuess[i], ""))
		} else if i == len(currentGuess) {
			b.WriteString("[ _]")
		} else {
			b.WriteString("[  ]")
		}
	}
}

func renderEmptyRow(b *strings.Builder, codeLength int) {
	for i := 0; i < codeLength; i++ {
		b.WriteString("[  ]")
	}
}

func RenderTurnRows(state *model.GameState, currentGuess []int, maxWidth int) string {
	var b strings.Builder
	level := state.Level
	palette := state.Palette

	guessPad := centerPad(level.CodeLength, maxWidth)
	pad := strings.Repeat(" ", guessPad)

	// Dynamic row count: for large or unlimited games, show filled + buffer
	var maxRows int
	if level.MaxTurns == 0 || level.MaxTurns > 20 {
		maxRows = len(state.Turns) + 5
		if level.MaxTurns > 0 {
			remaining := level.MaxTurns - state.CurrentTurn
			total := len(state.Turns) + remaining
			if maxRows > total {
				maxRows = total
			}
		}
	} else {
		remaining := level.MaxTurns - state.CurrentTurn
		maxRows = len(state.Turns) + remaining
	}

	for row := 0; row < maxRows; row++ {
		b.WriteString(pad)

		if row < len(state.Turns) {
			turn := state.Turns[row]
			if turn.IsHint {
				row = renderHintRow(&b, turn, palette, level.CodeLength, row, state.Turns)
			} else {
				renderGuessRow(&b, turn, palette)
			}
		} else if row == len(state.Turns) {
			renderInputRow(&b, currentGuess, palette, level.CodeLength)
		} else {
			renderEmptyRow(&b, level.CodeLength)
		}
		b.WriteString("\n")
	}

	// Show remaining turns indicator for large games with hidden rows
	if level.MaxTurns > 20 {
		remaining := level.MaxTurns - state.CurrentTurn
		total := len(state.Turns) + remaining
		if maxRows < total {
			b.WriteString(pad + CSI + "2m" + fmt.Sprintf("... %d turns remaining", remaining) + CSI + "0m" + "\n")
		}
	}

	return b.String()
}

func RenderHelp() string {
	var b strings.Builder
	w := 37
	b.WriteString("\r\n")
	b.WriteString("\u250c" + strings.Repeat("\u2500", w) + "\u2510\n")
	b.WriteString("\u2502" + centerText("CONTROLS", w) + "\u2502\n")
	b.WriteString("\u251c" + strings.Repeat("\u2500", w) + "\u2524\n")
	lines := []struct{ key, desc string }{
		{"0-9", "Select color"},
		{"Enter", "Confirm guess row"},
		{"Bksp", "Undo last slot"},
		{"h", "Request hint"},
		{"q", "Quit"},
		{"?", "This help"},
	}
	for _, l := range lines {
		fmt.Fprintf(&b, "\u2502  %-10s%-*s\u2502\n", l.key, w-12, l.desc)
	}
	b.WriteString("\u2514" + strings.Repeat("\u2500", w) + "\u2518\n")
	return b.String()
}

func RenderWinMessage(maxWidth int) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(CSI + "32m")
	b.WriteString(strings.Repeat("\u2550", maxWidth) + "\n")
	b.WriteString(centerText("YOU WON!", maxWidth) + "\n")
	b.WriteString(strings.Repeat("\u2550", maxWidth) + "\n")
	b.WriteString(CSI + "0m")
	return b.String()
}

func RenderLoseMessage(maxWidth int) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(CSI + "31m")
	b.WriteString(strings.Repeat("\u2550", maxWidth) + "\n")
	b.WriteString(centerText("GAME OVER", maxWidth) + "\n")
	b.WriteString(strings.Repeat("\u2550", maxWidth) + "\n")
	b.WriteString(CSI + "0m")
	return b.String()
}

func RenderIntroScreen(level model.LevelConfig) string {
	var b strings.Builder
	b.WriteString(ClearScreen())
	b.WriteString("\n\n")
	b.WriteString(CSI + "1m" + CSI + "36m")
	b.WriteString("    \u25cf \u25cb \u25cf \u25cb  M A S T E R M I N D  \u25cb \u25cf \u25cb \u25cf\n")
	b.WriteString(CSI + "0m")
	b.WriteString("\n")
	b.WriteString(CSI + "2m")
	b.WriteString("    A classic code-breaking game for the terminal\n")
	b.WriteString("    MIT License \u2022 \u00a9 2026 Zappo-II\n")
	b.WriteString("    github.com/Zappo-II/mastermind\n")
	b.WriteString("\n")
	b.WriteString("    Mastermind\u00ae is a trademark of Invicta Plastics Ltd.\n")
	b.WriteString("    Not affiliated with Invicta Plastics or Hasbro.\n")
	b.WriteString(CSI + "0m")
	b.WriteString("\n")
	b.WriteString("    " + CSI + "1m" + "HOW TO PLAY" + CSI + "0m" + "\n\n")
	fmt.Fprintf(&b, "    Guess the secret %d-color code from %d available colors.\n", level.CodeLength, level.MaxColors)
	b.WriteString("    After each guess, feedback pegs tell you how close you are:\n\n")
	b.WriteString("      \u25cf = Right color, right position\n")
	b.WriteString("      \u25cb = Right color, wrong position\n\n")
	b.WriteString("    " + CSI + "1m" + "CONTROLS" + CSI + "0m" + "\n\n")
	fmt.Fprintf(&b, "      1-%-7s Enter color number\n", strconv.Itoa(level.MaxColors))
	b.WriteString("      Enter     Confirm digit or guess row\n")
	b.WriteString("      Space     Confirm digit\n")
	b.WriteString("      Bksp      Undo last slot\n")
	b.WriteString("      h         Get a hint\n")
	b.WriteString("      ?         Help\n")
	b.WriteString("      q         Quit\n\n")
	b.WriteString("    Press any key to start...")
	return b.String()
}
