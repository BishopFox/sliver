// Package lscolors provides styling of filenames based on file features.
//
// This is a reverse-engineered implementation of the parsing and
// interpretation of the LS_COLORS environmental variable used by GNU
// coreutils.
package lscolors

import (
	"path"
	"strings"
	"sync"
)

// Colorist styles filenames based on the features of the file.
type Colorist interface {
	// GetStyle returns the style for the named file.
	GetStyle(fname string) string
	// GetStyle returns the style for the named file only by its extension.
	GetStyleExt(fname string) string
}

type colorist struct {
	styleForFeature map[feature]string
	styleForExt     map[string]string
}

const defaultLsColorString = `rs=:di=01;34:ln=01;36:mh=:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.Z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.jpg=01;35:*.jpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.axv=01;35:*.anx=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=36:*.au=36:*.flac=36:*.mid=36:*.midi=36:*.mka=36:*.mp3=36:*.mpc=36:*.ogg=36:*.ra=36:*.wav=36:*.axa=36:*.oga=36:*.spx=36:*.xspf=36:`

var (
	lastColorist      *colorist
	lastColoristMutex sync.Mutex
	lastLsColors      string
)

func init() {
	lastColorist = parseLsColor(defaultLsColorString)
}

func GetColorist(lsColorString string) Colorist {
	lastColoristMutex.Lock()
	defer lastColoristMutex.Unlock()

	s := getLsColors(lsColorString)
	if lastLsColors != s {
		lastLsColors = s
		lastColorist = parseLsColor(s)
	}
	return lastColorist
}

func getLsColors(lsColorString string) string {
	if len(lsColorString) == 0 {
		return defaultLsColorString
	}
	return lsColorString
}

var featureForName = map[string]feature{
	"rs": featureRegular,
	"di": featureDirectory,
	"ln": featureSymlink,
	"mh": featureMultiHardLink,
	"pi": featureNamedPipe,
	"so": featureSocket,
	"do": featureDoor,
	"bd": featureBlockDevice,
	"cd": featureCharDevice,
	"or": featureOrphanedSymlink,
	"su": featureSetuid,
	"sg": featureSetgid,
	"ca": featureCapability,
	"tw": featureWorldWritableStickyDirectory,
	"ow": featureWorldWritableDirectory,
	"st": featureStickyDirectory,
	"ex": featureExecutable,
}

// parseLsColor parses a string in the LS_COLORS format into lsColor. Erroneous
// fields are silently ignored.
func parseLsColor(s string) *colorist {
	lc := &colorist{make(map[feature]string), make(map[string]string)}
	for _, spec := range strings.Split(s, ":") {
		words := strings.Split(spec, "=")
		if len(words) != 2 {
			continue
		}
		key, value := words[0], words[1]
		filterValues := []string{}
		for _, splitValue := range strings.Split(value, ";") {
			if strings.Count(splitValue, "0") == len(splitValue) {
				continue
			}
			filterValues = append(filterValues, splitValue)
		}
		if len(filterValues) == 0 {
			continue
		}
		value = strings.Join(filterValues, ";")
		if strings.HasPrefix(key, "*.") {
			lc.styleForExt[key[1:]] = value
		} else {
			feature, ok := featureForName[key]
			if !ok {
				continue
			}
			lc.styleForFeature[feature] = value
		}
	}
	return lc
}

func (lc *colorist) GetStyle(fname string) string {
	mh := strings.Trim(lc.styleForFeature[featureMultiHardLink], "0") != ""
	// TODO Handle error from determineFeature
	feature, _ := determineFeature(fname, mh)
	if feature == featureRegular {
		if ext := path.Ext(fname); ext != "" {
			if style, ok := lc.styleForExt[ext]; ok {
				return style
			}
		}
	}
	return lc.styleForFeature[feature]
}

func (lc *colorist) GetStyleExt(fname string) string {
	if !strings.HasSuffix(fname, "/") {
		if ext := path.Ext(fname); ext != "" {
			if style, ok := lc.styleForExt[ext]; ok {
				return style
			}
		}
	}
	return lc.styleForFeature[featureDirectory]
}
