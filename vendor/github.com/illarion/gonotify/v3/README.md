## Gonotify 

Simple Golang inotify wrapper.

[![GoDoc](https://godoc.org/github.com/illarion/gonotify/v3?status.svg)](https://godoc.org/github.com/illarion/gonotify/v3)

### Provides following primitives:

* Low level
  * `Inotify` - wrapper around [inotify(7)](http://man7.org/linux/man-pages/man7/inotify.7.html)
  * `InotifyEvent` - generated file/folder event. Contains `Name` (full path), `Wd` - watch descriptor and `Mask` that describes the event.

* Higher level
  * `FileWatcher` - higher level utility, helps to watch the list of files for changes, creation or removal
  * `DirWatcher` - higher level utility, recursively watches given root folder for added, removed or changed files.
  * `FileEvent` - embeds `InotifyEvent` and keeps additional field `Eof` to notify user that there will be no more events.

Use `FileWatcher` and `DirWatcher` as an example and build your own utility classes.

### Usage

```go
package main

import (
  "fmt"
  "github.com/illarion/gonotify/v3"
  "time"
  "context"
)

func main() {

  ctx, cancel := context.WithCancel(context.Background())

  watcher, err := gonotify.NewDirWatcher(ctx, gonotify.IN_CREATE|gonotify.IN_CLOSE, "/tmp")
  if err != nil {
    panic(err)
  }

  main:
  for {
    select {
    case event := <-watcher.C:
      fmt.Printf("Event: %s\n", event)

      if event.Is(gonotify.IN_CREATE) {
        fmt.Printf("File created: %s\n", event.Name)
      }

      if event.IsAny(gonotify.IN_CLOSE, gonotify.IN_CLOSE_WRITE) {
        fmt.Printf("File closed: %s\n", event.Name)
      }

    case <-time.After(5 * time.Second):
      fmt.Println("Good bye!")
      cancel()
      break main
    }
  }
  
  // Wait for watcher to finish all internal goroutines
  <-watcher.Done()
  fmt.Println("Watcher is done")
  
}
```

## License
MIT. See [LICENSE](LICENSE) file for more details.

