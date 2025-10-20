package gonotify

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// DirWatcher recursively watches the given root folder, waiting for file events.
// Events can be masked by providing fileMask. DirWatcher does not generate events for
// folders or subfolders.
type DirWatcher struct {
	C    chan FileEvent
	done chan struct{}
}

// NewDirWatcher creates DirWatcher recursively waiting for events in the given root folder and
// emitting FileEvents in channel C, that correspond to fileMask. Folder events are ignored (having IN_ISDIR set to 1)
func NewDirWatcher(ctx context.Context, fileMask uint32, root string) (*DirWatcher, error) {
	dw := &DirWatcher{
		C:    make(chan FileEvent),
		done: make(chan struct{}),
	}

	i, err := NewInotify(ctx)
	if err != nil {
		return nil, err
	}

	queue := make([]FileEvent, 0, 100)

	err = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {

		if err != nil {
			return nil
		}

		if !f.IsDir() {

			//fake event for existing files
			queue = append(queue, FileEvent{
				InotifyEvent: InotifyEvent{
					Name: path,
					Mask: IN_CREATE,
				},
			})

			return nil
		}
		_, err = i.AddWatch(path, IN_ALL_EVENTS)
		return err
	})

	if err != nil {
		return nil, err
	}

	events := make(chan FileEvent)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, event := range queue {
			select {
			case <-ctx.Done():
				close(events)
				return
			case events <- event:

			}
		}
		queue = nil

		for {

			select {
			case <-ctx.Done():
				close(events)
				return
			default:
			}

			raw, err := i.Read()
			if err != nil {
				close(events)
				return
			}

			select {
			case <-ctx.Done():
				close(events)
				return
			default:
			}

			for _, event := range raw {

				// Skip ignored events queued from removed watchers
				if event.Mask&IN_IGNORED == IN_IGNORED {
					continue
				}

				// Add watch for folders created in watched folders (recursion)
				if event.Mask&(IN_CREATE|IN_ISDIR) == IN_CREATE|IN_ISDIR {

					// After the watch for subfolder is added, it may be already late to detect files
					// created there right after subfolder creation, so we should generate such events
					// ourselves:
					filepath.Walk(event.Name, func(path string, f os.FileInfo, err error) error {
						if err != nil {
							return nil
						}

						if !f.IsDir() {
							// fake event, but there can be duplicates of this event provided by real watcher
							select {
							case <-ctx.Done():
								return nil
							case events <- FileEvent{
								InotifyEvent: InotifyEvent{
									Name: path,
									Mask: IN_CREATE,
								},
							}: //noop
							}
						}

						return nil
					})

					// Wait for further files to be added
					i.AddWatch(event.Name, IN_ALL_EVENTS)

					continue
				}

				// Remove watch for deleted folders
				if event.Mask&IN_DELETE_SELF == IN_DELETE_SELF {
					i.RmWd(event.Wd)
					continue
				}

				// Skip sub-folder events
				if event.Mask&IN_ISDIR == IN_ISDIR {
					continue
				}

				select {
				case <-ctx.Done():
					return
				case events <- FileEvent{
					InotifyEvent: event,
				}: //noop
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(dw.C)

		for {
			select {
			case <-ctx.Done():
				// drain events
				for {
					select {
					case _, ok := <-events:
						if !ok {
							return
						}
					default:
						return
					}
				}
			case event, ok := <-events:
				if !ok {
					select {
					case <-ctx.Done():
					case dw.C <- FileEvent{
						Eof: true,
					}:
					}
					return
				}

				// Skip events not conforming with provided mask
				if event.Mask&fileMask == 0 {
					continue
				}

				select {
				case dw.C <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	go func() {
		wg.Wait()
		<-i.Done()
		close(dw.done)
	}()

	return dw, nil
}

// Done returns a channel that is closed when DirWatcher is done
func (dw *DirWatcher) Done() <-chan struct{} {
	return dw.done
}
