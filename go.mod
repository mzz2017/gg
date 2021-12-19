module github.com/mzz2017/gg

go 1.17

require (
	github.com/1lann/promptui v0.0.0-20201231203810-3d80f6bc68f3
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn v1.1.10
	github.com/fatih/structs v1.1.0
	github.com/gorilla/websocket v1.4.2
	github.com/json-iterator/go v1.1.12
	github.com/pelletier/go-toml v1.9.4
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	github.com/v2rayA/shadowsocksR v1.0.4
	golang.org/x/net v0.0.0-20210903162142-ad29c8ab022f
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881
	golang.org/x/tools v0.1.5
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	h12.io/socks v1.0.3
	inet.af/netaddr v0.0.0-20211027220019-c74959edd3b6
)

require (
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e // indirect
	github.com/dgryski/go-camellia v0.0.0-20191119043421-69a8a13fb23d // indirect
	github.com/dgryski/go-idea v0.0.0-20170306091226-d2fb45a411fb // indirect
	github.com/dgryski/go-metro v0.0.0-20200812162917-85c65e2d0165 // indirect
	github.com/dgryski/go-rc2 v0.0.0-20150621095337-8a9021637152 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/juju/ansiterm v0.0.0-20180109212912-720a0952cc2a // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lunixbochs/vtclean v0.0.0-20180621232353-2d01aacdc34a // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mzz2017/disk-bloom v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/seiflotfy/cuckoofilter v0.0.0-20201222105146-bc6005554a0c // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/yl2chen/cidranger v1.0.2 // indirect
	gitlab.com/yawning/chacha20.git v0.0.0-20190903091407-6d1cb28dc72c // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20211027215541-db492cf91b37 // indirect
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/text v0.3.6 // indirect
	gopkg.in/ini.v1 v1.63.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

//replace github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn => ../BitterJohn

replace github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn => github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn v0.0.0-20211219131341-01bccf4da3a0

replace (
	github.com/spf13/cobra => github.com/mzz2017/cobra v0.0.0-20211205075040-2b7f80d9e0b4
	github.com/spf13/pflag => github.com/mzz2017/pflag v0.0.0-20211204030847-74e9419ee6b3
)
