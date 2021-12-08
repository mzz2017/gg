package cmd

import (
	"fmt"
	"github.com/mzz2017/gg/tracer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/exec"
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
			if len(args) == 0 {
				fmt.Println(`No command is given, you can try:
$ gg --help
or
$ gg git clone https://github.com/mzz2017/gg.git`)
				return
			}
			// initiate config from args and config file
			log := GetLogger(verbose)
			v, _ = getConfig(log, true, viper.New, cmd)
			// validate command and get the fullPath from $PATH
			fullPath, err := exec.LookPath(args[0])
			if err != nil {
				logrus.Fatal("exec.LookPath:", err)
			}
			// get dialer
			dialer, err := GetDialer(log)
			if err != nil {
				logrus.Fatal("GetDialer:", err)
			}

			noUDP, err := cmd.Flags().GetBool("noudp")
			if err != nil {
				logrus.Fatal("GetBool(noudp):", err)
			}
			if !noUDP && !dialer.SupportUDP() {
				log.Warn("Your proxy server does not support UDP, so we will not redirect UDP traffic.")
				noUDP = true
			}
			preserveEnv, err := cmd.Flags().GetBool("preserve-env")
			if err != nil {
				logrus.Fatal("GetBool(preserve-env):", err)
			}
			var env []string
			if preserveEnv {
				env = os.Environ()
			}
			t, err := tracer.New(
				fullPath,
				args,
				&os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}, Env: env},
				dialer,
				noUDP,
				log,
			)
			if err != nil {
				logrus.Fatal("tracer.New:", err)
			}
			code, err := t.Wait()
			if err != nil {
				logrus.Fatal("tracer.Wait:", err)
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
	rootCmd.PersistentFlags().Bool("testnode", true, "test the connectivity before connecting to the node")
	rootCmd.PersistentFlags().Bool("select", false, "manually select the node to connect from the subscription")
	rootCmd.PersistentFlags().BoolP("preserve-env", "E", false, "preserve user environment when running command")
	rootCmd.AddCommand(configCmd)
}

func GetLogger(verbose int) *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
	if verbose > 0 {
		if verbose == 1 {
			log.SetLevel(logrus.InfoLevel)
		} else {
			log.SetLevel(logrus.TraceLevel)
		}
	}
	return log
}
