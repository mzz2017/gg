package cmd

import (
	"fmt"
	"github.com/mzz2017/gg/cmd/infra"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/config"
	"github.com/mzz2017/gg/tracer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
			initConfig(cmd, log)
			// validate command and get the fullPath from $PATH
			fullPath, err := exec.LookPath(args[0])
			if err != nil {
				logrus.Fatal(err)
			}
			// get dialer
			dialer, err := infra.GetDialer(log)
			if err != nil {
				logrus.Fatal(err)
			}

			noUDP, err := cmd.Flags().GetBool("noudp")
			if err != nil {
				logrus.Fatal(err)
			}
			if !noUDP && !dialer.SupportUDP() {
				log.Warn("Your proxy server does not support UDP, so we will not redirect UDP traffic.")
				noUDP = true
			}
			t, err := tracer.New(
				fullPath,
				args,
				&os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}},
				dialer,
				noUDP,
				log,
			)
			if err != nil {
				logrus.Fatal(err)
			}
			code, err := t.Wait()
			if err != nil {
				logrus.Fatal(err)
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
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Verbose (-v, or -vv)")

	rootCmd.PersistentFlags().StringP("node", "n", "", "Node share-link of your modern proxy")
	rootCmd.PersistentFlags().StringP("subscription", "s", "", "Subscription-link of your modern proxy")
	rootCmd.PersistentFlags().Bool("noudp", false, "Do not redirect UDP traffic, even though the proxy server supports")
	rootCmd.PersistentFlags().Bool("testnode", true, "Test the connectivity before connecting to the node.")
	rootCmd.PersistentFlags().Bool("select", false, "Manually select the node to connect from the subscription.")
	rootCmd.AddCommand(configCmd)
}

func initConfig(flagCmd *cobra.Command, log *logrus.Logger) (path string) {
	v = viper.New()
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "gg"))
		v.SetConfigName("config")
		v.SetConfigType("toml")
		err = v.ReadInConfig()
		if err != nil {
			v = viper.New()
			v.AddConfigPath(home)
			v.SetConfigName(".ggconfig")
			v.SetConfigType("toml")
			err = v.ReadInConfig()
		}
	}
	if err != nil {
		v = viper.New()
		v.AddConfigPath("/etc/")
		v.SetConfigName("ggconfig")
		v.SetConfigType("toml")
		err = v.ReadInConfig()
	}
	if err == nil {
		log.Tracef("Using config file: %v", v.ConfigFileUsed())
	} else if err != nil {
		switch err.(type) {
		default:
			log.Fatalf("Fatal error loading config file: %s: %s", v.ConfigFileUsed(), err)
		case viper.ConfigFileNotFoundError:
			log.Tracef("No config file found. Using default values.")
		}
	}

	if flagCmd != nil {
		v.BindPFlag("no_udp", flagCmd.PersistentFlags().Lookup("noudp"))
		v.BindPFlag("test_node_before_use", flagCmd.PersistentFlags().Lookup("testnode"))
		if node, _ := flagCmd.PersistentFlags().GetString("node"); node != "" {
			//log.Warn("Please use --node only on trusted computers, because it may leave a record in command history.")
			v.BindPFlag("node", flagCmd.PersistentFlags().Lookup("node"))
		}
		v.BindPFlag("subscription.link", flagCmd.PersistentFlags().Lookup("subscription"))
		if ok, _ := flagCmd.PersistentFlags().GetBool("select"); ok {
			v.Set("subscription.select", "manual")
			v.Set("subscription.cache_last_node", "false")
		}
	}
	if err := config.NewBinder(v).Bind(config.ParamsObj); err != nil {
		log.Fatalf("Fatal error loading config: %s", err)
	}
	if err := v.Unmarshal(&config.ParamsObj); err != nil {
		log.Fatalf("Fatal error loading config: %s", err)
	}

	log.Tracef("Config file path: %v\n", v.ConfigFileUsed())
	log.Tracef("Config:\n%v\n", strings.Join(common.MapToKV(v.AllSettings()), "\n"))

	if v.ConfigFileUsed() == "" {
		if home == "" {
			return filepath.Join("/etc/ggconfig.toml")
		}
		return filepath.Join(home, ".ggconfig.toml")
	}
	return v.ConfigFileUsed()
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
