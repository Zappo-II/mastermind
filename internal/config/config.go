package config

import (
	"fmt"
	"os"

	"github.com/Zappo-II/mastermind/internal/model"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Palette model.PaletteConfig
	Level   model.LevelConfig
}

func LoadPalette(path string) (model.PaletteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.PaletteConfig{}, fmt.Errorf("failed to read palette file: %w", err)
	}

	var palette model.PaletteConfig
	if err := yaml.Unmarshal(data, &palette); err != nil {
		return model.PaletteConfig{}, fmt.Errorf("failed to parse palette YAML: %w", err)
	}

	if len(palette.Colors) != 12 {
		return model.PaletteConfig{}, fmt.Errorf("palette must have exactly 12 colors, got %d", len(palette.Colors))
	}

	for _, c := range palette.Colors {
		if c.Number < 1 || c.Number > 12 {
			return model.PaletteConfig{}, fmt.Errorf("color number must be 1-12, got %d", c.Number)
		}
		if c.FG == "" || len(c.FG) != 7 || c.FG[0] != '#' {
			return model.PaletteConfig{}, fmt.Errorf("color %d: fg must be a valid #RRGGBB hex color, got %q", c.Number, c.FG)
		}
	}

	return palette, nil
}

type levelFileConfig struct {
	Rules struct {
		CodeLength      int  `yaml:"codeLength"`
		MaxColors       int  `yaml:"maxColors"`
		Duplicates      bool `yaml:"duplicates"`
		MaxTurns        int  `yaml:"maxTurns"`
		OrderedFeedback bool `yaml:"orderedFeedback"`
	} `yaml:"rules"`
	Hint model.HintConfig `yaml:"hint"`
}

func LoadLevel(path string) (model.LevelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.LevelConfig{}, fmt.Errorf("failed to read level file: %w", err)
	}

	var file levelFileConfig
	if err := yaml.Unmarshal(data, &file); err != nil {
		return model.LevelConfig{}, fmt.Errorf("failed to parse level YAML: %w", err)
	}

	level := model.LevelConfig{
		CodeLength:      file.Rules.CodeLength,
		MaxColors:       file.Rules.MaxColors,
		Duplicates:      file.Rules.Duplicates,
		MaxTurns:        file.Rules.MaxTurns,
		OrderedFeedback: file.Rules.OrderedFeedback,
		Hint:            file.Hint,
	}

	if level.CodeLength < 4 || level.CodeLength > 12 {
		return model.LevelConfig{}, fmt.Errorf("codeLength must be 4-12, got %d", level.CodeLength)
	}

	if level.MaxColors < 4 || level.MaxColors > 12 {
		return model.LevelConfig{}, fmt.Errorf("maxColors must be 4-12, got %d", level.MaxColors)
	}

	if level.MaxTurns < 0 || (level.MaxTurns > 0 && level.MaxTurns < 8) {
		return model.LevelConfig{}, fmt.Errorf("maxTurns must be 0 or >= 8, got %d", level.MaxTurns)
	}

	if level.Hint.Cost < 0 || level.Hint.Cost > 9 {
		return model.LevelConfig{}, fmt.Errorf("hint.cost must be 0-9, got %d", level.Hint.Cost)
	}

	// Default costFactor to 1 (flat cost) if omitted
	if level.Hint.CostFactor == 0 {
		level.Hint.CostFactor = 1
	}
	if level.Hint.CostFactor < 1 || level.Hint.CostFactor > 9 {
		return model.LevelConfig{}, fmt.Errorf("hint.costFactor must be 1-9, got %d", level.Hint.CostFactor)
	}

	if level.Hint.MaxHints < 0 {
		return model.LevelConfig{}, fmt.Errorf("hint.maxHints must be >= 0, got %d", level.Hint.MaxHints)
	}

	return level, nil
}

func Load(palettePath, levelPath string) (Config, error) {
	palette, err := LoadPalette(palettePath)
	if err != nil {
		return Config{}, err
	}

	level, err := LoadLevel(levelPath)
	if err != nil {
		return Config{}, err
	}

	if level.MaxColors > len(palette.Colors) {
		return Config{}, fmt.Errorf("maxColors %d exceeds palette size %d", level.MaxColors, len(palette.Colors))
	}

	return Config{
		Palette: palette,
		Level:   level,
	}, nil
}
