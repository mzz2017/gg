package cmd

import (
	"bytes"
	"fmt"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/config"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	configCmd = &cobra.Command{
		Use:   "config",
		Short: "Get and set persistent global options",
		Run: func(cmd *cobra.Command, args []string) {
			// read
			v, configPath := getConfig(logrus.New(), true, viper.New, nil)
			write, _ := cmd.PersistentFlags().GetString("write")
			if len(write) == 0 {
				kv := common.ObjectToKV(config.ParamsObj, "mapstructure")
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
				sort.Strings(kv)
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
			// make sure it is valid
			key := completeKey(fields[0])
			val := fields[1]
			if err := config.SetValueHierarchicalStruct(&config.Params{}, key, val); err != nil {
				logrus.Fatalln(err)
			}
			if err := config.SetValueHierarchicalMap(m, key, val); err != nil {
				logrus.Fatalln(err)
			}
			if err := WriteConfig(m, configPath); err != nil {
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

func WriteConfig(settings map[string]interface{}, configPath string) error {
	_ = os.MkdirAll(filepath.Dir(configPath), 0755)
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).SetTagName("mapstructure").Encode(settings); err != nil {
		return err
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
		return err
	}
	return nil
}

func completeKey(key string) string {
	switch key {
	case "subscription":
		return "subscription.link"
	}
	return key
}

func getConfig(log *logrus.Logger, bindToConfig bool, newViper func() *viper.Viper, flagCmd *cobra.Command) (v *viper.Viper, path string) {
	v = newViper()
	home, err := os.UserHomeDir()
	if err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "gg"))
		v.SetConfigName("config")
		v.SetConfigType("toml")
		err = v.ReadInConfig()
		if err != nil {
			v = newViper()
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
		if subscription, _ := flagCmd.PersistentFlags().GetString("subscription"); subscription != "" && subscription != v.Get("subscription.link") {
			v.Set("subscription.cache_last_node", "false")
			log.Infoln("subscription.cache_last_node will be disabled because the given subscription link is different from the configured one.")
		}
		v.BindPFlag("subscription.link", flagCmd.PersistentFlags().Lookup("subscription"))
		if ok, _ := flagCmd.PersistentFlags().GetBool("select"); ok {
			v.Set("subscription.select", "__select__")
		}
	}
	if bindToConfig {
		if err := config.NewBinder(v).Bind(config.ParamsObj); err != nil {
			log.Fatalf("Fatal error loading config: %s", err)
		}
		if err := v.Unmarshal(&config.ParamsObj); err != nil {
			log.Fatalf("Fatal error loading config: %s", err)
		}
	}

	log.Tracef("Config:\n%v\n", strings.Join(common.MapToKV(v.AllSettings()), "\n"))

	if v.ConfigFileUsed() == "" {
		if home == "" {
			return v, filepath.Join("/etc/ggconfig.toml")
		}
		return v, filepath.Join(home, ".ggconfig.toml")
	}
	return v, v.ConfigFileUsed()
}
