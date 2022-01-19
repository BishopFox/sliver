module github.com/bishopfox/sliver

go 1.17

// fix wgctrl requiring old wireguard
replace golang.zx2c4.com/wireguard => golang.zx2c4.com/wireguard v0.0.0-20210311162910-5f0c8b942d93

replace github.com/desertbit/grumble v1.1.1 => github.com/moloch--/grumble v1.1.4

require (
	github.com/AlecAivazis/survey/v2 v2.2.2
	github.com/Binject/binjection v0.0.0-20200705191933-da1a50d7013d
	github.com/Binject/debug v0.0.0-20210312092933-6277045c2fdf
	github.com/Binject/go-donut v0.0.0-20210701074227-67a31e2d883e
	github.com/Binject/universal v0.0.0-20210304094126-daefaa886313
	github.com/BurntSushi/xgb v0.0.0-20201008132610-5f9e7b3c49cd // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/alecthomas/chroma v0.8.1
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/desertbit/columnize v2.1.0+incompatible
	github.com/desertbit/go-shlex v0.1.1
	github.com/desertbit/grumble v1.1.1
	github.com/fatih/color v1.12.0
	github.com/gen2brain/shm v0.0.0-20200228170931-49f9650110c5 // indirect
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jedib0t/go-pretty/v6 v6.2.4
	github.com/kbinani/screenshot v0.0.0-20191211154542-3a185f1ce18f
	github.com/lesnuages/go-winio v0.4.19
	github.com/lesnuages/snitch v0.6.0
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mattn/go-sqlite3 v1.14.9
	github.com/miekg/dns v1.1.35
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/pquerna/otp v1.3.0
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e
	golang.org/x/sys v0.0.0-20211117180635-dee7805ff2e1
	golang.org/x/text v0.3.6
	golang.zx2c4.com/wireguard v0.0.20200121
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200609130330-bd2cb7843e1b
	google.golang.org/genproto v0.0.0-20210406143921-e86de6bf7a46 // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gorm.io/driver/mysql v1.0.3
	gorm.io/driver/postgres v1.0.5
	gorm.io/driver/sqlite v1.1.3
	gorm.io/gorm v1.21.14
	inet.af/netstack v0.0.0-20210317161235-a1bf4e56ef22
)

require (
	github.com/moloch--/memmod v0.0.0-20211120144554-8b37cc654945
	github.com/shirou/gopsutil/v3 v3.21.10
	github.com/things-go/go-socks5 v0.0.3-0.20210722055343-24af464efe43
	modernc.org/sqlite v1.14.3
)

require (
	github.com/Binject/shellcode v0.0.0-20191101084904-a8a90e7d4563 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VirusTotal/vt-go v0.0.0-20210528074736-45bbe34cc8ab // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/awgh/cppgo v0.0.0-20210224085512-3d24bca8edc0 // indirect
	github.com/awgh/rawreader v0.0.0-20200626064944-56820a9c6da4 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/demisto/goxforce v0.0.0-20160322194047-db8357535b1d // indirect
	github.com/desertbit/closer/v3 v3.1.2 // indirect
	github.com/desertbit/readline v1.5.1 // indirect
	github.com/dlclark/regexp2 v1.2.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.7.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.5 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.5.0 // indirect
	github.com/jackc/pgx/v4 v4.9.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.2 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20200410134404-eec4a21b6bb0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/thedevsaddam/gojsonq/v2 v2.5.2 // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/grpc/examples v0.0.0-20210910232509-03268c8ed29e // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
	lukechampine.com/uint128 v1.1.1 // indirect
	modernc.org/cc/v3 v3.35.18 // indirect
	modernc.org/ccgo/v3 v3.12.95 // indirect
	modernc.org/libc v1.11.104 // indirect
	modernc.org/mathutil v1.4.1 // indirect
	modernc.org/memory v1.0.5 // indirect
	modernc.org/opt v0.1.1 // indirect
	modernc.org/strutil v1.1.1 // indirect
	modernc.org/token v1.0.0 // indirect
)
