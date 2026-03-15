# JSON Pointer

[![Go Reference](https://pkg.go.dev/badge/github.com/kaptinlin/jsonpointer.svg)](https://pkg.go.dev/github.com/kaptinlin/jsonpointer)
[![Go Report Card](https://goreportcard.com/badge/github.com/kaptinlin/jsonpointer)](https://goreportcard.com/report/github.com/kaptinlin/jsonpointer)

Fast implementation of [JSON Pointer (RFC 6901)][json-pointer] specification in Go.

[json-pointer]: https://tools.ietf.org/html/rfc6901

## Installation

```bash
go get github.com/kaptinlin/jsonpointer
```

## Usage

### Basic Operations

Find a value in a JSON object using a JSON Pointer string:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    doc := map[string]any{
        "foo": map[string]any{
            "bar": 123,
        },
    }

    ref, err := jsonpointer.FindByPointer(doc, "/foo/bar")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(ref.Val) // 123
}
```

### Find by Path Components

Use variadic arguments to navigate to a value:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    doc := map[string]any{
        "foo": map[string]any{
            "bar": 123,
        },
    }

    ref, err := jsonpointer.Find(doc, "foo", "bar")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Value: %v, Object: %v, Key: %v\n", ref.Val, ref.Obj, ref.Key)
    // Value: 123, Object: map[bar:123], Key: bar
}
```

### Safe Get Operations

Get values with error handling:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    doc := map[string]any{
        "users": []any{
            map[string]any{"name": "Alice", "age": 30},
            map[string]any{"name": "Bob", "age": 25},
        },
    }

    // Get existing value using variadic arguments (array indices as strings)
    name, err := jsonpointer.Get(doc, "users", "0", "name")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(name) // Alice

    // Get non-existing value - returns error
    missing, err := jsonpointer.Get(doc, "users", "5", "name")
    if err != nil {
        fmt.Printf("Error: %v\n", err) // Error: array index out of bounds
    } else {
        fmt.Println(missing)
    }
    
    // Get using JSON Pointer string
    age, err := jsonpointer.GetByPointer(doc, "/users/1/age")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(age) // 25
}
```

### Path Manipulation

Convert between JSON Pointer strings and path arrays:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    // Parse JSON Pointer string to path array
    path := jsonpointer.Parse("/f~0o~1o/bar/1/baz")
    fmt.Printf("%+v\n", path)
    // [f~o/o bar 1 baz]

    // Format path components to JSON Pointer string
    pointer := jsonpointer.Format("f~o/o", "bar", "1", "baz")
    fmt.Println(pointer)
    // /f~0o~1o/bar/1/baz
    
    // Performance tip: For repeated access to the same path,
    // pre-parse the pointer once and reuse the path
    userNamePath := jsonpointer.Parse("/users/0/name")
    
    // Efficient repeated access
    for _, data := range datasets {
        name, err := jsonpointer.Get(data, userNamePath...)
        if err != nil {
            log.Printf("Error accessing user name: %v", err)
            continue
        }
        fmt.Println(name)
    }
}
```

### Component Encoding/Decoding

Encode and decode individual path components:

```go
package main

import (
    "fmt"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    // Unescape component
    unescaped := jsonpointer.Unescape("~0~1")
    fmt.Println(unescaped) // ~/

    // Escape component
    escaped := jsonpointer.Escape("~/")
    fmt.Println(escaped) // ~0~1
}
```

### Array Operations

Working with arrays and array indices:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    doc := map[string]any{
        "items": []any{1, 2, 3},
    }

    // Access array element using variadic arguments (index as string)
    ref, err := jsonpointer.Find(doc, "items", "1")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(ref.Val) // 2

    // Using JSON Pointer string with Get
    value, err := jsonpointer.GetByPointer(doc, "/items/0")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(value) // 1

    // Array end marker "-" refers to nonexistent element (returns error)
    // Per RFC 6901, "-" is used for append operations, not for reading
    _, err = jsonpointer.Find(doc, "items", "-")
    if err != nil {
        fmt.Printf("Array end marker error: %v\n", err) // array index out of bounds
    }
}
```

### Struct Operations

Working with Go structs and JSON tags:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/kaptinlin/jsonpointer"
)

type User struct {
    Name    string `json:"name"`
    Age     int    `json:"age"`
    Email   string // No JSON tag, uses field name
    private string // Private field, ignored
    Ignored string `json:"-"` // Explicitly ignored
}

type Profile struct {
    User     *User  `json:"user"` // Pointer to struct
    Location string `json:"location"`
}

func main() {
    profile := Profile{
        User: &User{ // Pointer to struct
            Name:    "Alice",
            Age:     30,
            Email:   "alice@example.com",
            private: "secret",
            Ignored: "ignored",
        },
        Location: "New York",
    }

    // JSON tag access using variadic arguments
    name, err := jsonpointer.Get(profile, "user", "name")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(name) // Alice

    // Field name access (no JSON tag)
    email, err := jsonpointer.Get(profile, "user", "Email")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(email) // alice@example.com

    // Private fields are ignored - returns error
    private, err := jsonpointer.Get(profile, "user", "private")
    if err != nil {
        fmt.Printf("Error: %v\n", err) // Error: struct field not found
    }

    // json:"-" fields are ignored - returns error
    ignored, err := jsonpointer.Get(profile, "user", "Ignored")
    if err != nil {
        fmt.Printf("Error: %v\n", err) // Error: struct field not found
    }

    // Nested struct navigation
    age, err := jsonpointer.Get(profile, "user", "age")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(age) // 30

    // JSON Pointer syntax
    ref, err := jsonpointer.FindByPointer(profile, "/user/name")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(ref.Val) // Alice

    // Mixed struct and map data
    data := map[string]any{
        "profile": profile,
        "meta":    map[string]any{"version": "1.0"},
        "users":   []User{{Name: "Bob", Age: 25}},
    }
    
    // Access struct in map
    location, err := jsonpointer.Get(data, "profile", "location")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(location) // New York
    
    // Access array of structs (index as string)
    userName, err := jsonpointer.Get(data, "users", "0", "name")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(userName) // Bob
}
```

### Validation

Validate JSON Pointer strings:

```go
package main

import (
    "fmt"
    
    "github.com/kaptinlin/jsonpointer"
)

func main() {
    // Valid JSON Pointer
    err := jsonpointer.Validate("/foo/bar")
    if err != nil {
        fmt.Printf("Invalid pointer: %v\n", err)
    } else {
        fmt.Println("Valid pointer")
    }

    // Invalid JSON Pointer
    err = jsonpointer.Validate("foo/bar") // missing leading slash
    if err != nil {
        fmt.Printf("Invalid pointer: %v\n", err)
    } else {
        fmt.Println("Valid pointer")
    }
}
```

### Performance

This library offers excellent performance with zero-allocation `Get` operations and competitive `Find` operations. Our `Get` function achieves optimal performance for common use cases, while `Find` provides rich reference objects when needed.

For detailed benchmark results and performance comparisons with other JSON Pointer libraries, see [benchmarks/README.md](benchmarks/README.md).

## Acknowledgments

This project is a Go port of the excellent [jsonjoy-com/json-pointer](https://github.com/jsonjoy-com/json-pointer) TypeScript implementation. We've adapted the core algorithms and added Go-specific performance optimizations while maintaining full RFC 6901 compatibility.

Special thanks to the original json-pointer project for providing a solid foundation and comprehensive test cases that enabled this high-quality Go implementation.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 
