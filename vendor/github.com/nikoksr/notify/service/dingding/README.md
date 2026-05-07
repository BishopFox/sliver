# dingtalk

## Prerequisites
To use the service you need to apply for robot of DingTalk Group.
### create robot of DingTalk group
1. add `custom` rebot。
2. look the safe setting, and select the `sign(加签)`
   ![Xnip2020-07-05_15-55-24.jpg](https://i.loli.net/2020/07/05/4XqHG2dOwo8StEu.jpg)

## Usage
```go
cfg := Config{
  token:  "dddd",
  secret: "xxx",
}

s := New(&cfg)
s.Send(context.Background(), "subject", "content")
```
