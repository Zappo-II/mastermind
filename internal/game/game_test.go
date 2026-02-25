package game

import (
	"reflect"
	"testing"

	"github.com/Zappo-II/mastermind/internal/model"
)

// --------------- CalculateFeedback ---------------

func TestCalculateFeedback(t *testing.T) {
	tests := []struct {
		name    string
		guess   []int
		secret  []int
		ordered bool
		want    model.Feedback
	}{
		{
			name:   "all exact",
			guess:  []int{1, 2, 3, 4},
			secret: []int{1, 2, 3, 4},
			want:   model.Feedback{Exact: 4, Partial: 0},
		},
		{
			name:   "all partial",
			guess:  []int{4, 3, 2, 1},
			secret: []int{1, 2, 3, 4},
			want:   model.Feedback{Exact: 0, Partial: 4},
		},
		{
			name:   "no matches",
			guess:  []int{5, 6, 7, 8},
			secret: []int{1, 2, 3, 4},
			want:   model.Feedback{Exact: 0, Partial: 0},
		},
		{
			name:   "mix exact and partial",
			guess:  []int{1, 3, 2, 4},
			secret: []int{1, 2, 3, 4},
			want:   model.Feedback{Exact: 2, Partial: 2},
		},
		{
			name:   "duplicate in guess not in secret",
			guess:  []int{1, 1, 2, 3},
			secret: []int{1, 4, 5, 6},
			want:   model.Feedback{Exact: 1, Partial: 0},
		},
		{
			name:   "duplicate in secret",
			guess:  []int{1, 2, 3, 4},
			secret: []int{1, 1, 1, 1},
			want:   model.Feedback{Exact: 1, Partial: 0},
		},
		{
			name:   "duplicate in both partial",
			guess:  []int{1, 1, 2, 2},
			secret: []int{2, 2, 1, 1},
			want:   model.Feedback{Exact: 0, Partial: 4},
		},
		{
			name:   "single element exact",
			guess:  []int{3},
			secret: []int{3},
			want:   model.Feedback{Exact: 1, Partial: 0},
		},
		{
			name:   "single element no match",
			guess:  []int{3},
			secret: []int{5},
			want:   model.Feedback{Exact: 0, Partial: 0},
		},
		{
			name:    "ordered all exact",
			guess:   []int{1, 2, 3},
			secret:  []int{1, 2, 3},
			ordered: true,
			want: model.Feedback{
				Exact: 3, Partial: 0,
				Positions: []model.FeedbackPeg{model.FeedbackExact, model.FeedbackExact, model.FeedbackExact},
			},
		},
		{
			name:    "ordered mixed",
			guess:   []int{1, 3, 2},
			secret:  []int{1, 2, 3},
			ordered: true,
			want: model.Feedback{
				Exact: 1, Partial: 2,
				Positions: []model.FeedbackPeg{model.FeedbackExact, model.FeedbackPartial, model.FeedbackPartial},
			},
		},
		{
			name:    "ordered no matches",
			guess:   []int{7, 8, 9},
			secret:  []int{1, 2, 3},
			ordered: true,
			want: model.Feedback{
				Exact: 0, Partial: 0,
				Positions: []model.FeedbackPeg{model.FeedbackNone, model.FeedbackNone, model.FeedbackNone},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateFeedback(tc.guess, tc.secret, tc.ordered)
			if got.Exact != tc.want.Exact || got.Partial != tc.want.Partial {
				t.Errorf("got %d exact %d partial, want %d exact %d partial",
					got.Exact, got.Partial, tc.want.Exact, tc.want.Partial)
			}
			if tc.ordered {
				if !reflect.DeepEqual(got.Positions, tc.want.Positions) {
					t.Errorf("positions = %v, want %v", got.Positions, tc.want.Positions)
				}
			} else {
				if got.Positions != nil {
					t.Errorf("positions should be nil for unordered, got %v", got.Positions)
				}
			}
		})
	}
}

// --------------- HintCost ---------------

func TestHintCost(t *testing.T) {
	tests := []struct {
		name       string
		cost       int
		costFactor int
		hintsGiven int
		want       int
	}{
		{"flat cost factor=1", 2, 1, 0, 2},
		{"flat cost factor=1 after 3 hints", 2, 1, 3, 2},
		{"exponential factor=2 first hint", 1, 2, 0, 1},
		{"exponential factor=2 second hint", 1, 2, 1, 2},
		{"exponential factor=2 third hint", 1, 2, 2, 4},
		{"exponential factor=2 fourth hint", 1, 2, 3, 8},
		{"free hints cost=0", 0, 1, 5, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{
				state: model.GameState{
					HintsGiven: tc.hintsGiven,
					Level: model.LevelConfig{
						Hint: model.HintConfig{
							Cost:       tc.cost,
							CostFactor: tc.costFactor,
						},
					},
				},
			}
			if got := g.HintCost(); got != tc.want {
				t.Errorf("HintCost() = %d, want %d", got, tc.want)
			}
		})
	}
}

// --------------- HintAvailable ---------------

func TestHintAvailable(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		maxHints   int
		hintsGiven int
		codeLength int
		sortByOrder bool
		want       bool
	}{
		{"disabled", false, 0, 0, 4, false, false},
		{"maxHints reached", true, 3, 3, 4, false, false},
		{"colors-first mode available", true, 0, 0, 4, false, true},
		{"colors-first mode at limit", true, 0, 8, 4, false, false},
		{"positional mode available", true, 0, 0, 4, true, true},
		{"positional mode at limit", true, 0, 4, 4, true, false},
		{"colors-first mode halfway", true, 0, 4, 4, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{
				state: model.GameState{
					HintsGiven: tc.hintsGiven,
					Level: model.LevelConfig{
						CodeLength: tc.codeLength,
						Hint: model.HintConfig{
							Enabled:     tc.enabled,
							MaxHints:    tc.maxHints,
							SortByOrder: tc.sortByOrder,
						},
					},
				},
			}
			if got := g.HintAvailable(); got != tc.want {
				t.Errorf("HintAvailable() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --------------- MakeGuess ---------------

func TestMakeGuess_Win(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{1, 2, 3, 4},
			Turns:      []model.Turn{},
			Level:      model.LevelConfig{MaxTurns: 10, CodeLength: 4},
		},
	}

	fb := g.MakeGuess([]int{1, 2, 3, 4})
	if fb.Exact != 4 {
		t.Fatalf("expected 4 exact, got %d", fb.Exact)
	}
	if !g.state.Won {
		t.Error("expected Won=true")
	}
	if g.state.Lost {
		t.Error("expected Lost=false")
	}
}

func TestMakeGuess_Lose(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode:  []int{1, 2, 3, 4},
			Turns:       []model.Turn{},
			CurrentTurn: 0,
			Level:       model.LevelConfig{MaxTurns: 1, CodeLength: 4},
		},
	}

	// MaxTurns validation requires >=8, but game logic just checks the field.
	// We set MaxTurns=1 directly for testing the lose condition.
	g.MakeGuess([]int{5, 6, 7, 8})
	if !g.state.Lost {
		t.Error("expected Lost=true after reaching MaxTurns")
	}
	if g.state.Won {
		t.Error("expected Won=false")
	}
}

func TestMakeGuess_UnlimitedNeverLoses(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{1, 2},
			Turns:      []model.Turn{},
			Level:      model.LevelConfig{MaxTurns: 0, CodeLength: 2},
		},
	}

	for i := 0; i < 100; i++ {
		g.MakeGuess([]int{5, 6})
	}
	if g.state.Lost {
		t.Error("unlimited mode should never set Lost=true")
	}
}

func TestMakeGuess_TurnCounter(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{1, 2, 3},
			Turns:      []model.Turn{},
			Level:      model.LevelConfig{MaxTurns: 0, CodeLength: 3},
		},
	}

	g.MakeGuess([]int{5, 6, 7})
	g.MakeGuess([]int{5, 6, 7})
	g.MakeGuess([]int{5, 6, 7})

	if g.state.CurrentTurn != 3 {
		t.Errorf("CurrentTurn = %d, want 3", g.state.CurrentTurn)
	}
	if len(g.state.Turns) != 3 {
		t.Errorf("len(Turns) = %d, want 3", len(g.state.Turns))
	}
}

// --------------- GetSecretColors / GetUsedColorsInOrder ---------------

func TestGetSecretColors(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{3, 1, 3, 2, 1},
		},
	}
	got := g.GetSecretColors()
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetSecretColors() = %v, want %v", got, want)
	}
}

func TestGetUsedColorsInOrder(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{3, 1, 3, 2, 1},
		},
	}
	got := g.GetUsedColorsInOrder()
	want := []int{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetUsedColorsInOrder() = %v, want %v", got, want)
	}
}

// --------------- ApplyHint ---------------

func TestApplyHint_ZeroCost(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{1, 2, 3},
			Turns:      []model.Turn{},
			Level:      model.LevelConfig{MaxTurns: 10, CodeLength: 3},
		},
	}

	hd := &model.HintData{Number: 1, Cost: 0, Text: "hint"}
	g.ApplyHint(hd)

	if g.state.CurrentTurn != 0 {
		t.Errorf("zero-cost hint should not increment CurrentTurn, got %d", g.state.CurrentTurn)
	}
	if len(g.state.Turns) != 1 {
		t.Errorf("should have 1 turn row, got %d", len(g.state.Turns))
	}
	if !g.state.Turns[0].IsHint {
		t.Error("turn should be marked as hint")
	}
}

func TestApplyHint_MultiTurnCost(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode: []int{1, 2, 3},
			Turns:      []model.Turn{},
			Level:      model.LevelConfig{MaxTurns: 20, CodeLength: 3},
		},
	}

	hd := &model.HintData{Number: 1, Cost: 3, Text: "hint"}
	g.ApplyHint(hd)

	if g.state.CurrentTurn != 3 {
		t.Errorf("CurrentTurn = %d, want 3", g.state.CurrentTurn)
	}
	// 1 display row + 2 cost-only rows
	if len(g.state.Turns) != 3 {
		t.Errorf("len(Turns) = %d, want 3", len(g.state.Turns))
	}
}

func TestApplyHint_TriggersLoss(t *testing.T) {
	g := &Game{
		state: model.GameState{
			SecretCode:  []int{1, 2, 3},
			Turns:       []model.Turn{},
			CurrentTurn: 8,
			Level:       model.LevelConfig{MaxTurns: 10, CodeLength: 3},
		},
	}

	hd := &model.HintData{Number: 1, Cost: 2, Text: "hint"}
	g.ApplyHint(hd)

	if !g.state.Lost {
		t.Error("hint cost should trigger loss when reaching MaxTurns")
	}
}
