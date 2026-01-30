# 微信小程序

[官方文档](https://developers.weixin.qq.com/miniprogram/dev/framework/)

## 包说明

- analysis 数据分析相关 API

## 快速入门

```go
wc := wechat.NewWechat()
memory := cache.NewMemory()
cfg := &miniConfig.Config{
    AppID:     "xxx",
    AppSecret: "xxx",
    Cache: memory,
}
miniprogram := wc.GetMiniProgram(cfg)
miniprogram.GetAnalysis().GetAnalysisDailyRetain()
```

### 小程序虚拟支付 
#### `注意：需要传入 Appkey、OfferID 的值`
相关文档：[小程序虚拟支付](https://developers.weixin.qq.com/miniprogram/dev/platform-capabilities/industry/virtual-payment.html)
```go
wc := wechat.NewWechat()
miniprogram := wc.GetMiniProgram(&miniConfig.Config{
    AppID:     "xxx",
    AppSecret: "xxx",
    AppKey:    "xxx",
    OfferID:   "xxx",
    Cache: cache.NewRedis(&redis.Options{
        Addr: "",
    }),
})
virtualPayment := miniprogram.GetVirtualPayment()
virtualPayment.SetSessionKey("xxx")
// 查询用户余额
var (
    res *virtualPayment.QueryUserBalanceResponse
    err error
)

if res, err = virtualPayment.QueryUserBalance(context.TODO(), &virtualPayment.QueryUserBalanceRequest{
    OpenID: "xxx",
    Env: virtualPayment.EnvProduction,
    UserIP: "xxx",
}); err != nil {
    panic(err)
}

```