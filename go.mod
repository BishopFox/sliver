module github.com/bishopfox/sliver

go 1.16

// fix wgctrl requiring old wireguard
replace golang.zx2c4.com/wireguard => golang.zx2c4.com/wireguard v0.0.0-20210311162910-5f0c8b942d93

require (
	github.com/AlecAivazis/survey/v2 v2.2.2
	github.com/Binject/binjection v0.0.0-20200705191933-da1a50d7013d
	github.com/Binject/debug v0.0.0-20210225042342-c9b8b45728d2
	github.com/BurntSushi/xgb v0.0.0-20201008132610-5f9e7b3c49cd // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667 // indirect
	github.com/alecthomas/chroma v0.8.1
	github.com/binject/go-donut v0.0.0-20201215224200-d947cf4d090d
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/evilsocket/islazy v1.10.6
	github.com/fatih/color v1.10.0 // indirect
	github.com/gen2brain/shm v0.0.0-20200228170931-49f9650110c5 // indirect
	github.com/go-cmd/cmd v1.3.0
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/jessevdk/go-flags v1.5.0
	github.com/kbinani/screenshot v0.0.0-20191211154542-3a185f1ce18f
	github.com/lesnuages/go-socks5 v0.0.0-20210409090601-adbe23bd0194
	github.com/lesnuages/go-winio v0.4.19
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mattn/go-sqlite3 v1.14.5
	github.com/maxlandon/gonsole v0.0.0-20210502160124-377d77a27617
	github.com/maxlandon/readline v0.1.0-beta.0.20210502150652-b9c641193f54
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.35
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20210317152858-513c2a44f670
	golang.org/x/net v0.0.0-20210410081132-afb366fc7cd1
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887
	golang.zx2c4.com/wireguard v0.0.20200121
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46 // indirect
	google.golang.org/grpc v1.37.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/postgres v1.0.5
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.20.6
	inet.af/netstack v0.0.0-20210317161235-a1bf4e56ef22
)
