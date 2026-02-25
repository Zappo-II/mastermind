package model

import "fmt"

type Color struct {
	Number int    `yaml:"number"`
	Name   string `yaml:"name"`
	RGB    string `yaml:"rgb"`
	FG     string `yaml:"fg"`
}

func (c Color) String() string {
	return fmt.Sprintf("Color{Number:%d Name:%s RGB:%s FG:%s}", c.Number, c.Name, c.RGB, c.FG)
}

type FeedbackConfig struct {
	ColorNumber int    `yaml:"colorNumber"`
	Symbol      string `yaml:"symbol"`
}

type FeedbackSettings struct {
	Exact   FeedbackConfig `yaml:"exact"`
	Partial FeedbackConfig `yaml:"partial"`
	Empty   FeedbackConfig `yaml:"empty"`
}

type PaletteConfig struct {
	Colors   []Color          `yaml:"colors"`
	Feedback FeedbackSettings `yaml:"feedback"`
}

func (p PaletteConfig) GetColor(num int) *Color {
	for _, c := range p.Colors {
		if c.Number == num {
			return &c
		}
	}
	return nil
}

type HintConfig struct {
	Enabled     bool `yaml:"enabled"`
	Cost        int  `yaml:"cost"`
	CostFactor  int  `yaml:"costFactor"`
	MaxHints    int  `yaml:"maxHints"`
	SortByOrder bool `yaml:"sortByOrder"`
}

type HintData struct {
	Number     int    // 1-based hint number
	Cost       int    // actual turn cost of this hint
	Text       string // accumulated hint text
	Slots      []int  // len == codeLength; 0=unknown, >0=revealed color at position
	OnlyColors []int  // colors known but not yet positioned
}

type LevelConfig struct {
	CodeLength      int        `yaml:"codeLength"`
	MaxColors       int        `yaml:"maxColors"`
	Duplicates      bool       `yaml:"duplicates"`
	MaxTurns        int        `yaml:"maxTurns"`
	OrderedFeedback bool       `yaml:"orderedFeedback"`
	Hint            HintConfig `yaml:"hint"`
}

type FeedbackPeg int

const (
	FeedbackNone FeedbackPeg = iota
	FeedbackExact
	FeedbackPartial
)

type Feedback struct {
	Exact     int
	Partial   int
	Positions []FeedbackPeg // nil for unordered, populated for ordered
}

type Turn struct {
	Guess    []int
	Feedback Feedback
	IsHint   bool
	HintData *HintData // non-nil on hint display rows, nil on cost-only rows
}

type GameState struct {
	SecretCode   []int
	Turns        []Turn
	CurrentTurn  int
	Won          bool
	Lost         bool
	HintsGiven   int
	ShuffledCode []int // shuffled secret for color hint reveal order, initialized on first hint
	Level        LevelConfig
	Palette      PaletteConfig
}

func (f Feedback) String() string {
	return fmt.Sprintf("%d exact, %d partial", f.Exact, f.Partial)
}

func (g *GameState) GetMaxTurns() int {
	return g.Level.MaxTurns
}

func (g *GameState) IsUnlimited() bool {
	return g.Level.MaxTurns == 0
}

func (g *GameState) CurrentTurnNumber() int {
	return len(g.Turns) + 1
}
