package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/mzz2017/gg/cmd/infra"
	"github.com/mzz2017/gg/tracer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	v       *viper.Viper
	Version = "unknown"
	verbose int
	rootCmd = &cobra.Command{
		Use:   "gg [flags] [command [argument ...]]",
		Short: "go-graft redirects the traffic of given program to your proxy.",
		Long: `go-graft is a portable tool to redirect the traffic of a given 
program to your modern proxy without installing any other programs.`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			program := filepath.Base(os.Args[0])
			hasSelectFlag, _ := cmd.PersistentFlags().GetBool("select")
			if len(args) == 0 && !hasSelectFlag {
				fmt.Println(`No command is given, you can try:
$ gg --help
or
$ gg git clone https://github.com/mzz2017/gg.git`)
				return
			}

			// auto su if use 'gg sudo' or 'gg su'
			if len(os.Args) >= 2 {
				cmdName := filepath.Base(os.Args[1])
				if cmdName == "sudo" || cmdName == "su" {
					infra.AutoSu()
				}
			}
			// initiate config from args and config file
			log := NewLogger(verbose)
			log.Traceln("Version:", Version)
			log.Tracef("OS/Arch: %v/%v\n", runtime.GOOS, runtime.GOARCH)
			v, _ = getConfig(log, true, viper.New, cmd)

			// check ptrace_scope and capability
			if err := infra.CheckPtraceCapability(); err != nil {
				switch err {
				case infra.ErrBadCapability:
					path, err := filepath.Abs(os.Args[0])
					if err != nil {
						path = filepath.Clean(os.Args[0])
					}
					log.Fatalf("Your ptrace_scope is 2 and you should give the correct capability to %v:\nsudo setcap cap_net_raw,cap_sys_ptrace+ep %v", program, path)
				case infra.ErrBadPtraceScope:
					log.Fatalln("Your kernel does not allow ptrace permission, please use following command and reboot:\necho kernel.yama.ptrace_scope = 1 | sudo tee -a /etc/sysctl.d/10-ptrace.conf")
				default:
					log.Infoln(err)
				}
			}

			// validate command and get the fullPath from $PATH
			var (
				fullPath string
				err      error
			)
			if len(args) != 0 {
				fullPath, err = exec.LookPath(args[0])
				if err != nil {
					logrus.Fatal("exec.LookPath:", err)
				}
			}
			// get dialer
			dialer, err := GetDialer(log)
			if err != nil {
				logrus.Fatal("GetDialer:", err)
			}

			if len(args) == 0 {
				return
			}

			// Get no_udp from argument first, then from configuration file.
			var noUDP bool
			noUDPFlag := cmd.Flags().Lookup("noudp")
			if noUDPFlag != nil && noUDPFlag.Changed {
				if noUDP, err = cmd.Flags().GetBool("noudp"); err != nil {
					logrus.Fatal("GetBool(noudp):", err)
				}
			} else {
				noUDP = v.GetBool("no_udp")
			}
			if !noUDP && !dialer.SupportUDP() {
				log.Info("Your proxy server does not support UDP, so we will not redirect UDP traffic.")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			t, err := tracer.New(
				ctx,
				fullPath,
				args,
				&os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}, Env: os.Environ()},
				dialer,
				noUDP,
				log,
			)
			if err != nil {
				logrus.Fatal("tracer.New:", err)
			}
			go func() {
				// listen signal
				sigs := make(chan os.Signal, 1)
				signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGILL)
				<-sigs
				cancel()
			}()
			code, err := t.Wait()
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					logrus.Fatal("tracer.Wait:", err)
				}
			}
			os.Exit(code)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "verbose (-v, or -vv)")

	rootCmd.PersistentFlags().StringP("node", "n", "", "node share-link of your modern proxy")
	rootCmd.PersistentFlags().StringP("subscription", "s", "", "subscription-link of your modern proxy")
	rootCmd.PersistentFlags().Bool("noudp", false, "do not redirect UDP traffic, even though the proxy server supports")
	rootCmd.PersistentFlags().String("testnode", "true", "test the connectivity before connecting to the node")
	rootCmd.PersistentFlags().Bool("select", false, "manually select the node to connect from the subscription")
	rootCmd.AddCommand(configCmd)
}

func NewLogger(verbose int) *logrus.Logger {
	log := logrus.New()

	var level logrus.Level
	switch verbose {
	case 0:
		level = logrus.WarnLevel
	case 1:
		level = logrus.InfoLevel
	default:
		level = logrus.TraceLevel
	}
	log.SetLevel(level)

	return log
}
