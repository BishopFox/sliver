# lmorg/readline

## Examples: 01 - Getting Started

This example shows how to create a console application which uses `readline`
for input and tab-completion at it's most basic level.

To use these examples you can just `go build ./main.go` (while inside this
directory) or create your own project using the code demonstrated here.

### Create a list of tab-completion suggestions

First we need to create a slice of strings which `readline` will use as part of
it's tab-complete suggestions:

```go
var items = []string{
	"hello",
	"world",
}
```

Then we can create a handler function for `readline` to call which will return
this slice we've just created:

```go
func Tab(line []rune, pos int) (string, []string, map[string]string, readline.TabDisplayType) {
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
	// `suggestions` is the tab-completion suggestions slice we created above.
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
```

### Create a `readline` instance

Lastly lets now create the main function which will instantiate `readline` and
print it's returns back to the console:


```go
func main() {
	// Create a new readline instance
	rl := readline.NewInstance()

	// Attach the tab-completion handler (function defined below)
	rl.TabCompleter = Tab

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

		// Print the returned line from readline
		fmt.Println("Readline: '" + line + "'")
	}
}
```

### Developing this example further

The way the `Tab()` function is handled in `readline`'s sister project, _murex_,
is rather than pulling from a pre-defined slice of suggestions, the `Tab()`
function dynamically builds the suggestions based on what text preceded it.
For example, if `cd <tab>` was typed then `Tab()` would return a list of
directories. If `vi <tab>` was typed then `Tab()` would return a list of files.
Or if `git <tab>` was typed then `Tab()` would return a list of parameters for
`git` CLI.

## License Information

All examples are unlicensed - they belong to the public domain. So you can use
these examples in any which way you choose in any project with any licence and
without attribution. However the `readline` package itself is covered under a
difference license. Please see the LICENSE file under the root directory of
this repository for details about `readline`'s license.
