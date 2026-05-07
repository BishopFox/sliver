package reddit

import (
	"context"
	"sync"
	"time"
)

// StreamService allows streaming new content from Reddit as it appears.
type StreamService struct {
	client *Client
}

// Posts streams posts from the specified subreddit.
// It returns 2 channels and a function:
//   - a channel into which new posts will be sent
//   - a channel into which any errors will be sent
//   - a function that the client can call once to stop the streaming and close the channels
// Because of the 100 post limit imposed by Reddit when fetching posts, some high-traffic
// streams might drop submissions between API requests, such as when streaming r/all.
func (s *StreamService) Posts(subreddit string, opts ...StreamOpt) (<-chan *Post, <-chan error, func()) {
	streamConfig := &streamConfig{
		Interval:       defaultStreamInterval,
		DiscardInitial: false,
		MaxRequests:    0,
	}
	for _, opt := range opts {
		opt(streamConfig)
	}

	ticker := time.NewTicker(streamConfig.Interval)
	postsCh := make(chan *Post)
	errsCh := make(chan error)

	var once sync.Once
	stop := func() {
		once.Do(func() {
			ticker.Stop()
			close(postsCh)
			close(errsCh)
		})
	}

	// originally used the "before" parameter, but if that post gets deleted, subsequent requests
	// would just return empty listings; easier to just keep track of all post ids encountered
	ids := set{}

	go func() {
		defer stop()

		var n int
		infinite := streamConfig.MaxRequests == 0

		for ; ; <-ticker.C {
			n++

			posts, err := s.getPosts(subreddit)
			if err != nil {
				errsCh <- err
				if !infinite && n >= streamConfig.MaxRequests {
					break
				}
				continue
			}

			for _, post := range posts {
				id := post.FullID

				// if this post id is already part of the set, it means that it and the ones
				// after it in the list have already been streamed, so break out of the loop
				if ids.Exists(id) {
					break
				}
				ids.Add(id)

				if streamConfig.DiscardInitial {
					streamConfig.DiscardInitial = false
					break
				}

				postsCh <- post
			}

			if !infinite && n >= streamConfig.MaxRequests {
				break
			}
		}
	}()

	return postsCh, errsCh, stop
}

func (s *StreamService) getPosts(subreddit string) ([]*Post, error) {
	posts, _, err := s.client.Subreddit.NewPosts(context.Background(), subreddit, &ListOptions{Limit: 100})
	return posts, err
}

type set map[string]struct{}

func (s set) Add(v string) {
	s[v] = struct{}{}
}

func (s set) Delete(v string) {
	delete(s, v)
}

func (s set) Len() int {
	return len(s)
}

func (s set) Exists(v string) bool {
	_, ok := s[v]
	return ok
}
