module github.com/bishopfox/sliver

go 1.16

// fix wgctrl requiring old wireguard
replace golang.zx2c4.com/wireguard => golang.zx2c4.com/wireguard v0.0.0-20210311162910-5f0c8b942d93

require (
	github.com/AlecAivazis/survey/v2 v2.2.2
	github.com/Binject/binjection v0.0.0-20200705191933-da1a50d7013d
	github.com/Binject/debug v0.0.0-20210225042342-c9b8b45728d2
	github.com/BurntSushi/xgb v0.0.0-20201008132610-5f9e7b3c49cd // indirect
	github.com/Microsoft/go-winio v0.4.16
	github.com/alecthomas/chroma v0.8.1
	github.com/binject/go-donut v0.0.0-20201215224200-d947cf4d090d
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/desertbit/closer/v3 v3.1.2 // indirect
	github.com/desertbit/columnize v2.1.0+incompatible
	github.com/desertbit/grumble v1.0.8
	github.com/fatih/color v1.10.0
	github.com/gen2brain/shm v0.0.0-20200228170931-49f9650110c5 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/kbinani/screenshot v0.0.0-20191211154542-3a185f1ce18f
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mattn/go-sqlite3 v1.14.5
	github.com/miekg/dns v1.1.35
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.6.1
	golang.org/x/crypto v0.0.0-20210317152858-513c2a44f670
	golang.org/x/net v0.0.0-20210316092652-d523dce5a7f4
	golang.org/x/sys v0.0.0-20210320140829-1e4c9ba3b0c4
	golang.zx2c4.com/wireguard v0.0.20200121
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b
	google.golang.org/grpc v1.36.0-dev.0.20210208035533-9280052d3665
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/postgres v1.0.5
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.20.6
	inet.af/netstack v0.0.0-20210317161235-a1bf4e56ef22
)
