# Closer - A simple thread-safe closer

[![GoDoc](https://godoc.org/github.com/desertbit/closer?status.svg)](https://godoc.org/github.com/desertbit/closer)
[![Go Report Card](https://goreportcard.com/badge/github.com/desertbit/closer)](https://goreportcard.com/report/github.com/desertbit/closer)

```go
type Server struct {
    closer.Closer // Embedded
}

func New() *Server {
    return &Server {
        Closer: closer.New(),
    }
}

func main() {
    s := New()

    // Do something on close...
    go func() {
        <-s.CloseChan
        // ...
    } ()

    // ...
    s.Close()
}
```

```go
type Server struct {
    closer.Closer // Embedded
    conn net.Conn
}

func New() *Server {
    // ...
    s := &Server {
        conn: conn,
    }
    s.Closer = closer.New(s.onClose)
    return s
}

func (s *server) onClose() error {
    return s.conn.Close()
}

func main() {
    s := New()
    // ...

    // The s.onClose function will be called only once.
    s.Close()
    s.Close()
}
```
