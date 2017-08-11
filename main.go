package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/alexflint/go-arg"

	. "github.com/cirocosta/l7/lib"
)

type config struct {
	Port    int      `arg:"-p,help:port to listen to"`
	Config  string   `arg:"-c,help:configuration file to use"`
	Servers []string `arg:"positional"`
}

var (
	args     = &config{Port: 80}
	l7Config Config
	err      error
	sigs     = make(chan os.Signal)
)

func ShowBackendsConfig(backends map[string]Backend) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 4, '\t', 0)
	fmt.Fprintf(w, "BACKEND\tSERVER\n")

	for domain, backend := range backends {
		for ndx, srv := range backend.Servers {
			if ndx == 0 {
				fmt.Fprintf(w, "%s\t%s\n", domain, srv.Address)
			} else {
				fmt.Fprintf(w, "*\t%s\n", srv.Address)
			}
		}
		fmt.Fprintf(w, "---\t---\n")
	}
	w.Flush()
}

func handleSignals(lb *Flb, args *config) {
	for {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs,
			syscall.SIGHUP,
			syscall.SIGUSR1,
			syscall.SIGINT,
			syscall.SIGTERM)
		switch <-sigs {
		case syscall.SIGHUP:
			fmt.Println("INFO: Received SIGHUP.")
			if args.Config == "" {
				fmt.Println("Can't reload configuration.")
				fmt.Println("No configuration file specified at startup.")
				fmt.Println("No action taken.")
				continue
			}

			l7Config, err = NewConfigFromYamlFile(args.Config)
			if err != nil {
				fmt.Printf("ERROR: Couldn't parse configuration "+
					"file supplied %s\n%+v\n", args.Config, err)
				fmt.Println("No action taken.")
				continue
			}

			err = lb.LoadBackends(l7Config.Backends)
			if err != nil {
				fmt.Printf("ERROR: Couldn't load configuration from "+
					"file supplied %s\n%+v\n", args.Config, err)
				fmt.Println("No action taken.")
				continue
			}

			fmt.Println("INFO: Configuration reloaded")
			ShowBackendsConfig(lb.GetBackends())
		case syscall.SIGUSR1:
			ShowBackendsConfig(lb.GetBackends())
		case syscall.SIGINT:
			fmt.Println("Received SIGINT. Gracefully exiting.")
			lb.Stop()
		case syscall.SIGTERM:
			fmt.Println("Received SIGTERM. Gracefully exiting.")
			lb.Stop()
		}
	}
}

func main() {
	arg.MustParse(args)
	if args.Config != "" {
		l7Config, err = NewConfigFromYamlFile(args.Config)
		if err != nil {
			fmt.Printf("ERROR: Couldn't parse configuration "+
				"file supplied %s\n%+v\n", args.Config, err)
			os.Exit(1)
		}
	} else {
		backends, err := EqualSeparatedToBackends(args.Servers)
		if err != nil {
			fmt.Printf("ERROR: Couldn't create server "+
				"configuration from arguments.\n%+v\n", err)
			fmt.Printf("See use help by issuing 'l7 --help'.\n")
			os.Exit(1)
		}

		l7Config = Config{Port: args.Port, Backends: backends}
	}

	lb, err := New(l7Config)
	if err != nil {
		fmt.Printf("ERROR: Couldn't initialize flb with provided "+
			"config %+v\n %+v\n", l7Config, err)
		os.Exit(1)
	}

	go handleSignals(&lb, args)

	ShowBackendsConfig(lb.GetBackends())

	err = lb.Listen()
	if err != nil {
		fmt.Printf("ERROR: Couldn't make load-balancer listen %+v\n %+v\n",
			l7Config, err)
		os.Exit(1)
	}
}
