// The gml command runs an interactive shell for
// interpreting the GML language.
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ergochat/readline"
	"github.com/timdestan/go-raytracer/internal/gml"
)

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
	args      []string
	evalState *gml.EvalState
	commands  []*Command
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

	evalState := gml.NewEvalState()
	evalState.Render = func(e *gml.EvalState, args *gml.RenderArgs) error {
		// TODO: Actually render.
		fmt.Printf("render: %v\n", args)
		return nil
	}

	var commands []*Command
	commandLookup := make(map[string]*Command)

	registerCommand := func(command *Command) {
		mustAddToLookup := func(symbol string) {
			if commandLookup[symbol] != nil {
				log.Fatalf("duplicate command: %v vs %v", command, commandLookup[symbol])
			}
			commandLookup[symbol] = command
		}
		commands = append(commands, command)
		mustAddToLookup(command.Symbol)
		for _, alias := range command.Aliases {
			mustAddToLookup(alias)
		}
	}

	registerCommand(&Command{
		Symbol:       ":load",
		Aliases:      []string{":l"},
		ExpectedArgs: []string{"<filename>"},
		HelpText:     "Load a file",
		Run: func(st *State) error {
			if len(st.args) < 1 {
				return errors.New("usage: :load <filename>")
			}
			prog, err := os.ReadFile(st.args[0])
			if err != nil {
				return err
			}
			return evalGML(string(prog), st.evalState)
		},
	})
	registerCommand(&Command{
		Symbol:   ":env",
		Aliases:  []string{":e"},
		HelpText: "Print the current environment",
		Run: func(st *State) error {
			fmt.Printf("env:\n")
			for k, v := range st.evalState.Env {
				fmt.Printf("  %v = %v\n", k, v)
			}
			return nil
		},
	})
	registerCommand(&Command{
		Symbol:   ":stack",
		Aliases:  []string{":s"},
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
			cmd := commandLookup[args[0]]
			if cmd == nil {
				fmt.Printf("Unknown command: %v\n", args[0])
				continue
			}
			err := cmd.Run(&State{
				args:      args[1:],
				evalState: evalState,
				commands:  commands,
			})
			if errors.Is(err, errQuit) {
				return
			}
			if err != nil {
				fmt.Printf("command error: %v\n", err)
				continue
			}
		} else {
			// Otherwise treat the line as GML input.
			err := evalGML(line, evalState)
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

func evalGML(text string, state *gml.EvalState) error {
	tokens, err := gml.NewParser(text).Parse()
	if err != nil {
		return err
	}
	return state.Eval(tokens)
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
