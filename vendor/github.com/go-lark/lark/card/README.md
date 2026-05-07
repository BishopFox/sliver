# Card Builder

Interactive card is rich in formats. However, it takes much efforts to build one. Thus, we provide a declarative card builder to make it easier.

We map every element of card (`div`, `text`, `button`, etc.) to declarative calls as shown in the demo.

All inner elements (e.g. `fields` of `div` blocks) are considered as arguments,
while all element properties (e.g. `forward` property of `card` blocks) are considered as chained calls.

## Getting Started

```go
b := NewCardBuilder()
c := b.Card(
    b.Markdown("some text"),
).
    Title("title").
    NoForward()

fmt.Println(c.String())
```

which will render as following:

```json
{
  "config": {
    "wide_screen_mode": true,
    "enable_forward": false
  },
  "header": {
    "title": {
      "tag": "plain_text",
      "content": "title"
    }
  },
  "elements": [
    {
      "tag": "markdown",
      "content": "some text"
    }
  ]
}
```

Example: [examples/interactive-message](https://github.com/go-lark/examples/tree/main/interactive-message)

## I18N

Card with I18N support is also supported. We introduce a `I18N` struct to build localizable parts (e.g. title and contents):

```go
b := lark.NewCardBuilder()
c := b.I18N.Card(
        b.I18N.WithLocale(
            LocaleEnUS,
            b.Div(
                b.Field(b.Text("English Content")),
            ),
        ),
        b.I18N.WithLocale(
            LocaleZhCN,
            b.Div(
                b.Field(b.Text("中文内容")),
            ),
        ),
    ).
    Title(
        b.I18N.LocalizedText(LocaleEnUS, "English Title"),
        b.I18N.LocalizedText(LocaleZhCN, "中文标题"),
    )
```

## Sending Message

```go
b := lark.NewCardBuilder()
card := b.Card(
    b.Div(
        b.Field(b.Text("Content")).Short(),
    ),
).
    Wathet().
    Title("Card Title")
msg := lark.NewMsgBuffer(lark.MsgInteractive)
om := msg.BindEmail("youremail@example.com").Card(card.String()).Build()
resp, err := bot.PostMessage(om)
```
