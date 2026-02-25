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

// inputAction signals to the main loop what to do after a key handler runs.
type inputAction int

const (
	actionContinue inputAction = iota // re-draw and read next key
	actionSubmit                      // return the guess
	actionAbort                       // return nil (quit or lost)
)

// inputState bundles the mutable state that key handlers read/write.
type inputState struct {
	guess        []int
	pendingDigit int
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
	is := &inputState{
		guess:        make([]int, 0, state.Level.CodeLength),
		pendingDigit: -1,
	}
	maxColors := state.Level.MaxColors
	multiDigit := maxColors >= 10

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
		prompt := inputPrompt(is.guess, is.pendingDigit, state.Level.CodeLength, multiDigit, hintAvailable)
		drawGameWithGuess(state, is.guess, maxWidth, prompt)

		b, err := ui.ReadByte()
		if err != nil {
			return nil
		}

		var act inputAction
		switch {
		case b == 'q':
			ui.DisableRawMode()
			fmt.Print("\r\nQuitting...\r\n")
			os.Exit(0)
		case b == '?':
			act = handleHelp(is)
		case b == 'h':
			act, state = handleHint(is, state, g)
		case b == 127 || b == 8:
			act = handleBackspace(is)
		case b == '\r' || b == '\n':
			act = handleEnter(is, maxColors, state.Level.CodeLength)
		case b == ' ':
			act = handleSpace(is, maxColors, state.Level.CodeLength)
		case b >= '0' && b <= '9':
			act = handleDigit(is, b, maxColors, state.Level.CodeLength, multiDigit)
		default:
			act = actionContinue
		}

		switch act {
		case actionSubmit:
			return is.guess
		case actionAbort:
			return nil
		}
	}
}

func handleHelp(is *inputState) inputAction {
	fmt.Print("\r\n")
	fmt.Print(ui.RenderHelp())
	fmt.Print("Press any key to continue...")
	_, _ = ui.ReadByte()
	is.pendingDigit = -1
	return actionContinue
}

func handleHint(is *inputState, state *model.GameState, g *game.Game) (inputAction, *model.GameState) {
	if !state.Level.Hint.Enabled {
		fmt.Print("\r\nHints are disabled. Press any key...")
		_, _ = ui.ReadByte()
		return actionContinue, state
	}
	if !g.HintAvailable() {
		fmt.Print("\r\nNo more hints available. Press any key...")
		_, _ = ui.ReadByte()
		return actionContinue, state
	}
	nextCost := g.HintCost()
	if nextCost > 0 && state.Level.MaxTurns > 0 {
		turnsLeft := state.Level.MaxTurns - g.State().CurrentTurn
		if nextCost > turnsLeft {
			fmt.Printf("\r\nHint would cost %d turns, only %d left. Press any key...", nextCost, turnsLeft)
			_, _ = ui.ReadByte()
			return actionContinue, state
		}
	}
	if nextCost > 0 {
		fmt.Printf("\r\nHint costs %d turn(s). Enter to confirm, any other key to cancel...", nextCost)
		cb, _ := ui.ReadByte()
		if cb != '\r' && cb != '\n' {
			return actionContinue, state
		}
	}
	hd := g.GetHint()
	if hd == nil {
		return actionContinue, state
	}
	hd.Cost = nextCost
	g.ApplyHint(hd)
	state = g.State()
	if state.Lost {
		return actionAbort, state
	}
	is.pendingDigit = -1
	return actionContinue, state
}

func handleBackspace(is *inputState) inputAction {
	if is.pendingDigit >= 0 {
		is.pendingDigit = -1
	} else if len(is.guess) > 0 {
		is.guess = is.guess[:len(is.guess)-1]
	}
	return actionContinue
}

func handleEnter(is *inputState, maxColors int, codeLength int) inputAction {
	if is.pendingDigit >= 0 {
		if is.pendingDigit >= 1 && is.pendingDigit <= maxColors && len(is.guess) < codeLength {
			is.guess = append(is.guess, is.pendingDigit)
		}
		is.pendingDigit = -1
		return actionContinue
	}
	if len(is.guess) >= codeLength {
		return actionSubmit
	}
	return actionContinue
}

func handleSpace(is *inputState, maxColors int, codeLength int) inputAction {
	if is.pendingDigit >= 0 {
		if is.pendingDigit >= 1 && is.pendingDigit <= maxColors && len(is.guess) < codeLength {
			is.guess = append(is.guess, is.pendingDigit)
		}
		is.pendingDigit = -1
	}
	return actionContinue
}

func handleDigit(is *inputState, b byte, maxColors int, codeLength int, multiDigit bool) inputAction {
	if len(is.guess) >= codeLength {
		return actionContinue
	}
	d := int(b - '0')

	if multiDigit {
		if is.pendingDigit >= 0 {
			twoDigit := is.pendingDigit*10 + d
			if twoDigit >= 1 && twoDigit <= maxColors {
				is.guess = append(is.guess, twoDigit)
				is.pendingDigit = -1
			} else if d >= 1 && d <= maxColors && d*10 > maxColors {
				is.guess = append(is.guess, d)
				is.pendingDigit = -1
			} else {
				is.pendingDigit = d
			}
		} else if d >= 1 && d <= maxColors && d*10 > maxColors {
			is.guess = append(is.guess, d)
		} else {
			is.pendingDigit = d
		}
	} else {
		if d >= 1 && d <= maxColors {
			is.guess = append(is.guess, d)
		}
	}
	return actionContinue
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
