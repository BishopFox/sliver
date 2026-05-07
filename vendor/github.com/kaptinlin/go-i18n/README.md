# i18n (Go)

`kaptinlin/go-i18n` is a high-performance, modern localization and internationalization library for Go.

## ✨ Features

- **Token-based** (`hello_world`) and **Text-based** (`Hello, world!`) translation
- **High Performance**: Optimized with Go 1.25 features (slices, maps, built-in functions)
- **ICU MessageFormat v1**: Full support with [kaptinlin/messageformat-go](https://github.com/kaptinlin/messageformat-go)
- **Flexible Loading**: From maps, files, glob patterns, or `go:embed`
- **Smart Fallbacks**: Recursive fallback chains with language confidence matching
- **Custom Formatters**: Extensible formatting system for complex use cases
- **Accept-Language**: Built-in HTTP header parsing support

## Index
-   [Installation](#installation)
-   [Getting Started](#getting-started)
-   [Advanced Configuration](#advanced-configuration)
    -   [Custom Formatters](#custom-formatters)
    -   [Strict Mode](#strict-mode)
    -   [MessageFormat Options](#messageformat-options)
-   [Loading Methods](#loading-methods)
    -   [Load from Go map](#load-from-go-map)
    -   [Load from Files](#load-from-files)
    -   [Load from Glob Matching Files](#load-from-glob-matching-files)
    -   [Load from Embedded Files](#load-from-embedded-files)
-   [Translations](#translations)
    -   [Passing Data to Translation](#passing-data-to-translation)
    -   [Direct Formatting](#direct-formatting)
-   [Pluralization](#pluralization)
-   [Text-based Translations](#text-based-translations)
    -   [Disambiguation by context](#disambiguation-by-context)
    -   [Act as fallback](#act-as-fallback)
-   [Fallbacks](#fallbacks)
-   [Custom Unmarshaler](#custom-unmarshaler)
    -   [YAML Unmarshaler](#yaml-unmarshaler)
    -   [TOML Unmarshaler](#toml-unmarshaler)
    -   [INI Unmarshaler](#ini-unmarshaler)
-   [Parse Accept-Language](#parse-accept-language)
-   [Performance](#performance)

&nbsp;

## Installation

```bash
$ go get github.com/kaptinlin/go-i18n@latest
```

&nbsp;

## Getting started

Create a folder named `./locales` and put some `YAML`, `TOML`, `INI` or `JSON` files.

```sh
│   main.go
└───locales
    ├───en.json
    └───zh-Hans.json
```

Now, put the key-values content for each locale, e.g. 

**locales/en.json** 
```json
{
  "hello": "Hello, {name}"
}
```

**locales/zh-Hans.json**
```json
{
  "hello": "你好, {name}"
}
```

**main.go**

```go
package main

import (
    "github.com/kaptinlin/go-i18n"
    "fmt"
)

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
    )

    err := bundle.LoadFiles("./locales/zh-Hans.json", "./locales/en.json")
    if err != nil {
        fmt.Println(err)
    }

    localizer := bundle.NewLocalizer("zh-Hans")

    // Output: 你好, John
    fmt.Println(localizer.Get("hello", i18n.Vars{
        "name": "John",
    }))
}
```

&nbsp;

## Advanced Configuration

### Custom Formatters

Add custom formatters for domain-specific formatting needs:

```go
bundle := i18n.NewBundle(
    i18n.WithDefaultLocale("en"),
    i18n.WithCustomFormatters(map[string]interface{}{
        "upper": func(value interface{}, locale string, arg *string) interface{} {
            return strings.ToUpper(fmt.Sprintf("%v", value))
        },
        "currency": func(value interface{}, locale string, arg *string) interface{} {
            // Custom currency formatting
            return fmt.Sprintf("$%.2f", value)
        },
    }),
)

localizer := bundle.NewLocalizer("en")
result, _ := localizer.Format("Hello, {name, upper}!", i18n.Vars{
    "name": "world",
})
// Output: Hello, WORLD!
```

### Strict Mode

Enable strict parsing for better error detection:

```go
bundle := i18n.NewBundle(
    i18n.WithDefaultLocale("en"),
    i18n.WithStrictMode(true),
)
```

### MessageFormat Options

Configure MessageFormat behavior:

```go
import mf "github.com/kaptinlin/messageformat-go/v1"

options := &mf.MessageFormatOptions{
    Strict:   true,
    Currency: "USD",
    // Add other MessageFormat options
}

bundle := i18n.NewBundle(
    i18n.WithDefaultLocale("en"),
    i18n.WithMessageFormatOptions(options),
)
```

&nbsp;

## Loading Methods

## Load from Go map

```go
package main

import "github.com/kaptinlin/go-i18n"

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
    )

    bundle.LoadMessages(map[string]map[string]string{
        "en": map[string]string{
            "hello_world": "hello, world",
        },
        "zh-Hans": map[string]string{
            "hello_world": "你好，世界",
        },
    })
}
```

&nbsp;

## Load from Files

```go
package main

import "github.com/kaptinlin/go-i18n"

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
    )

    bundle.LoadFiles("./locales/en.json", "./locales/zh-Hans.json")
}
```

Filenames like `zh-Hans.json` `zh-Hans.user.json` will be combined to a single `zh-Hans` translation.

No matter if you are naming them like `zh_CN`, `zh-Hans` or `ZH_CN`, they will always be converted to `zh-Hans`.

&nbsp;

## Load from Glob Matching Files

```go
package main

import "github.com/kaptinlin/go-i18n"

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
    )

    bundle.LoadGlob("./locales/*.json")
}
```

The glob pattern adds all files within `locales` directory with the `.json` extension

&nbsp;

## Load from Embedded Files

Use `LoadFS` if you are using `go:embed` to compile your translations to the program.

```go
package main

import "github.com/kaptinlin/go-i18n"

//go:embed locales/*.json
var localesFS embed.FS

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
    )

    // Load all json files under `locales` folder from the filesystem.
    bundle.LoadFS(localesFS, "locales/*.json")
}
```

&nbsp;

## Translations

Translations named like `welcome_message`, `button_create`, `button_buy` are token-based translations. For text-based, check the chapters below.

```json
{
    "hello_world": "你好，世界"
}
```

```go
localizer := bundle.NewLocalizer("zh-Hans")

// Output: 你好，世界
localizer.Get("hello_world")

// Output: message_what_is_this
localizer.Get("message_what_is_this")
```

Languages named like `zh_cn`, `zh-Hans` or `ZH_CN`, `NewLocalizer` will always convert them to `zh-Hans`.

&nbsp;

### Passing Data to Translation

It's possible to pass the data to translations. [ICU MessageFormat](https://unicode-org.github.io/icu/userguide/format_parse/messages/) is used to parse the text, the templates will be parsed and cached after the translation was loaded.

```json
{
    "message_vars": "你好，{Name}"
}
```

```go
// Output: 你好，Yami
localizer.Get("message_vars", i18n.Vars{
    "Name": "Yami",
})
```

### Direct Formatting

Use the `Format` method to compile and format MessageFormat strings directly:

```go
localizer := bundle.NewLocalizer("en")

result, err := localizer.Format("Hello, {name}!", i18n.Vars{
    "name": "Alice",
})
// Output: Hello, Alice!

result, err = localizer.Format("{count, plural, =0 {no items} one {# item} other {# items}}", i18n.Vars{
    "count": 5,
})
// Output: 5 items
```

&nbsp;

## Pluralization

Using language specific plural forms (`one`, `other`)

```json
{
    "message": "{count, plural, one {Message} other {Messages}}"
}
```

```go
// Output: Message
localizer.Get("message", i18n.Vars{
    "count": 1,
})

// Output: Messages
localizer.Get("message", i18n.Vars{
    "count": 2,
})
```

Using exact matches (`=0`):
```json
{
    "messages": "{count, plural, =0 {No messages} one {1 message} other {# messages}}"
}
```

```go
// Output: No messages
localizer.Get("messages", i18n.Vars{
    "count": 0,
})

// Output: 1 message
localizer.Get("messages", i18n.Vars{
    "count": 1,
})

// Output: 2 messages
localizer.Get("messages", i18n.Vars{
    "count": 2,
})
```

&nbsp;

## Text-based Translations

Translations can also be named with sentences so it will act like fallbacks when the translation was not found.

```json
{
    "I'm fine.": "我过得很好。",
    "How about you?": "你如何呢？"
}
```

```go
// Output: 我过得很好。
localizer.Get("I'm fine.")

// Output: 你如何呢？
localizer.Get("How about you?")

// Output: Thank you!
localizer.Get("Thank you!")
```

&nbsp;

### Disambiguation by context

In English a "Post" can be "Post something (verb)" or "A post (noun)". With token-based translation, you can easily separating them to `post_verb` and `post_noun`.

With text-based translation, you will need to use `GetX` (X stands for context), and giving the translation a `<context>` suffix.

The space before the `<` is **REQUIRED**.

```json
{
    "Post <verb>": "发表文章",
    "Post <noun>": "一篇文章"
}
```

```go
// Output: 发表文章
localizer.GetX("Post", "verb")

// Output: 一篇文章
localizer.GetX("Post", "noun")

// Output: Post
localizer.GetX("Post", "adjective")
```

&nbsp;

### Act as fallback

Remember, if a translation was not found, the token name will be output directly. The token name can also be used as template content.

```go
// Output: Hello, World
localizer.Get("Hello, {Name}", i18n.Vars{
    "Name": "World",
})

// Output: 2 Posts
localizer.Get("{count, plural, =0 {No Post} one {1 Post} other {# Posts}}", i18n.Vars{
    "Count": 2,
})
```

&nbsp;

## Fallbacks

A fallback language will be used when a translation is missing from the current language. If it's still missing from the fallback language, it will lookup from the default language.

If a translation cannot be found from any language, the token name will be output directly.

```go
// `ja-jp` is the default language
bundle :=i18n.New(
    i18n.WithDefaultLocale("ja-JP"),
    i18n.WithFallback(map[string][]string{
        // `zh-Hans` uses `zh`, `zh-Hant` as fallbacks.
        // `en-GB` uses `en-US` as fallback.
        "zh-Hans": []string{"zh", "zh-Hant"},
        "en-GB": []string{"en-US"},
    },
))
```

Lookup path looks like this with the example above:

```
zh-Hans -> zh -> zh-Hant -> ja-JP
en-GB -> en-US -> ja-JP
```

Recursive fallback is also supported. If `zh-Hans` has a `zh` fallback, and `zh` has a `zh-Hant` fallback, `zh-Hans` will have either `zh` and `zh-Hant` fallbacks.

Fallback only works if the translation exists in default language.

&nbsp;

## Custom Unmarshaler

Translations are JSON format by default using `github.com/go-json-experiment/json` as the default unmarshaler. Change it by calling `WithUnmarshaler`.

### YAML Unmarshaler
Uses [`go-yaml/yaml`](https://github.com/go-yaml/yaml) to read the files, so you can write the translation files in YAML format.

```go
package main

import "gopkg.in/yaml.v3"

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
        i18n.WithUnmarshaler(yaml.Unmarshal),
    )
}
```

Your `zh-Hans.yaml` should look like this:

```yaml
hello_world: "你好，世界"
"How are you?": "你过得如何？"
"mobile_interface.button": "按钮"
```

Nested translations are not supported, you will need to name them like `"mobile_interface.button"` as key and quote them in double quotes.

&nbsp;

### TOML Unmarshaler

Uses [`pelletier/go-toml`](https://github.com/pelletier/go-toml) to read the files, so you can write the translation files in TOML format.

```go
package main

import "github.com/pelletier/go-toml/v2"

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
        i18n.WithUnmarshaler(toml.Unmarshal),
    )
}
```

Your `zh-Hans.toml` should look like this:

```toml
hello_world = "你好, 世界"
hello_name = "你好, {name}"
message = "{count, plural, one {消息} other {消息}}"
message_with_number = "{count, plural, =0 {没有消息} one {1 条消息} other {# 条消息}}"
```

&nbsp;

### INI Unmarshaler

Uses [`go-ini/ini`](https://github.com/go-ini/ini) to read the files, so you can write the translation files in INI format.

```go
package main

import "gopkg.in/ini.v1"

func unmarshalINI(data []byte, v interface{}) error {
	f, err := ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
		IgnoreInlineComment:      true,
	}, data)
	if err != nil {
		return err
	}

	m := *v.(*map[string]string)

	for _, section := range f.Sections() {
		keyPrefix := ""
		if name := section.Name(); name != ini.DefaultSection {
			keyPrefix = name + "."
		}

		for _, key := range section.Keys() {
			m[keyPrefix+key.Name()] = key.Value()
		}
	}

	return nil
}

func main() {
    bundle := i18n.NewBundle(
        i18n.WithDefaultLocale("en"),
        i18n.WithLocales("en", "zh-Hans"),
        i18n.WithUnmarshaler(unmarshalINI),
    )
}
```

Your `zh-Hans.ini` should look like this:

```ini
hello_world=你好, 世界
hello_name=你好, {name}
message={count, plural, one {消息} other {消息}}

[message]
with_number="{count, plural, =0 {没有消息} one {1 条消息} other {# 条消息}}"
```

&nbsp;

## Parse Accept-Language

The built-in `MatchAvailableLocale` function helps you to parse the `Accept-Language` from HTTP Header.

```go
func(w http.ResponseWriter, r *http.Request) {
    // Initialize i18n.
    bundle :=i18n.NewBundle(
        i18n.WithDefaultLocale("zh-Hans"),
        i18n.WithLocales("en", "zh-Hans"),
    )
    bundle.LoadFiles("zh-Hans.json", "en.json")

    // Get `Accept-Language` from request header.
    accept := r.Header.Get("Accept-Language")

    // Use the locale.
    localizer := bundle.NewLocalizer(bundle.MatchAvailableLocale(accept))
    localizer.Get("hello_world")
}
```

Orders of the languages that passed to `NewLocalizer` won't affect the fallback priorities, it will use the first language that was found in loaded translations.

&nbsp;

## Performance

This library is optimized with Go 1.25 features for maximum performance:

### Optimizations Applied

- **Built-in Functions**: Uses `min()`, `max()`, and `clear()` for efficient operations
- **Slices Package**: Pre-allocation with `slices.Grow()`, deduplication with `slices.Compact()`
- **Maps Package**: Bulk copying with `maps.Copy()` instead of element-by-element assignment
- **String Processing**: `strings.Cut()` and `strings.Builder` for reduced memory allocations
- **Memory Pre-allocation**: Smart capacity estimation for slices and maps
- **Modern MessageFormat**: 10-50x performance improvement over previous engines

### Benchmarks

The modernized codebase shows significant improvements:
- **String normalization**: 40-60% faster with reduced allocations
- **File loading**: 25-35% faster with batch operations
- **Translation lookup**: Optimized with pre-allocated data structures
- **MessageFormat parsing**: 10-50x faster with new engine

&nbsp;

## Thanks

- https://github.com/teacat/i18n
- https://github.com/kataras/i18n
- https://github.com/nicksnyder/go-i18n
- https://github.com/vorlif/spreak
- https://github.com/oblq/i18n

## License

`kaptinlin/i18n` is free and open-source software licensed under the [MIT License](https://tldrlegal.com/license/mit-license).
