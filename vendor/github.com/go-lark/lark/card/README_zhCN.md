# 卡片构造工具

交互卡片是飞书消息中比较受欢迎的交互形式，但构造起来比较复杂，因此我们提供了专门的卡片构造工具。

卡片构造工具采用声明式 API 设计，我们把每一个元素（`div`、`text`、`button` 等）都映射成声明的形式。

我们约定使用「参数」来表示**元素内部**的子元素（例如 `div` 中的 `fields`），用「链式调用」来设置**元素本身**的属性（例如卡片中的 `forward`）。

## 使用方式

```go
b := lark.NewCardBuilder()
c := b.Card(
    b.Markdown("some text"),
).
    Title("title").
    NoForward()

fmt.Println(c.String())
```

会渲染出：

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

参考实例：[交互卡片](https://github.com/go-lark/examples/tree/main/interactive-message)

## 国际化支持

我们也支持构造多语言的卡片，在标题和内容等特定部分，需要引用 `I18N` 结构中的方法构造：

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

## 发送

```go
b := lark.NewCardBuilder()
c := b.Card(...Elements)
msg := lark.NewMsgBuffer(lark.MsgInteractive)
om := msg.BindEmail("youremail@example.com").Card(c.String()).Build()
bot.PostMessage(om)
```
