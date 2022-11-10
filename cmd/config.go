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
	"log"
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
			v, configPath := getConfig(logrus.New(), true, viper.New, nil)
			write, _ := cmd.PersistentFlags().GetString("write")
			unset, _ := cmd.PersistentFlags().GetString("unset")
			if len(write) != 0 && len(unset) != 0 {
				logrus.Fatalln(`Conflicting flags --unset and --write detected; you can only specify one of them.`)
			}

			// read
			if len(write) == 0 && len(unset) == 0 {
				kv := common.ObjectToKV(config.ParamsObj, "mapstructure")
				if len(args) != 0 {
					keySetToFind := common.StringsMapToSet(args, completeKey)
					for _, arg := range args {
						if strings.Contains(arg, "=") {
							logrus.Warnf("Did you forget to add '-w'?")
							break
						}
					}
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
			var (
				key string
				val string
			)
			if len(write) != 0 {
				fields := strings.SplitN(write, "=", 2)
				if len(fields) == 1 {
					logrus.Fatalln(`Unexpected format.
For example:
gg config -w no_udp=true`)
				}
				key = completeKey(fields[0])
				val = fields[1]
			} else if len(unset) != 0 {
				key = completeKey(unset)
				// Use empty viper and empty params to get the default value of target key.
				var (
					emptyViper  = viper.New()
					emptyParams config.Params
				)
				config.NewBinder(emptyViper).Bind(config.Params{})
				if err := emptyViper.Unmarshal(&emptyParams); err != nil {
					log.Fatalf("Fatal error loading empty config: %s", err)
				}
				defaultValue, err := config.GetValueHierarchicalStruct(emptyParams, key)
				if err != nil {
					logrus.Fatalln(err)
				}
				val = fmt.Sprint(defaultValue.Interface())
				fmt.Printf("%v=%v", key, val)
			} else {
				panic("unexpected flag")
			}
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
	configCmd.PersistentFlags().StringP("unset", "u", "", "unset config variable")
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

func ConfigHome() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome
	}

	if home, e := os.UserHomeDir(); e == nil {
		return filepath.Join(home, ".config")
	}

	return ""
}

func getConfig(log *logrus.Logger, bindToConfig bool, newViper func() *viper.Viper, flagCmd *cobra.Command) (v *viper.Viper, path string) {
	v = newViper()
	configHome := ConfigHome()
	var errReading error
	if configHome != "" {
		// {XDG_CONFIG_HOME:-$HOME/.config}/gg/config.toml
		v.AddConfigPath(filepath.Join(configHome, "gg"))
		v.SetConfigName("config")
		v.SetConfigType("toml")
		errReading = v.ReadInConfig()
		if errReading != nil {
			// $HOME/.ggconfig.toml
			if home, e := os.UserHomeDir(); e == nil {
				v = newViper()
				v.AddConfigPath(home)
				v.SetConfigName(".ggconfig")
				v.SetConfigType("toml")
				errReading = v.ReadInConfig()
			}
		}
	}
	if errReading != nil {
		// /etc/ggconfig.toml
		v = viper.New()
		v.AddConfigPath("/etc/")
		v.SetConfigName("ggconfig")
		v.SetConfigType("toml")
		errReading = v.ReadInConfig()
	}
	if errReading == nil {
		log.Tracef("Using config file: %v", v.ConfigFileUsed())
	} else if errReading != nil {
		switch errReading.(type) {
		default:
			log.Fatalf("Fatal error loading config file: %s: %s", v.ConfigFileUsed(), errReading)
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
		// no config file found
		if configHome == "" {
			return v, filepath.Join("/etc/ggconfig.toml")
		}
		return v, filepath.Join(configHome, "gg", "config.toml")
	}
	return v, v.ConfigFileUsed()
}
