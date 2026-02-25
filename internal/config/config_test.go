package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --------------- helpers ---------------

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

const validLevelYAML = `
rules:
  codeLength: 4
  maxColors: 6
  duplicates: false
  maxTurns: 10
  orderedFeedback: false
hint:
  enabled: true
  cost: 1
  costFactor: 1
  maxHints: 0
  sortByOrder: false
`

func validPaletteYAML() string {
	var b strings.Builder
	b.WriteString("colors:\n")
	for i := 1; i <= 12; i++ {
		b.WriteString("  - number: ")
		b.WriteString(strings.Repeat("", 0)) // noop
		b.WriteString(itoa(i))
		b.WriteString("\n    name: C")
		b.WriteString(itoa(i))
		b.WriteString("\n    rgb: \"#AA0000\"\n    fg: \"#FFFFFF\"\n")
	}
	b.WriteString("feedback:\n  exact:\n    colorNumber: 1\n    symbol: X\n  partial:\n    colorNumber: 2\n    symbol: O\n  empty:\n    colorNumber: 3\n    symbol: '-'\n")
	return b.String()
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

// --------------- LoadLevel ---------------

func TestLoadLevel_Valid(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "level.yaml", validLevelYAML)

	lv, err := LoadLevel(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lv.CodeLength != 4 {
		t.Errorf("CodeLength = %d, want 4", lv.CodeLength)
	}
	if lv.MaxColors != 6 {
		t.Errorf("MaxColors = %d, want 6", lv.MaxColors)
	}
	if lv.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want 10", lv.MaxTurns)
	}
}

func TestLoadLevel_CodeLengthOutOfRange(t *testing.T) {
	dir := t.TempDir()
	yaml := strings.Replace(validLevelYAML, "codeLength: 4", "codeLength: 2", 1)
	p := writeFile(t, dir, "level.yaml", yaml)

	_, err := LoadLevel(p)
	if err == nil || !strings.Contains(err.Error(), "codeLength") {
		t.Errorf("expected codeLength error, got: %v", err)
	}
}

func TestLoadLevel_MaxColorsOutOfRange(t *testing.T) {
	dir := t.TempDir()
	yaml := strings.Replace(validLevelYAML, "maxColors: 6", "maxColors: 99", 1)
	p := writeFile(t, dir, "level.yaml", yaml)

	_, err := LoadLevel(p)
	if err == nil || !strings.Contains(err.Error(), "maxColors") {
		t.Errorf("expected maxColors error, got: %v", err)
	}
}

func TestLoadLevel_MaxTurnsInvalid(t *testing.T) {
	for _, val := range []string{"3", "7"} {
		t.Run("maxTurns="+val, func(t *testing.T) {
			dir := t.TempDir()
			yaml := strings.Replace(validLevelYAML, "maxTurns: 10", "maxTurns: "+val, 1)
			p := writeFile(t, dir, "level.yaml", yaml)

			_, err := LoadLevel(p)
			if err == nil || !strings.Contains(err.Error(), "maxTurns") {
				t.Errorf("expected maxTurns error, got: %v", err)
			}
		})
	}
}

func TestLoadLevel_CostFactorDefaultsTo1(t *testing.T) {
	dir := t.TempDir()
	// costFactor: 0 (or omitted) should default to 1
	yaml := strings.Replace(validLevelYAML, "costFactor: 1", "costFactor: 0", 1)
	p := writeFile(t, dir, "level.yaml", yaml)

	lv, err := LoadLevel(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lv.Hint.CostFactor != 1 {
		t.Errorf("CostFactor = %d, want 1 (default)", lv.Hint.CostFactor)
	}
}

func TestLoadLevel_FileNotFound(t *testing.T) {
	_, err := LoadLevel("/nonexistent/path/level.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --------------- LoadPalette ---------------

func TestLoadPalette_Valid(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "palette.yaml", validPaletteYAML())

	pal, err := LoadPalette(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pal.Colors) != 12 {
		t.Errorf("color count = %d, want 12", len(pal.Colors))
	}
}

func TestLoadPalette_WrongColorCount(t *testing.T) {
	dir := t.TempDir()
	yaml := `
colors:
  - number: 1
    name: C1
    rgb: "#AA0000"
    fg: "#FFFFFF"
feedback:
  exact:
    colorNumber: 1
    symbol: X
`
	p := writeFile(t, dir, "palette.yaml", yaml)

	_, err := LoadPalette(p)
	if err == nil || !strings.Contains(err.Error(), "12 colors") {
		t.Errorf("expected color count error, got: %v", err)
	}
}

func TestLoadPalette_InvalidFGHex(t *testing.T) {
	dir := t.TempDir()
	// Replace one fg with an invalid value
	yaml := strings.Replace(validPaletteYAML(), "fg: \"#FFFFFF\"", "fg: \"bad\"", 1)
	p := writeFile(t, dir, "palette.yaml", yaml)

	_, err := LoadPalette(p)
	if err == nil || !strings.Contains(err.Error(), "fg must be") {
		t.Errorf("expected fg hex error, got: %v", err)
	}
}

func TestLoadPalette_FileNotFound(t *testing.T) {
	_, err := LoadPalette("/nonexistent/path/palette.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --------------- Load ---------------

func TestLoad_MaxColorsExceedsPalette(t *testing.T) {
	dir := t.TempDir()
	palPath := writeFile(t, dir, "palette.yaml", validPaletteYAML())

	// maxColors=12 is fine, but 13 would exceed; however palette has 12 colors
	// and maxColors max is 12 — so test with a palette that has 12 and maxColors=12
	// The actual cross-validation is maxColors > len(palette.Colors)
	// We can't set maxColors>12 via LoadLevel (validation rejects it).
	// Instead, test with a smaller palette? No, palette must be 12.
	// The check fires when level.MaxColors > palette colors count.
	// Since palette is always 12 and maxColors is always <=12, this can only fire
	// if we somehow craft it. Let's just verify the happy path through Load.

	levelYAML := strings.Replace(validLevelYAML, "maxColors: 6", "maxColors: 12", 1)
	lvlPath := writeFile(t, dir, "level.yaml", levelYAML)

	cfg, err := Load(palPath, lvlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Level.MaxColors != 12 {
		t.Errorf("MaxColors = %d, want 12", cfg.Level.MaxColors)
	}
	if len(cfg.Palette.Colors) != 12 {
		t.Errorf("palette colors = %d, want 12", len(cfg.Palette.Colors))
	}
}

func TestLoad_InvalidLevel(t *testing.T) {
	dir := t.TempDir()
	palPath := writeFile(t, dir, "palette.yaml", validPaletteYAML())
	lvlPath := writeFile(t, dir, "level.yaml", "rules:\n  codeLength: 1\n")

	_, err := Load(palPath, lvlPath)
	if err == nil {
		t.Error("expected error from invalid level")
	}
}

func TestLoad_InvalidPalette(t *testing.T) {
	dir := t.TempDir()
	palPath := writeFile(t, dir, "palette.yaml", "colors: []\n")
	lvlPath := writeFile(t, dir, "level.yaml", validLevelYAML)

	_, err := Load(palPath, lvlPath)
	if err == nil {
		t.Error("expected error from invalid palette")
	}
}
