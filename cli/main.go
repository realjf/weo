package main

import (
	"encoding/json"
	"fmt"
	"github.com/flynn/go-docopt"
	"os"
	"log"
	"strings"
	"unicode"
	controller "weo/controller/client"
	"weo/pkg/shutdown"
	"weo/pkg/version"
)

var (
	flagCluster = os.Getenv("WEO_CLUSTER")
	flagApp string
)

func main() {
	defer shutdown.Exit()

	log.SetFlags(0)

	usage := `
usage: weo [-a <app>] [-c <cluster>] <command> [<args>...]

Options:
	-a <app>
	-c <cluster>
	-h, --help

Commands:
	help		show usage for a specific command
	cluster		manage clusters
	create		create an app
	delete		delete an app
	apps 		list apps
	info		show app information
	ps			list jobs
	kill		kill jobs
	log			get app log
	scale		change formation
	run			run a job
	env			manage env variables
	limit		manage resource limits
	meta		manage app metadata
	route		manage routes
	pg			manage postgres database
	mysql		manage mysql database
	mongodb		manage mongodb database
	redis		manage redis database
	provider	manage resource providers
	docker		deploy Docker images to a Weo cluster
	remote		manage git remotes
	resource	provision a new resource
	release		manage app releases
	deployment	list deployments
	volume		manage volumes
	export		export app data
	import		create app from exported data
	version		show weo version

See 'weo help <command>' for more information on a specific command.
`[1:]
	args, _ := docopt.Parse(usage, nil, true, version.String(), true)

	cmd := args.String["<command>"]
	cmdArgs := args.All["<args>"].([]string)

	if cmd == "help" {
		if len(cmdArgs) == 0 {
			fmt.Println(usage)
			return
		}else if cmdArgs[0] == "--json" {
			cmds := make(map[string]string)
			for name, cmd := range commands {
				cmds[name] = cmd.usage
			}
			out, err := json.MarshalIndent(cmds, "", "\t")
			if err != nil {
				shutdown.Fatal(err)
			}
			fmt.Println(string(out))
			return
		}else {
			cmd = cmdArgs[0]
			cmdArgs = make([]string, 1)
			cmdArgs[0] = "--help"
		}
	}

	if cmd == "update" {
		if err := runUpdate(); err != nil {
			shutdown.Fatal(err)
		}
		return
	} else {
		defer updater.backgroundRun()
	}

	if args.String["-c"] != "" {
		flagCluster = args.String["-c"]
	}

	flagApp = args.String["-a"]
	if flagApp != "" {
		if err := readConfig(); err != nil {
			shutdown.Fatal(err)
		}

		if ra, err := appFromGitRemote(flagApp); err != nil {
			clusterConf = ra.Cluster
			flagApp = ra.Name
		}
	}

	if err := runCommand(cmd, cmdArgs); err != nil {
		log.Println(err)
		shutdown.ExitWithCode(1)
		return
	}
}

type command struct {
	usage string
	f interface{}
	optsFirst bool
}

var commands = make(map[string]*command)

func register(cmd string, f interface{}, usage string) *command {
	switch f.(type) {
	case func(*docopt.Args, controller.Client) error, func(*docopt.Args) error, func() error, func():
	default:
		panic(fmt.Sprintf("invalid command function %s '%T", cmd, f))
	}
	c := &command{usage: strings.TrimLeftFunc(usage, unicode.IsSpace), f: f}
	commands[cmd] = c
	return c
}

func runCommand(name string, args []string) (err error) {
	argv := make([]string, 1, 1+len(args))
	argv[0] = name
	argv = append(argv, args...)

	cmd, ok := commands[name]
	if !ok {
		return fmt.Errorf("%s is not a weo command. See 'weo help", name)
	}
	parseArgs, err := docopt.Parse(cmd.usage, argv, true, "", cmd.optsFirst)
	if err != nil {
		return err
	}

	switch f := cmd.f.(type) {
	case func(*docopt.Args, controller.Client) error:
		client, err := getClusterClient()
		if err != nil {
			shutdown.Fatal(err)
		}

		return f(parseArgs, client)
	case func(*docopt.Args) error:
		return f(parseArgs)
	case func() error:
		return f()
	case func():
		f()
		return nil
	}

	return fmt.Errorf("unexcepted command type %T", cmd.f)
}

var config *cfg.Config
var clusterConf *cfg.Cluster

func configPath() string {
	return cfg.DefaultPath
}
