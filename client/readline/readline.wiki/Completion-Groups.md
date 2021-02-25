
## Group types

Similarly to the Z-Shell completion engine, readline supports 3 different types of completions.
For each group added to the completions, the engine will print its title if the group's `Name` 
field is not nil, and below it, the candidate strings.

#### Grid 
This type is mostly used for items like directory/file paths,  or some command and option arguments.

![tab-grid](https://github.com/bishopfox/sliver/client/readline/blob/assets/grid.gif)

#### Lists 
This mode allows you to add descriptions to each item, as well as an optional alias. The example below
shows such a list, in command options are completed, with some of them having a short -f flag.
![tab-list](https://github.com/bishopfox/sliver/client/readline/blob/assets/list.gif)

#### Maps
- This type is visually identical to the List type, but which the description is the actual candidate. 
This is used for things like history lines. To be noted, declaring such ones works **the same way** as for Maps.
![tab-map](https://github.com/bishopfox/sliver/client/readline/blob/assets/map.gif)


## Stacking completion groups

As you can see in the return values of the `TabCompleter` function [here](https://github.com/bishopfox/sliver/client/readline/completers/tab-completer.go), 
the Completion engine accepts lists of completion groups: it will combine them for printing, search and navigation, respecting the order in which
they are in the completion groups list. If you are readling this, it most certainly means you want to write a completer. 
Please see this as an overview, and come back to it later when you'll have read the docs for 
[writing a completion function](https://github.com/maxlandon/Writing-A-Completer).

Such an example:
![multiple-groups](https://github.com/bishopfox/sliver/client/readline/blob/assets/stacking-completions.gif)


## Limiting the Group printing length

Because your completers or their commands might produce massive amounts of completions (OS processes are a good example),
you may wish to limit the size of candidates being printed at once.  The default is 20 when more than 5 groups are passed 
to the engine (which means 20 lines, not 20 items -would be unfair for grids-). Normally these defaults play nicely.

You can specify this length by setting the groups `MaxLength`.
See the GIF in **Stacking Completions** for a demonstration of the effect.

**Important Warning:**
- `Grid`: There is no rollout for this group, the remaining items will just be ignored for printing (they are still here in some sense).
- `List`: If the group includes alternative options, rollout is disabled due the implementation. Normally this should not a problem, because you 
rarely end up having many different groups of options for a command, and each of theses groups will never be more than 10-15 options long.
- `Map`: Works normally.


## Populating the Completions list 

Below are examples on how to populate a group of completion candidates, for each of the types:

#### Grid 

```go
var grid = &readline.CompletionGroup {
        DisplayType: readline.TabDisplayGrid,                                // Set the type of completion group we want
        Name:        "files",                                                // Will appear on top of the group if not nil.
        Suggestions: []string{"/home", "/path/to/dir", "/path/to/file.ext"}, // Candidates
        TrimSlash:   true,                                                   // Used for directories. Please see Search & Navigation
        MaxLength:   10,                                                     // Maximum 10 lines of completions
}

completions = append(completions, grid) // Adds up to any existing list of completion groups to be passed to the shell.
```

#### Lists 

Note that usually the list would be built in a for loop, so that the mapping for each candidate will be clearer.

```go
var optionsList = &readline.CompletionGroup{
        DisplayType: readline.TabDisplayList,                                  
        Name: "command options",
        Description: "a group of command options"                                // Currently not printed anywhere
        Suggestions: []string{"--force", "--long-option", "--remove", "--save"}, // Normal candidates
        Aliases: []string{"-f", "", "-l", "-r", "-s"},                           // Alternative candidates, printed besides.
        Descriptions: map[string]string{                                         // Descriptions are printed besides.
                "--force":          "force the command action",
                "--long-option":    "description for a long option",
                "--remove":         "remove the argument",
                "--save":           "save the argument to file",
        }
        MaxLength: 10 // Just here to mention that the engine will not take it into account: we have aliases.
}

```

#### Maps
```go
var commandHistory = &readline.CompletionGroup{
        DisplayType: readline.TabDisplayMap,                                    
        Name: "command history",
        Suggestions: []string{"ls -al /path", "command subcommand --options", "another command"}
        Descriptions: "1", "2", "3",            // You could replace with anything: a context string, a timestamp, whatever
        MaxLength: 20,                          // Works correctly
}
```
