package game

import (
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"

	"github.com/Zappo-II/mastermind/internal/model"
)

type Game struct {
	state model.GameState
}

func NewGame(level model.LevelConfig, palette model.PaletteConfig) *Game {
	return &Game{
		state: model.GameState{
			SecretCode:  generateCode(level, palette),
			Turns:       []model.Turn{},
			CurrentTurn: 0,
			Won:         false,
			Lost:        false,
			Level:       level,
			Palette:     palette,
		},
	}
}

func (g *Game) State() *model.GameState {
	return &g.state
}

func generateCode(level model.LevelConfig, palette model.PaletteConfig) []int {
	colors := make([]int, level.MaxColors)
	for i := 0; i < level.MaxColors; i++ {
		colors[i] = i + 1
	}

	code := make([]int, level.CodeLength)

	if !level.Duplicates {
		rand.Shuffle(level.MaxColors, func(i, j int) {
			colors[i], colors[j] = colors[j], colors[i]
		})
		copy(code, colors[:level.CodeLength])
	} else {
		for i := 0; i < level.CodeLength; i++ {
			code[i] = colors[rand.Intn(level.MaxColors)]
		}
	}

	return code
}

func CalculateFeedback(guess, secret []int, ordered bool) model.Feedback {
	exact := 0
	partial := 0

	secretCopy := make([]int, len(secret))
	copy(secretCopy, secret)
	guessCopy := make([]int, len(guess))
	copy(guessCopy, guess)

	// Track which positions are exact matches
	exactPos := make([]bool, len(guess))
	for i := 0; i < len(guess) && i < len(secret); i++ {
		if guessCopy[i] == secretCopy[i] {
			exact++
			exactPos[i] = true
			secretCopy[i] = -1
			guessCopy[i] = -2
		}
	}

	// Always calculate partial matches
	partialPos := make([]bool, len(guess))
	for i := 0; i < len(guess); i++ {
		if guessCopy[i] == -2 {
			continue
		}
		for j := 0; j < len(secret); j++ {
			if secretCopy[j] == -1 {
				continue
			}
			if guessCopy[i] == secretCopy[j] {
				partial++
				partialPos[i] = true
				secretCopy[j] = -1
				break
			}
		}
	}

	fb := model.Feedback{Exact: exact, Partial: partial}

	if ordered {
		positions := make([]model.FeedbackPeg, len(guess))
		for i := range positions {
			if exactPos[i] {
				positions[i] = model.FeedbackExact
			} else if partialPos[i] {
				positions[i] = model.FeedbackPartial
			} else {
				positions[i] = model.FeedbackNone
			}
		}
		fb.Positions = positions
	}

	return fb
}

func (g *Game) MakeGuess(guess []int) model.Feedback {
	feedback := CalculateFeedback(guess, g.state.SecretCode, g.state.Level.OrderedFeedback)

	turn := model.Turn{
		Guess:    make([]int, len(guess)),
		Feedback: feedback,
	}
	copy(turn.Guess, guess)

	g.state.Turns = append(g.state.Turns, turn)
	g.state.CurrentTurn++

	if feedback.Exact == len(g.state.SecretCode) {
		g.state.Won = true
	} else if g.state.Level.MaxTurns > 0 && g.state.CurrentTurn >= g.state.Level.MaxTurns {
		g.state.Lost = true
	}

	return feedback
}

func (g *Game) GetSecretColors() []int {
	unique := []int{}
	for _, c := range g.state.SecretCode {
		if !slices.Contains(unique, c) {
			unique = append(unique, c)
		}
	}
	slices.Sort(unique)
	return unique
}

func (g *Game) GetUsedColorsInOrder() []int {
	seen := make(map[int]bool)
	result := []int{}
	for _, c := range g.state.SecretCode {
		if !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}
	return result
}

// HintCost returns the turn cost for the next hint (cost * costFactor^hintsGiven).
func (g *Game) HintCost() int {
	base := g.state.Level.Hint.Cost
	factor := g.state.Level.Hint.CostFactor
	if factor <= 1 {
		return base
	}
	cost := base
	for i := 0; i < g.state.HintsGiven; i++ {
		cost *= factor
	}
	return cost
}

func (g *Game) ApplyHint(hd *model.HintData) {
	cost := hd.Cost

	// Always insert the hint display row
	g.state.Turns = append(g.state.Turns, model.Turn{
		IsHint:   true,
		HintData: hd,
	})
	if cost > 0 {
		g.state.CurrentTurn++ // hint display row is the first cost turn
	}

	// Additional cost-only rows
	for i := 1; i < cost; i++ {
		g.state.Turns = append(g.state.Turns, model.Turn{IsHint: true})
		g.state.CurrentTurn++
	}

	// Check lose after cost
	if g.state.Level.MaxTurns > 0 && g.state.CurrentTurn >= g.state.Level.MaxTurns {
		g.state.Lost = true
	}
}

// HintAvailable returns true if another hint can be dispensed.
func (g *Game) HintAvailable() bool {
	if !g.state.Level.Hint.Enabled {
		return false
	}
	if g.state.Level.Hint.MaxHints > 0 && g.state.HintsGiven >= g.state.Level.Hint.MaxHints {
		return false
	}
	n := g.state.HintsGiven
	codeLen := g.state.Level.CodeLength
	if g.state.Level.Hint.SortByOrder {
		return n < codeLen
	}
	return n < codeLen*2 // color phase + position phase
}

func (g *Game) GetHint() *model.HintData {
	if !g.state.Level.Hint.Enabled {
		return nil
	}
	if g.state.Level.Hint.MaxHints > 0 && g.state.HintsGiven >= g.state.Level.Hint.MaxHints {
		return nil
	}

	n := g.state.HintsGiven
	codeLen := g.state.Level.CodeLength

	// Initialize shuffled code on first hint (stable random reveal order)
	if g.state.ShuffledCode == nil {
		g.state.ShuffledCode = make([]int, codeLen)
		copy(g.state.ShuffledCode, g.state.SecretCode)
		rand.Shuffle(codeLen, func(i, j int) {
			g.state.ShuffledCode[i], g.state.ShuffledCode[j] = g.state.ShuffledCode[j], g.state.ShuffledCode[i]
		})
	}

	var hd *model.HintData
	if g.state.Level.Hint.SortByOrder {
		hd = hintPositional(&g.state, n, codeLen)
	} else {
		hd = hintColorFirst(&g.state, n, codeLen)
	}
	return hd
}

func hintPositional(state *model.GameState, n int, codeLen int) *model.HintData {
	if n >= codeLen {
		return nil
	}
	state.HintsGiven++
	slots := make([]int, codeLen)
	for i := 0; i <= n; i++ {
		slots[i] = state.SecretCode[i]
	}
	var parts []string
	for i := 0; i <= n; i++ {
		parts = append(parts, fmt.Sprintf("Position %d = %d", i+1, state.SecretCode[i]))
	}
	return &model.HintData{
		Number: state.HintsGiven,
		Text:   strings.Join(parts, ", "),
		Slots:  slots,
	}
}

func hintColorFirst(state *model.GameState, n int, codeLen int) *model.HintData {
	shuffled := state.ShuffledCode
	if n < codeLen {
		state.HintsGiven++
		onlyColors := make([]int, n+1)
		copy(onlyColors, shuffled[:n+1])
		return &model.HintData{
			Number:     state.HintsGiven,
			Text:       "Colors " + intSliceToString(onlyColors) + " are used",
			OnlyColors: onlyColors,
		}
	}
	offset := n - codeLen
	if offset >= codeLen {
		return nil
	}
	state.HintsGiven++
	slots := make([]int, codeLen)
	for i := 0; i <= offset; i++ {
		slots[i] = state.SecretCode[i]
	}
	// Unpositioned: start with full shuffled set, remove one
	// instance for each positioned color
	remaining := make([]int, codeLen)
	copy(remaining, shuffled)
	for i := 0; i <= offset; i++ {
		c := state.SecretCode[i]
		for j, r := range remaining {
			if r == c {
				remaining = append(remaining[:j], remaining[j+1:]...)
				break
			}
		}
	}
	colorPart := "Colors " + intSliceToString(shuffled) + " are used"
	var posParts []string
	for i := 0; i <= offset; i++ {
		posParts = append(posParts, fmt.Sprintf("Pos %d = %d", i+1, state.SecretCode[i]))
	}
	return &model.HintData{
		Number:     state.HintsGiven,
		Text:       colorPart + " | " + strings.Join(posParts, ", "),
		Slots:      slots,
		OnlyColors: remaining,
	}
}

func intSliceToString(s []int) string {
	result := ""
	for i, n := range s {
		if i > 0 {
			result += ","
		}
		result += strconv.Itoa(n)
	}
	return result
}
