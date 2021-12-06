package cmd

import (
	"bytes"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Get and set persistent global options",
		Run: func(cmd *cobra.Command, args []string) {
			// read
			configPath := initConfig(nil, logrus.New())
			write, _ := cmd.PersistentFlags().GetString("write")
			if len(write) == 0 {
				kv := common.ObjectToKV(config.ParamsObj, "toml")
				if len(args) != 0 {
					keySetToFind := common.StringsMapToSet(args, completeKey)
					// show config variable
					for i := range kv {
						fields := strings.SplitN(kv[i], "=", 2)
						if _, ok := keySetToFind[fields[0]]; ok {
							fmt.Println(fields[1])
						}
					}
					return
				}
				// show all config variables
				fmt.Println(strings.Join(kv, "\n"))
				return
			}
			// set value
			m := v.AllSettings()
			fields := strings.SplitN(write, "=", 2)
			if len(fields) == 1 {
				logrus.Fatalln(`Unexpected format.
For example:
gg config -w no_udp=true`)
			}
			if err := config.SetValueHierarchicalMap(m, completeKey(fields[0]), fields[1]); err != nil {
				logrus.Fatalln(err)
			}

			_ = os.MkdirAll(filepath.Dir(configPath), 0755)
			buf := new(bytes.Buffer)
			if err := toml.NewEncoder(buf).Encode(m); err != nil {
				logrus.Fatalln(err)
			}
			if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
				logrus.Fatalln(err)
			}
			//logrus.Info("Make sure you are using config on a trusted computer")
			fmt.Println(write)
		},
	}
)

func init() {
	configCmd.PersistentFlags().StringP("write", "w", "", "write config variable")
}

func WriteConfig() {
	// TODO: cache last node
}

func completeKey(key string) string {
	switch key {
	case "subscription":
		return "subscription.link"
	}
	return key
}
