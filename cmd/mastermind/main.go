package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Zappo-II/mastermind/internal/config"
	"github.com/Zappo-II/mastermind/internal/game"
	"github.com/Zappo-II/mastermind/internal/model"
	ui "github.com/Zappo-II/mastermind/internal/ui"
)

func main() {
	levelPath := flag.String("level", "default-level.yaml", "Path to level config")
	palettePath := flag.String("palette", "default-palette.yaml", "Path to palette config")
	flag.Parse()

	cfg, err := config.Load(*palettePath, *levelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	if err := ui.EnableRawMode(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		os.Exit(1)
	}
	defer ui.DisableRawMode()

	// Restore terminal on signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		ui.DisableRawMode()
		fmt.Print("\r\n")
		os.Exit(0)
	}()

	// Intro screen
	fmt.Print(ui.RenderIntroScreen(cfg.Level))
	_, _ = ui.ReadByte()

	maxWidth := ui.ComputeMaxWidth(cfg.Level)
	g := game.NewGame(cfg.Level, cfg.Palette)

	for {
		state := g.State()
		if state.Won {
			renderWinScreen(state, maxWidth)
			break
		}
		if state.Lost {
			renderLoseScreen(state, maxWidth)
			break
		}

		guess := collectGuess(state, g, maxWidth)
		if guess == nil {
			// Check if game ended due to hint cost
			state = g.State()
			if state.Lost {
				renderLoseScreen(state, maxWidth)
			}
			break
		}

		g.MakeGuess(guess)
	}
}

func inputPrompt(guess []int, pendingDigit int, codeLength int, multiDigit bool, hintAvailable bool) string {
	if len(guess) >= codeLength {
		return "Enter to confirm, Bksp to undo: "
	}
	if multiDigit && pendingDigit >= 0 {
		return fmt.Sprintf("%d_ (digit/Enter/Space): ", pendingDigit)
	}
	if hintAvailable {
		return "Color# (? help, h hint): "
	}
	return "Color# (? help): "
}

func collectGuess(state *model.GameState, g *game.Game, maxWidth int) []int {
	guess := make([]int, 0, state.Level.CodeLength)
	maxColors := state.Level.MaxColors
	multiDigit := maxColors >= 10
	pendingDigit := -1

	for {
		hintAvailable := g.HintAvailable()
		if hintAvailable {
			nextCost := g.HintCost()
			if nextCost > 0 && state.Level.MaxTurns > 0 {
				turnsLeft := state.Level.MaxTurns - g.State().CurrentTurn
				if nextCost > turnsLeft {
					hintAvailable = false
				}
			}
		}
		prompt := inputPrompt(guess, pendingDigit, state.Level.CodeLength, multiDigit, hintAvailable)
		drawGameWithGuess(state, guess, maxWidth, prompt)

		b, err := ui.ReadByte()
		if err != nil {
			return nil
		}

		// Quit
		if b == 'q' {
			ui.DisableRawMode()
			fmt.Print("\r\nQuitting...\r\n")
			os.Exit(0)
		}

		// Help
		if b == '?' {
			fmt.Print("\r\n")
			fmt.Print(ui.RenderHelp())
			fmt.Print("Press any key to continue...")
			_, _ = ui.ReadByte()
			pendingDigit = -1
			continue
		}

		// Hint
		if b == 'h' {
			if !state.Level.Hint.Enabled {
				fmt.Print("\r\nHints are disabled. Press any key...")
				_, _ = ui.ReadByte()
				continue
			}
			if !g.HintAvailable() {
				fmt.Print("\r\nNo more hints available. Press any key...")
				_, _ = ui.ReadByte()
				continue
			}
			nextCost := g.HintCost()
			if nextCost > 0 && state.Level.MaxTurns > 0 {
				turnsLeft := state.Level.MaxTurns - g.State().CurrentTurn
				if nextCost > turnsLeft {
					fmt.Printf("\r\nHint would cost %d turns, only %d left. Press any key...", nextCost, turnsLeft)
					_, _ = ui.ReadByte()
					continue
				}
			}
			if nextCost > 0 {
				fmt.Printf("\r\nHint costs %d turn(s). Enter to confirm, any other key to cancel...", nextCost)
				cb, _ := ui.ReadByte()
				if cb != '\r' && cb != '\n' {
					continue
				}
			}
			hd := g.GetHint()
			if hd == nil {
				continue
			}
			hd.Cost = nextCost
			g.ApplyHint(hd)
			state = g.State()
			if state.Lost {
				return nil
			}
			pendingDigit = -1
			continue
		}

		// Backspace / DEL
		if b == 127 || b == 8 {
			if pendingDigit >= 0 {
				pendingDigit = -1
			} else if len(guess) > 0 {
				guess = guess[:len(guess)-1]
			}
			continue
		}

		// Enter: confirm pending digit, or submit complete row
		if b == '\r' || b == '\n' {
			if pendingDigit >= 0 {
				if pendingDigit >= 1 && pendingDigit <= maxColors && len(guess) < state.Level.CodeLength {
					guess = append(guess, pendingDigit)
				}
				pendingDigit = -1
				continue
			}
			if len(guess) >= state.Level.CodeLength {
				return guess
			}
			continue
		}

		// Space: confirm pending digit only (does not confirm row)
		if b == ' ' {
			if pendingDigit >= 0 {
				if pendingDigit >= 1 && pendingDigit <= maxColors && len(guess) < state.Level.CodeLength {
					guess = append(guess, pendingDigit)
				}
				pendingDigit = -1
			}
			continue
		}

		// Digit input (never auto-confirms row — Enter always required)
		if b >= '0' && b <= '9' {
			if len(guess) >= state.Level.CodeLength {
				continue
			}
			d := int(b - '0')

			if multiDigit {
				if pendingDigit >= 0 {
					// Second digit: form two-digit number
					twoDigit := pendingDigit*10 + d
					if twoDigit >= 1 && twoDigit <= maxColors {
						guess = append(guess, twoDigit)
						pendingDigit = -1
					} else {
						// Invalid two-digit: treat second digit as new first digit
						if d >= 1 && d <= maxColors && d*10 > maxColors {
							guess = append(guess, d)
							pendingDigit = -1
						} else {
							pendingDigit = d
						}
					}
				} else if d >= 1 && d <= maxColors && d*10 > maxColors {
					// Unambiguous single digit — auto-accept
					guess = append(guess, d)
				} else {
					pendingDigit = d
				}
			} else {
				if d >= 1 && d <= maxColors {
					guess = append(guess, d)
				}
			}
			continue
		}
	}
}

func renderWinScreen(state *model.GameState, maxWidth int) {
	fmt.Print(ui.ClearScreen())
	fmt.Print(ui.RenderRulesHeader(state, maxWidth))
	fmt.Print(ui.RenderColorPalette(state, maxWidth))
	fmt.Print(ui.RenderSecretShield(state, maxWidth))
	fmt.Print(ui.RenderTurnRows(state, nil, maxWidth))
	fmt.Print(ui.RenderWinMessage(maxWidth))
}

func renderLoseScreen(state *model.GameState, maxWidth int) {
	fmt.Print(ui.ClearScreen())
	fmt.Print(ui.RenderRulesHeader(state, maxWidth))
	fmt.Print(ui.RenderColorPalette(state, maxWidth))
	fmt.Print(ui.RenderSecretShield(state, maxWidth))
	fmt.Print(ui.RenderTurnRows(state, nil, maxWidth))
	fmt.Print(ui.RenderLoseMessage(maxWidth))
}

func drawGameWithGuess(state *model.GameState, guess []int, maxWidth int, prompt string) {
	fmt.Print(ui.ClearScreen())
	fmt.Print(ui.RenderRulesHeader(state, maxWidth))
	fmt.Print(ui.RenderColorPalette(state, maxWidth))
	fmt.Print(ui.RenderSecretShield(state, maxWidth))
	fmt.Print(ui.RenderTurnRows(state, guess, maxWidth))
	fmt.Print(prompt)
}
