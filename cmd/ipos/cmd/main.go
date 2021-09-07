package cmd

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/storeros/ipos/pkg/cli"
	"github.com/storeros/ipos/pkg/console"
	"github.com/storeros/ipos/pkg/trie"
	"github.com/storeros/ipos/pkg/words"
	"github.com/storeros/ipos/version"
)

var GlobalFlags = []cli.Flag{}

var iposHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

DESCRIPTION:
  {{.Description}}

USAGE:
  {{.HelpName}} {{if .VisibleFlags}}[FLAGS] {{end}}COMMAND{{if .VisibleFlags}}{{end}} [ARGS...]

COMMANDS:
  {{range .VisibleCommands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
VERSION:
  {{.Version}}
`

func newApp(name string) *cli.App {
	commands := []cli.Command{}

	commandsTree := trie.NewTrie()

	registerCommand := func(command cli.Command) {
		commands = append(commands, command)
		commandsTree.Insert(command.Name)
	}

	findClosestCommands := func(command string) []string {
		var closestCommands []string
		for _, value := range commandsTree.PrefixMatch(command) {
			closestCommands = append(closestCommands, value.(string))
		}

		sort.Strings(closestCommands)
		for _, value := range commandsTree.Walk(commandsTree.Root()) {
			if sort.SearchStrings(closestCommands, value.(string)) < len(closestCommands) {
				continue
			}
			if words.DamerauLevenshteinDistance(command, value.(string)) < 2 {
				closestCommands = append(closestCommands, value.(string))
			}
		}

		return closestCommands
	}

	registerCommand(serverCmd)

	cli.HelpFlag = cli.BoolFlag{
		Name:  "help, h",
		Usage: "show help",
	}

	app := cli.NewApp()
	app.Name = name
	app.Author = "StorerOS"
	app.Version = version.Version
	app.Usage = "High Performance Object Storage"
	app.Description = `Build high performance data infrastructure for machine learning, analytics and application data workloads with IPOS`
	app.Flags = GlobalFlags
	app.HideHelpCommand = true
	app.Commands = commands
	app.CustomAppHelpTemplate = iposHelpTemplate
	app.CommandNotFound = func(ctx *cli.Context, command string) {
		console.Printf("‘%s’ is not a ipos sub-command. See ‘ipos --help’.\n", command)
		closestCommands := findClosestCommands(command)
		if len(closestCommands) > 0 {
			console.Println()
			console.Println("Did you mean one of these?")
			for _, cmd := range closestCommands {
				console.Printf("\t‘%s’\n", cmd)
			}
		}

		os.Exit(1)
	}

	return app
}

func Main(args []string) {
	appName := filepath.Base(args[0])
	if err := newApp(appName).Run(args); err != nil {
		os.Exit(1)
	}
}
