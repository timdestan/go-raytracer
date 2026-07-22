// The gml command runs an interactive shell for
// interpreting the GML language.
package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ergochat/readline"
	"github.com/timdestan/go-raytracer"
	"github.com/timdestan/go-raytracer/internal/gml"
)

type Breakpoint struct {
	Line int // Just a line number for now.
}

func parseBreakpoint(arg string) (*Breakpoint, error) {
	val, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return nil, err
	}
	if val <= 0 {
		return nil, fmt.Errorf("breakpoint must be positive line number.")
	}

	// TODO: Should we check it's in range of the file?
	// TODO: Should we restrict breakpoints to the current file, so they don't
	//       apply if we reload the file? Probably....

	return &Breakpoint{Line: int(val)}, nil
}

type Command struct {
	// Symbol is the canonical name of the command.
	// It should include the leading ":".
	Symbol       string
	Aliases      []string
	ExpectedArgs []string // For generating help.
	HelpText     string
	Run          func(*State) error
}

type State struct {
	args          []string
	evalState     *gml.EvalState
	pc            int
	program       gml.TokenList
	commands      []*Command
	commandLookup map[string]*Command
	breakpoints   []*Breakpoint
}

func (s *State) toggleBreakpoint(bp *Breakpoint) (wasPresent bool) {
	for i := 0; i < len(s.breakpoints); i++ {
		s.breakpoints = slices.Delete(s.breakpoints, i, i+1)
		return true
	}
	s.breakpoints = append(s.breakpoints, bp)
	return false
}

func (s *State) findMatchingBreakpoint(line int) *Breakpoint {
	for _, bp := range s.breakpoints {
		if bp.Line == line {
			return bp
		}
	}
	return nil
}

func (s *State) loadFile(filepath string) error {
	programText, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	prog, err := s.evalState.Parse(string(programText))
	if err == nil {
		s.program = prog
		s.pc = 0
	}
	return err
}

// errQuit is a signal to the main loop to quit.
var errQuit = errors.New("quit")

func main() {
	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:       "gml> ",
		HistoryFile:  readlineHistoryFilePath(),
		HistoryLimit: 10000,
		// TODO: Autocomplete.
	})
	if err != nil {
		log.Fatalf("readline init error: %v", err)
	}

	images := make(map[string]image.Image)

	evalState := gml.NewEvalState()
	evalState.Render = func(e *gml.EvalState, args *gml.RenderArgs) error {
		scene, err := raytracer.ConvertRenderArgsToScene(args, e)
		if err != nil {
			return err
		}
		images[args.File] = raytracer.Render(scene)
		fmt.Printf("Rendered image with name %s\n", args.File)
		return nil
	}

	state := State{
		evalState:     evalState,
		commandLookup: make(map[string]*Command),
	}

	registerCommand := func(command *Command) {
		mustAddToLookup := func(symbol string) {
			if state.commandLookup[symbol] != nil {
				log.Fatalf("duplicate command: %v vs %v", command, state.commandLookup[symbol])
			}
			state.commandLookup[symbol] = command
		}
		state.commands = append(state.commands, command)
		mustAddToLookup(command.Symbol)
		for _, alias := range command.Aliases {
			mustAddToLookup(alias)
		}
	}

	registerCommand(&Command{
		Symbol:       ":load",
		Aliases:      []string{":l"},
		ExpectedArgs: []string{"<filename>"},
		HelpText:     "Load and parse a file",
		Run: func(st *State) error {
			if len(st.args) < 1 {
				return errors.New("usage: :load filename")
			}
			return st.loadFile(st.args[0])
		},
	})

	registerCommand(&Command{
		Symbol:   ":step",
		Aliases:  []string{":s"},
		HelpText: "Runs a single step of the evaluator",
		Run: func(st *State) error {
			if len(st.args) != 0 {
				return errors.New("usage: :step")
			}
			if len(st.program) == 0 {
				return errors.New("No program loaded, use :load filename to load a program")
			}
			if st.pc >= len(st.program) {
				return errors.New("program halted")
			}
			curr := st.program[st.pc]
			defer func() { st.pc++ }()
			fmt.Printf("%s: %s\n", curr.Position().String(), gml.TokenGroupDebugString(curr))
			return st.evalState.EvalOneStep(curr)
		},
	})

	registerCommand(&Command{
		Symbol:   ":break",
		Aliases:  []string{":b"},
		HelpText: "Sets or clears a breakpoint at a given line. Run without arguments to list current breakpoints.",
		Run: func(st *State) error {
			if len(st.args) > 1 {
				return errors.New("usage: :break line?")
			}
			if len(st.args) == 0 {
				fmt.Printf("All breakpoints:\n")
				if len(st.breakpoints) == 0 {
					fmt.Printf("  (none)\n")
				}
				for _, b := range st.breakpoints {
					fmt.Printf("  Line: %d\n", b.Line)
				}
			} else {
				bp, err := parseBreakpoint(st.args[0])
				if err != nil {
					return err
				}
				wasExisting := st.toggleBreakpoint(bp)
				if wasExisting {
					fmt.Printf("Removed breakpoint at line %d\n", bp.Line)
				} else {
					fmt.Printf("Added breakpoint at line %d\n", bp.Line)
				}
			}
			return nil
		},
	})

	registerCommand(&Command{
		Symbol:   ":run",
		Aliases:  []string{":r"},
		HelpText: "Runs to the end of the loaded file. If <filename> provided, loads the file first.",
		Run: func(st *State) error {
			if len(st.args) > 1 {
				return errors.New("usage: :run filename?")
			}
			if len(st.args) == 1 {
				if err := st.loadFile(st.args[0]); err != nil {
					return err
				}
			}
			if len(st.program) == 0 {
				return errors.New("No program loaded, use :load filename to load a program")
			}
			if st.pc >= len(st.program) {
				return errors.New("program halted")
			}

			currLine := st.program[st.pc].Position().Line
			for ; st.pc < len(st.program); st.pc++ {
				curr := st.program[st.pc]

				// Only trigger breakpoint when we first hit the line.
				nextLine := curr.Position().Line
				if nextLine != currLine {
					bp := st.findMatchingBreakpoint(nextLine)
					if bp != nil {
						fmt.Printf("Hit breakpoint at line %d\n", bp.Line)
						return nil
					}
				}
				currLine = nextLine

				fmt.Printf("%s: %s\n", curr.Position().String(), gml.TokenGroupDebugString(curr))
				if err := st.evalState.EvalOneStep(curr); err != nil {
					return err
				}
			}
			return nil
		},
	})

	registerCommand(&Command{
		Symbol:   ":env",
		HelpText: "Print the current environment",
		Run: func(st *State) error {
			// TODO: Without debug string context, the variable names are only
			// shown as numeric ids, which is not very useful.
			fmt.Printf("env: %v\n", st.evalState.Env)
			return nil
		},
	})
	registerCommand(&Command{
		Symbol:   ":write-png",
		HelpText: "Writes an image that was previously generated to a PNG file",
		Run: func(st *State) error {
			if len(st.args) < 2 {
				return errors.New("usage: :write-png <imagename> <filename.png>")
			}
			img, ok := images[st.args[0]]
			if !ok {
				return fmt.Errorf("no image with name %s", st.args[0])
			}
			return writeImage(img, st.args[1])
		},
	})
	registerCommand(&Command{
		Symbol:   ":stack",
		HelpText: "Print the current stack",
		Run: func(st *State) error {
			fmt.Printf("stack:\n")
			for i, v := range st.evalState.Stack {
				fmt.Printf("  %v: %v\n", i, v)
			}
			return nil
		},
	})
	registerCommand(&Command{
		Symbol:   ":help",
		Aliases:  []string{":h"},
		HelpText: "Prints this help text",
		Run:      showHelp,
	})
	registerCommand(&Command{
		Symbol:   ":quit",
		Aliases:  []string{":q"},
		HelpText: "Exit the shell",
		Run: func(st *State) error {
			return errQuit
		},
	})

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
				// Exit gracefully on expected errors.
				return
			}
			log.Fatalf("readline error: %v", err)
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		} else if line[0] == ':' {
			// Parse and evaluate a shell command.
			args := parseCommandArgs(line)
			if len(args) == 0 {
				log.Fatalf("bug in command parser: %q", line)
			}
			cmd := state.commandLookup[args[0]]
			if cmd == nil {
				fmt.Printf("Unknown command: %v\n", args[0])
				continue
			}
			state.args = args[1:]
			err := cmd.Run(&state)
			if errors.Is(err, errQuit) {
				return
			}
			if err != nil {
				fmt.Printf("command error: %v\n", err)
				continue
			}
		} else {
			// Otherwise treat the line as GML input.
			err := evalState.ParseAndEval(line)
			if err != nil {
				fmt.Printf("GML error: %v\n", err)
				continue
			}
		}
	}
}

func showHelp(st *State) error {
	usageHelp := make([]string, len(st.commands))
	maxLen := 0
	for i, command := range st.commands {
		parts := []string{command.Symbol}
		parts = append(parts, command.Aliases...)
		parts = append(parts, command.ExpectedArgs...)
		usageHelp[i] = strings.Join(parts, " ")
		maxLen = max(maxLen, len(usageHelp[i]))
	}
	fmt.Printf("Commands:\n")
	for i, command := range st.commands {
		fmt.Printf("  %-*s : %s\n", maxLen, usageHelp[i], command.HelpText)
	}
	return nil
}

func readlineHistoryFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("user home dir error: %v\n", err)
		return ""
	}
	return filepath.Join(home, ".gml_history")
}

func parseCommandArgs(line string) []string {
	var args []string
	var start int
	for i := range line {
		curr := line[i]
		if strings.IndexByte(" \t\n\r", curr) != -1 {
			if start < i {
				args = append(args, line[start:i])
			}
			start = i + 1
		}
	}
	if start < len(line) {
		args = append(args, line[start:])
	}
	return args
}

func writeImage(img image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
