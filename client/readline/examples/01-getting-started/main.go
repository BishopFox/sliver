package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/maxlandon/readline"
)

// This is an example program using readline at a basic level. Please feel free
// to hack this around - change values here and there - to get an understanding
// of how the readline API works.
//
// This example covers tab-completion. Other - as of yet unwritten - examples
// will cover other features of readline.

func main() {
	// Create a new readline instance
	rl := readline.NewInstance()

	// Attach the tab-completion handler (function defined below)
	rl.TabCompleter = Tab

	// Set multiline prompt and Vim status in it
	rl.Multiline = true                                  // Two-line prompt
	rl.SetPrompt("@localhost => exploit(multi/handler)") // Sets the first line
	rl.ShowVimMode = true                                // Sets the second line to Vim

	for {
		// Call readline - which will put the terminal into a pseudo-raw mode
		// and then read from STDIN. After the user has hit <ENTER> the terminal
		// is put back to it's original mode.
		//
		// In this example, `line` is a returned string of the key presses
		// typed into readline.

		line, err := rl.Readline()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if line == "refresh_delay" {
			go func() {
				time.Sleep(time.Second)
				rl.HideNextPrompt = true
				time.Sleep(time.Second * 4)
				rl.RefreshMultiline("@localhost => payload(ghost/multi/stager/HTTP)", 4, true) // Sets the first line
			}()
		}

		// Print the returned line from readline
		fmt.Println("Readline: '" + line + "'")
	}
}

// Tab is the tab-completion handler for this readline example program
func Tab(input []rune, pos int) (line string, groups []*readline.CompletionGroup) {

	var core readline.CompletionGroup
	core = coreCommands
	groups = append(groups, &core)

	var addrs readline.CompletionGroup
	addrs = addresses
	groups = append(groups, &addrs)

	var opts readline.CompletionGroup
	opts = options
	groups = append(groups, &opts)

	var admin readline.CompletionGroup
	admin = adminCommands
	groups = append(groups, &admin)

	var procs readline.CompletionGroup
	procs = processes
	groups = append(groups, &procs)

	var oth readline.CompletionGroup
	oth = other
	groups = append(groups, &oth)

	return string(input[:pos]), groups
}

// AltTab is the tab-completion handler for this readline example program
func AltTab(line []rune, pos int) (string, []string, map[string]string, readline.TabDisplayType) {
	var suggestions []string

	for i := range items {
		// Since in this example we don't want all items to be suggested, only
		// those that we have already started typing, lets build a new slice
		// from `items` with the matched suggestions
		if strings.HasPrefix(items[i], string(line)) {
			// The append that happens here should be mostly self explanatory
			// however there is one surprise in that we are also cropping the
			// string. Basically readline will output the completion suggestions
			// verbatim. This means if your user types "foo" and your suggestion
			// is "foobar" then the result returned will be "foofoobar". So you
			// need to crop the partial string from the suggestions.
			//
			// I do admit this is a rather ugly and solution. In all honesty I
			// don't like this approach much myself however that seems to be
			// how the existing readline APIs function (in Go at least) and thus
			// I wanted to keep compatibility with them when I started writing
			// this alternative. This function has since diverged from them in
			// other ways as I've added more features but I've left this
			// particular anti-pattern in for the sake of minimizing breaking
			// changes. That all said, I fully expect that there might be some
			// weird edge case scenarios where this approach might be required
			// by whoever picks this package up as they might need some more
			// complex completion logic than what I used this for.
			suggestions = append(suggestions, items[i][pos:])
		}
	}

	// `line[:pos]` is a string to prefix the tab-completion suggestions. For
	// most use cases this will be the string you're literally just cropped out
	// in the `items[i][pos:]` part of the `append` above. While this does seem
	// unduly complicated and pointless, there may be some instances where this
	// proves a useful feature (for example where the tab-completion suggestion
	// needs to differ from what value it returns when selected). It is also
	// worth noting that any value you enter here will not be entered on to the
	// interactive line you're typing when the suggestion is selected. ie this
	// string is a prefix purely for display purposes.
	//
	// `suggestions` is clearly the tab-completion suggestions slice we created
	// above.
	//
	// I agree having a `nil` in a return is ugly. The rational is you can have
	// one single tab handler that can return either a slice of suggestions or
	// a map (eg when you want a description with the suggestion) and can do so
	// with compile type checking intact (ie had I used an interface{} for the
	// suggestion return). This example doesn't make use of that feature.
	//
	// `TabDisplayGrid` is the style to output the tab-completion suggestions.
	// The grid display is the normal default to use when you don't have
	// descriptions. I will cover the other display formats in other examples.
	return string(line[:pos]), suggestions, nil, readline.TabDisplayGrid
}

// ----------------------------------------- COMPLETION ITEMS -------------------------------------------

var coreCommands = readline.CompletionGroup{
	Name:        "core commands",
	Description: "All commands for core console usage",
	Suggestions: []string{"exit", "compiler", "jobs", "module", "db", "server"},
	Descriptions: map[string]string{"exit": "Quit the console",
		"compiler": "Enter the compiler menu, for implant setup and compilation",
		"jobs":     "Asynchronous server jobs",
		"module":   "Use a module (post, handler, route, exploit)",
		"db":       "Database commands and queries",
		"server":   "Server commands, for requiring editing"},
	// MaxLength:   4,
	DisplayType: readline.TabDisplayList,
}

var adminCommands = readline.CompletionGroup{
	Name:        "administrator commands",
	Description: "All commands for managing permissions and other users",
	Suggestions: []string{"multiplayer", "delete_user", "jobs", "chown", "chmod", "server"},
	Descriptions: map[string]string{
		"multiplayer": "Start multiplayer mode on this server",
		"delete_user": "Delete a user from the server",
		"jobs":        "Asynchronous server jobs",
		"chown":       "Change owner of an implant",
		"chmod":       "Change permissions for an implant",
		"server":      "Server admin commands, for requiring editing"},
	MaxLength:   3,
	DisplayType: readline.TabDisplayMap,
}

var processes = readline.CompletionGroup{
	Name:        "target processes",
	Description: "All processes running on the target",
	Suggestions: []string{"chromium", "init", "/bin/sh", "/usr/bin/gofmt", "/usr/bin/nmap"},
	Descriptions: map[string]string{
		"add_user":      "Enter the compiler menu, for implant setup and compilation",
		"delete_user":   "Asynchronous server jobs",
		"chown":         "Change owner of an implant",
		"chmod":         "Change permissions for an implant",
		"/usr/bin/nmap": ""},
	MaxLength:   20,
	DisplayType: readline.TabDisplayGrid,
}

var addresses = readline.CompletionGroup{
	Name:        "neighbour network addresses",
	Description: "All addresses known on this particular implant subnet",
	Suggestions: []string{"http://github.com/target.com",
		"mtls://192.168.2.2:443",
		"dns://122.223.234.45:53",
		"http://www.target.com/usercontent/hole",
		"socks5://23.245.53.932:8888"},
	MaxLength:   20,
	DisplayType: readline.TabDisplayGrid,
}

var options = readline.CompletionGroup{
	Name:        "long/short options",
	Description: "All addresses known on this particular implant subnet",
	Suggestions: []string{"--protocol",
		"--direction",
		"--reverse",
		"--lhost",
		"--forwarder",
		"--session-id",
		"--id",
		"--exploit",
		"--proxy",
		"--close-conns"},
	Descriptions: map[string]string{
		"--protocol":    "Transport protocol to use",
		"--direction":   "Direction of the forwarder",
		"--reverse":     "Start reverse",
		"--lhost":       "Host to reach back",
		"--close-conns": "Close active connections",
	},
	SuggestionsAlt: map[string]string{
		"--protocol":    "-p",
		"--reverse":     "-r",
		"--close-conns": "-c",
		"--session-id":  "-s",
		"--proxy":       "-x",
		"--id":          "-i",
	},
	MaxLength:   5,
	DisplayType: readline.TabDisplayList,
}

var other = readline.CompletionGroup{
	Name:        "other",
	Description: "Other names completed",
	Suggestions: items,
	DisplayType: readline.TabDisplayGrid,
}

// items is an example list of possible suggestions to display in readline's
// tab-completion. For the perpose of this example, I basically just grabbed
// a few entries from some random dictionary of terms.
var items = []string{
	"abaya",
	"abomasum",
	"absquatulate",
	"adscititious",
	"afreet",
	"Albertopolis",
	"alcazar",
	"amphibology",
	"amphisbaena",
	"anfractuous",
	"anguilliform",
	"apoptosis",
	"apple-knocker",
	"argle-bargle",
	"Argus-eyed",
	"argute",
	"ariel",
	"aristotle",
	"aspergillum",
	"astrobleme",
	"Attic",
	"autotomy",
	"badmash",
	"bandoline",
	"bardolatry",
	"Barmecide",
	"barn",
	"bashment",
	"bawbee",
	"benthos",
	"bergschrund",
	"bezoar",
	"bibliopole",
	"bichon",
	"bilboes",
	"bindlestiff",
	"bingle",
	"blatherskite",
	"bleeding",
	"blind",
	"bobsy-die",
	"boffola",
	"boilover",
	"borborygmus",
	"breatharian",
	"Brobdingnagian",
	"bruxism",
	"bumbo",
}
