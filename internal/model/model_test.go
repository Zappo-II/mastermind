package model

import "testing"

func TestGetColor_Found(t *testing.T) {
	p := PaletteConfig{
		Colors: []Color{
			{Number: 1, Name: "RED", RGB: "#FF0000", FG: "#FFFFFF"},
			{Number: 2, Name: "GREEN", RGB: "#00FF00", FG: "#000000"},
		},
	}

	c := p.GetColor(2)
	if c == nil {
		t.Fatal("expected color, got nil")
	}
	if c.Name != "GREEN" {
		t.Errorf("Name = %q, want GREEN", c.Name)
	}
}

func TestGetColor_NotFound(t *testing.T) {
	p := PaletteConfig{
		Colors: []Color{
			{Number: 1, Name: "RED", RGB: "#FF0000", FG: "#FFFFFF"},
		},
	}

	c := p.GetColor(99)
	if c != nil {
		t.Errorf("expected nil, got %v", c)
	}
}

func TestIsUnlimited(t *testing.T) {
	tests := []struct {
		name     string
		maxTurns int
		want     bool
	}{
		{"zero means unlimited", 0, true},
		{"positive means limited", 10, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GameState{Level: LevelConfig{MaxTurns: tc.maxTurns}}
			if got := g.IsUnlimited(); got != tc.want {
				t.Errorf("IsUnlimited() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCurrentTurnNumber(t *testing.T) {
	tests := []struct {
		name  string
		turns []Turn
		want  int
	}{
		{"no turns", nil, 1},
		{"empty slice", []Turn{}, 1},
		{"two turns", []Turn{{}, {}}, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &GameState{Turns: tc.turns}
			if got := g.CurrentTurnNumber(); got != tc.want {
				t.Errorf("CurrentTurnNumber() = %d, want %d", got, tc.want)
			}
		})
	}
}
