package cmd

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/zeroalloc/bufio"
	"github.com/mzz2017/gg/dialer"
	"github.com/mzz2017/gg/tracer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var (
	cfgFile string
	Version = "unknown"
	verbose int
	rootCmd = &cobra.Command{
		Use:     "gg [flags] [-- command [arument ...]]",
		Short:   "go-graft redirects the traffic of given program to your proxy.",
		Long:    `go-graft redirects the traffic of given program to your proxy.`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logrus.Println("no")
				return
			}
			log := GetLogger(verbose)

			fullPath, err := exec.LookPath(args[0])
			if err != nil {
				logrus.Fatal(err)
			}
			dialer, err := GetDialer(log, cmd)
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
	rootCmd.PersistentFlags().StringP("link", "l", "", "share-link of your modern proxy")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "verbose (-v, or -vv)")
	rootCmd.PersistentFlags().Bool("noudp", false, "Do not redirect UDP traffic, even though the proxy server supports")
	//rootCmd.AddCommand(installCmd)
	//rootCmd.AddCommand(runCmd)
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

func GetDialer(log *logrus.Logger,cmd *cobra.Command) (*dialer.Dialer, error) {
	link, err := cmd.Flags().GetString("link")
	if err != nil {
		return nil, err
	}
	if len(link) > 0 {
		log.Warn("Please use --link only on trusted computers, because it may leave a record in the command history.")
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter the share-link of your proxy: ")
		link, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		link = strings.TrimSpace(link)
	}
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	return dialer.NewFromLink(u.Scheme, u.String())
}
