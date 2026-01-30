# dingtalk

## 使用
### 创建钉钉群机器人
1. 选择添加`自定义`机器人。
2. 安全设置`加签设置`
   ![Xnip2020-07-05_15-55-24.jpg](https://i.loli.net/2020/07/05/4XqHG2dOwo8StEu.jpg)

### 使用说明
```go
cfg := Config{
  token:  "dddd",
  secret: "xxx",
}

s := New(&cfg)
s.Send(context.Background(), "subject", "content")
```
