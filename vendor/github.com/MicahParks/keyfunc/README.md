[![Go Report Card](https://goreportcard.com/badge/github.com/MicahParks/keyfunc)](https://goreportcard.com/report/github.com/MicahParks/keyfunc) [![Go Reference](https://pkg.go.dev/badge/github.com/MicahParks/keyfunc.svg)](https://pkg.go.dev/github.com/MicahParks/keyfunc)

# keyfunc

The purpose of this package is to provide a
[`jwt.Keyfunc`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4#Keyfunc) for the
[github.com/golang-jwt/jwt/v4](https://github.com/golang-jwt/jwt) package using a JSON Web Key Set (JWK Set or JWKS) for
parsing and verifying JSON Web Tokens (JWTs).

There is legacy support for `github.com/dgrijalva/jwt-go` and its popular forks. It's in a separate project to keep this
project minimal. If your use case supports a legacy fork, please
see: [github.com/MicahParks/compatibility-keyfunc](https://github.com/MicahParks/compatibility-keyfunc).

It's common for an identity provider, such as [Keycloak](https://www.keycloak.org/)
or [Amazon Cognito (AWS)](https://aws.amazon.com/cognito/) to expose a JWKS via an HTTPS endpoint. This package has the
ability to consume that JWKS and produce a
[`jwt.Keyfunc`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4#Keyfunc). It is important that a JWKS endpoint is using
HTTPS to ensure the keys are from the correct trusted source.

This repository only depends on: [github.com/golang-jwt/jwt/v4](https://github.com/golang-jwt/jwt)

`jwt.Keyfunc` signatures are imported from these, implemented, then exported as methods.

## Supported Algorithms

Currently, this package supports JWTs signed with a `kty` that matches one of the following:
* `EC` [Elliptic Curve Digital Signature Algorithm (ECDSA)](https://en.wikipedia.org/wiki/Elliptic_Curve_Digital_Signature_Algorithm)
* `RSA` [Rivest–Shamir–Adleman (RSA)](https://en.wikipedia.org/wiki/RSA_(cryptosystem))
* `OKP` [Edwards-curve Digital Signature Algorithm (EdDSA)](https://en.wikipedia.org/wiki/EdDSA)
* `OCT` [HMAC](https://en.wikipedia.org/wiki/HMAC), [AES Key Wrap](https://en.wikipedia.org/wiki/Key_Wrap), and others

Additionally, the supported `EC` elliptical curve types are below:
* `P-256`
* `P-384`
* `P-521`

For `OKP`, EdDSA, only the `Ed25519` curve type is supported. (I have not found a standard library package that
supports `Ed448`.)

This _does_ include HMAC keys. For an example using HMAC keys, please see the `examples/hmac` directory. Do _not_ expose
HMAC keys in public facing JWKS, as HMAC keys are secret keys that do not use public key cryptography.

If there are cryptographic algorithms, curve types, or something else already standardized that you'd like supported in
this Go package, please open an issue or pull request.

## Basic usage

For complete examples, please see the `examples` directory.

```go
import "github.com/MicahParks/keyfunc"
```

#### A note on read-only keys
The [`JWKS.ReadOnlyKeys`](https://pkg.go.dev/github.com/MicahParks/keyfunc#JWKS.ReadOnlyKeys) method returns a read-only
copy of a `map[string]interface{}`. The key to this map is the key ID, `kid`, and the value is the cryptographic key.
This is a useful map for use of keys within a JWKS outside of `github.com/golang-jwt/jwt/v4`.

The map itself is a copy. So it can be modified safely. However, the values are of type `interface{}`. If these values
are modified, it may cause undefined behavior.

### Preconditions: Acquire the JWKS URL, JSON, or gather cryptographic keys (given keys)

A JWKS URL is not required, one can be created directly from JSON with the
[`keyfunc.NewJSON`](https://pkg.go.dev/github.com/MicahParks/keyfunc#NewJSON) function.

```go
// Get the JWKS URL from an environment variable.
jwksURL := os.Getenv("JWKS_URL")

// Confirm the environment variable is not empty.
if jwksURL == "" {
	log.Fatalln("JWKS_URL environment variable must be populated.")
}
```

### Step 1: Create the JWKS

Via HTTP:
```go
// Create the JWKS from the resource at the given URL.
jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{}) // See recommended options in the examples directory.
if err != nil {
	log.Fatalf("Failed to get the JWKS from the given URL.\nError: %s", err)
}
```
Via JSON:
```go
// Get the JWKS as JSON.
var jwksJSON = json.RawMessage(`{"keys":[{"kid":"zXew0UJ1h6Q4CCcd_9wxMzvcp5cEBifH0KWrCz2Kyxc","kty":"RSA","alg":"PS256","use":"sig","n":"wqS81x6fItPUdh1OWCT8p3AuLYgFlpmg61WXp6sp1pVijoyF29GOSaD9xE-vLtegX-5h0BnP7va0bwsOAPdh6SdeVslEifNGHCtID0xNFqHNWcXSt4eLfQKAPFUq0TsEO-8P1QHRq6yeG8JAFaxakkaagLFuV8Vd_21PGJFWhvJodJLhX_-Ym9L8XUpIPps_mQriMUOWDe-5DWjHnDtfV7mgaOxbBvVo3wj8V2Lmo5Li4HabT4MEzeJ6e9IdFo2kj_44Yy9osX-PMPtu8BQz_onPgf0wjrVWt349Rj6OkS8RxlNGYeuIxYZr0TOhP5F-yEPhSXDsKdVTwPf7zAAaKQ","e":"AQAB","x5c":["MIICmzCCAYMCBgF4HR7HNDANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzEwMTcwOTE5WhcNMzEwMzEwMTcxMDU5WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDCpLzXHp8i09R2HU5YJPyncC4tiAWWmaDrVZenqynWlWKOjIXb0Y5JoP3ET68u16Bf7mHQGc/u9rRvCw4A92HpJ15WyUSJ80YcK0gPTE0Woc1ZxdK3h4t9AoA8VSrROwQ77w/VAdGrrJ4bwkAVrFqSRpqAsW5XxV3/bU8YkVaG8mh0kuFf/5ib0vxdSkg+mz+ZCuIxQ5YN77kNaMecO19XuaBo7FsG9WjfCPxXYuajkuLgdptPgwTN4np70h0WjaSP/jhjL2ixf48w+27wFDP+ic+B/TCOtVa3fj1GPo6RLxHGU0Zh64jFhmvRM6E/kX7IQ+FJcOwp1VPA9/vMABopAgMBAAEwDQYJKoZIhvcNAQELBQADggEBALILq1Z4oQNJZEUt24VZcvknsWtQtvPxl3JNcBQgDR5/IMgl5VndRZ9OT56KUqrR5xRsWiCvh5Lgv4fUEzAAo9ToiPLub1SKP063zWrvfgi3YZ19bty0iXFm7l2cpQ3ejFV7WpcdLJE0lapFdPLo6QaRdgNu/1p4vbYg7zSK1fQ0OY5b3ajhAx/bhWlrN685owRbO5/r4rUOa6oo9l4Qn7jUxKUx4rcoe7zUM7qrpOPqKvn0DBp3n1/+9pOZXCjIfZGvYwP5NhzBDCkRzaXcJHlOqWzMBzyovVrzVmUilBcj+EsTYJs0gVXKzduX5zO6YWhFs23lu7AijdkxTY65YM0="],"x5t":"IYIeevIT57t8ppUejM42Bqx6f3I","x5t#S256":"TuOrBy2NcTlFSWuZ8Kh8W8AjQagb4fnfP1SlKMO8-So"},{"kid":"ebJxnm9B3QDBljB5XJWEu72qx6BawDaMAhwz4aKPkQ0","kty":"EC","alg":"ES512","use":"sig","crv":"P-521","x":"YQ95Xj8MTzcHytbU1h8YkCN2kdEQA7ThuZ1ctB9Ekiw6tlM9RwL62eQvzEt4Rz8qN69uRqgU9RzxQOkSU5xVvyo","y":"SMMuP3QnAPHtx7Go2ARsG3NBaySWBLmVvS8s2Ss7Vm_ISWenNbdjKOsY1XvtiQz5scGzWDCEUoZzgV8Ve1mLOV0"},{"kid":"TVAAet63O3xy_KK6_bxVIu7Ra3_z1wlB543Fbwi5VaU","kty":"EC","alg":"ES384","use":"sig","crv":"P-384","x":"Pik2o5as-evijFABH5p6YLXHnWw8iQ_N1ummPY1c_UgG6NO0za-gNOhTz2-tsd_w","y":"e98VSff71k19SY_mHgp3707lgQVrhfVpiGa-sGaKxOWVpxd2jWMhB0Q4RpSRuCp5"},{"kid":"arlUxX4hh56rNO-XdIPhDT7bqBMqcBwNQuP_TnZJNGs","kty":"RSA","alg":"RS512","use":"sig","n":"hhtifu8LL3ICE3BAX5l1KZv6Lni0lhlhBusSfepnpxcb4C_z2U71cQTnLY27kt8WB4bNG6e5_KMx9K3xUdd3euj9MCq8vytwEPieeHE1KXQuhJfLv017lhpK_dRMOHyc-9-50YNdgs_8KWRkrzjjuYrCiO9Iu76n5319e-SC8OPvNUglqxp2N0Sp2ltne2ZrpN8T3OEEXT62TSGmLAVopRGw5gllNVrJfmEyZJCRrBM6s5CQcz8un0FjkAAC4DI6QD-eBL0qG3_NR0hQvR1he2o4BLwjOKH45Pk_jj-eArp-DD6Xq6ABQVb5SNOSdaxl5lnmuotRoY3G5d9YSl-K3w","e":"AQAB","x5c":["MIICmzCCAYMCBgF4HSCcDzANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzEwMTcxMTE5WhcNMzEwMzEwMTcxMjU5WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCGG2J+7wsvcgITcEBfmXUpm/oueLSWGWEG6xJ96menFxvgL/PZTvVxBOctjbuS3xYHhs0bp7n8ozH0rfFR13d66P0wKry/K3AQ+J54cTUpdC6El8u/TXuWGkr91Ew4fJz737nRg12Cz/wpZGSvOOO5isKI70i7vqfnfX175ILw4+81SCWrGnY3RKnaW2d7Zmuk3xPc4QRdPrZNIaYsBWilEbDmCWU1Wsl+YTJkkJGsEzqzkJBzPy6fQWOQAALgMjpAP54EvSobf81HSFC9HWF7ajgEvCM4ofjk+T+OP54Cun4MPperoAFBVvlI05J1rGXmWea6i1Ghjcbl31hKX4rfAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAB7bpwPoL02WGCCVhCsbDkq9GeFUwF01opVyFTijZlTUoTf5RcaR2qAH9/irkLjZeFeyozzC5mGvIVruBwnx/6l4PcAMxKK4YiheFVoO/dytpGMCj6ToNmKpjlXzOLAHelieWIUDtAFSYzENjIO01PyXTGYpxebpQCocJBvppj5HqARS9iNPcqBltMhxWrWmMu81tOG3Y7yd2xsIYXk6KjaoefLeN8Was4BPJ0zR6tTSEm6ZOvSRvlppqh84kz7LmWem7gGHAsY2G3tWBUmOdO/SMNMThqV62yLf7sKsuoE1w06lfmrf6D2zGwoEyz+TT6fdSkc34Yeh7+c01X6nFWU="],"x5t":"geiCPGtT_10T8xGLUK1LA0_YQEE","x5t#S256":"dLp3_QNGwMbYll5VecnR8Q9NSeFVfqJPBTa2_8qf48I"},{"kid":"tW6ae7TomE6_2jooM-sf9N_6lWg7HNtaQXrDsElBzM4","kty":"RSA","alg":"PS512","use":"sig","n":"p32N7jqKfMUB6_dKY1uZ3wizzPlBAXg9XrntfUcwNLRPfTBnshpt4uQBf3T8fexkbzhtR18oHvim-YvcWfC5eLGQmWHYiVwACa_C7oGqx51ijK2LRbUg4TKhnZX2X3Ld9xvr3HsosKh2UXn_Ay8nuvdfH-U6S7btT6a-AIFlt3BpqZP0EOl7rY-ie8nXoA13xX6BoyzYiNcugdYCU6czQcmTIJ1JLS0zohi4aTNehRt-1VMRpIMx7q7Ouq3Zhbi7RcDo-_D8FPRhWc2eEKd-h8ebFTIxEOrkguBIomjEFTf3SfYbOB_h-14v9Q2yz-NzyId3-ujRCQGC0hn-cixe2w","e":"AQAB","x5c":["MIICmzCCAYMCBgF4BKAxqzANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzA1MjMwMDEwWhcNMzEwMzA1MjMwMTUwWjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCnfY3uOop8xQHr90pjW5nfCLPM+UEBeD1eue19RzA0tE99MGeyGm3i5AF/dPx97GRvOG1HXyge+Kb5i9xZ8Ll4sZCZYdiJXAAJr8LugarHnWKMrYtFtSDhMqGdlfZfct33G+vceyiwqHZRef8DLye6918f5TpLtu1Ppr4AgWW3cGmpk/QQ6Xutj6J7ydegDXfFfoGjLNiI1y6B1gJTpzNByZMgnUktLTOiGLhpM16FG37VUxGkgzHurs66rdmFuLtFwOj78PwU9GFZzZ4Qp36Hx5sVMjEQ6uSC4EiiaMQVN/dJ9hs4H+H7Xi/1DbLP43PIh3f66NEJAYLSGf5yLF7bAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAHVWNBTExqlg4LTcyhUXI5U0iNPcMIVdKDoGPDc3EPjXyYNyjURX0oZ6b1Wv5t+XGmpZRqJNYb92xraQatIzLEsRn4IrmzViP+dIyFU8BEDubixTxeqx7LSw2j6LIFnZ05XdmWknlksNTlqi4CT6KL+1c24+QU3CcmU3mkQEIPA2yC4SdAB1oXI0jh49uP6a+JrE7JREZGAdwbIpZ1cqV6acPiJW3tOYfLrHwo7KYn3KwJvIBHXgFBNwx7fl2gYNQ0VEGKub3qVwW5RO5R/6Tcla9uZEfEiamms/Pn4hFA1qbsNHtA9IRGVRSmVeBKDxRvo0fxOUXp+NuZxEnhsoP3I="],"x5t":"f1l1fxICz1fe9mI-sSrtc19EDhU","x5t#S256":"NUJWRA4ADpLEg_SMkSoE4FKQN0H1Tlz85L-i7puVcqQ"},{"kid":"Lx1FmayP2YBtxaqS1SKJRJGiXRKnw2ov5WmYIMG-BLE","kty":"RSA","alg":"PS384","use":"sig","n":"q7WM4SnrdzlFSo_A1DRhc-8Ho-pBsfs49kGRbw3O_OKFIUyZrzHaRuovW_QaEAyiO3HX8CNcGPcpHdmpl4DhTGEBLcd6xXtCaa65ct00Mq7ZHCRRCrKLh6lJ0rY9fP8vCV0RBigpkNoRfrqLQQN4VeVFTbGSrDaS0LzPbap0-q5FKXUR-OQmQEtOupXhKFQtbB73tL83YnG6Swl7nXsx54ulEoDzcCCYt7pjCVVp7L9fzI2_ucTdtQclAJVQZGKpsx7vabOJuiMUwuAIz56lOJyXRMePsW8UogwC4FA2A52STsYlhOPsDEW4iIExFVNqs-CGoDGhYLIavaCkZhXM0w","e":"AQAB","x5c":["MIICmzCCAYMCBgF4HR+9XjANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzEwMTcxMDIyWhcNMzEwMzEwMTcxMjAyWjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrtYzhKet3OUVKj8DUNGFz7wej6kGx+zj2QZFvDc784oUhTJmvMdpG6i9b9BoQDKI7cdfwI1wY9ykd2amXgOFMYQEtx3rFe0Jprrly3TQyrtkcJFEKsouHqUnStj18/y8JXREGKCmQ2hF+uotBA3hV5UVNsZKsNpLQvM9tqnT6rkUpdRH45CZAS066leEoVC1sHve0vzdicbpLCXudezHni6USgPNwIJi3umMJVWnsv1/Mjb+5xN21ByUAlVBkYqmzHu9ps4m6IxTC4AjPnqU4nJdEx4+xbxSiDALgUDYDnZJOxiWE4+wMRbiIgTEVU2qz4IagMaFgshq9oKRmFczTAgMBAAEwDQYJKoZIhvcNAQELBQADggEBADTgP3SrcG3p9XUB7sM4a2IeY0J4bSEtqlZBuHgdgekYJ5DXETJ3hV/82GjitU50NBup0IJyI9KZ0KCwqHIKC2Jn/6biOpM9Ipk4BtNVzx3qKNsDac9qZmyMpm4V9QuWakajknerhwyynG3siGUntbPmLvf5UKvKtbiKlWS4dBPwfedIUnC85mYEnNKSzSI1NiM6TWHB9zQYkARXlb89sh0HBYs08BfRMyBVM+l3OczIyGeQAfhcL+pxPP/0jqPr1ctHUBj2zXkjZxDw1oJFgeD9GDtPcjc3spB20vsRtQUBlzbJElbGflqWGHJK5l5n7gNd3ZXZT0HJ+wUpPE8EUaM="],"x5t":"fjRYR1986VCLzbaZaw5r25UKahw","x5t#S256":"ZHNHpizlsjD3qSZh7gJQQBu8W9jBL2HR0y7-3u2Wb-g"},{"kid":"gnmAfvmlsi3kKH3VlM1AJ85P2hekQ8ON_XvJqs3xPD8","kty":"RSA","alg":"RS384","use":"sig","n":"qUNQewKl3APQcbpACMNJ2XphPpupt395z6OZvj5CW9tiRXY3J7dqi8U0bWoIhtmmc7Js6hjp-A5W_FVStuXlT1hLyjJsHeu9ZVPnfIl2MnYN83zQBKw8E4mFsVv0UXNvkVPBF_k0yXrz-ABleWLOgFGnkNU9csc3Z5aihHcwRmC_oS7PZ9Vc-l0xBCyF3YRHI-al8ppSHwFreOweF3-JP3poNAXd906_tjX2KlHSJmNqcUNiSfEluyCp02ALlRFKXUQ1HlfSupHcHySDlanfUyIzZgM9ysCvC1vfNdAuwZ44oUBMul_XPxxhzlewL2Y8PtSDLUDWGTIou8M8049D8Q","e":"AQAB","x5c":["MIICmzCCAYMCBgF4BJVfaDANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzA1MjI0ODIxWhcNMzEwMzA1MjI1MDAxWjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCpQ1B7AqXcA9BxukAIw0nZemE+m6m3f3nPo5m+PkJb22JFdjcnt2qLxTRtagiG2aZzsmzqGOn4Dlb8VVK25eVPWEvKMmwd671lU+d8iXYydg3zfNAErDwTiYWxW/RRc2+RU8EX+TTJevP4AGV5Ys6AUaeQ1T1yxzdnlqKEdzBGYL+hLs9n1Vz6XTEELIXdhEcj5qXymlIfAWt47B4Xf4k/emg0Bd33Tr+2NfYqUdImY2pxQ2JJ8SW7IKnTYAuVEUpdRDUeV9K6kdwfJIOVqd9TIjNmAz3KwK8LW9810C7BnjihQEy6X9c/HGHOV7AvZjw+1IMtQNYZMii7wzzTj0PxAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABoThxhMd7Xiq4x0GJeoJFv2yDKXCL3dJEAEWtOr2+PqdeJl/ZfOxBXynIvrdtYnQdICztN5ydEgDsZ02piDsxZ+s/0SA0iqjw/MEoBYobmr8V+xwUv+WtRLpTBXqWGMuG7NEtrbjKid0iKLLAOAU4dcHQ49iOF9VLnbTkf1EXp4iphJreaubOXMwT6/JDzQPT1dRR34hlhYeKKzMSA0Cz5aYL1tI+eH12rar0MDczXykLChNS/8MlyTzreEf0siUiS9S1kj/lOZKQDg9E/z8fm5vmHEHzAVwf4ON5iO29tDsqLw7BeJqC4AESjliXIqMrdpFynfPnIsGgf3dnph5BM="],"x5t":"CmRnQVduZWtEsdOC4mauUUsSWxA","x5t#S256":"BvC0LmuM8ZIApN3TQQZWWbGO-d082Ah5d3D6vPvahGw"},{"kid":"CGt0ZWS4Lc5faiKSdi0tU0fjCAdvGROQRGU9iR7tV0A","kty":"EC","alg":"ES256","use":"sig","crv":"P-256","x":"DPW7n9yjfE6Rt-VvVmEdeu4QdW44qifocAPPDxACDDY","y":"-ejsVw8222-hg2dJWx3QV0hE4-I0Ujp7ZsWebE68JE0"},{"kid":"C65q0EKQyhpd1m4fr7SKO2He_nAxgCtAdws64d2BLt8","kty":"RSA","alg":"RS256","use":"sig","n":"ja99ybDrLvw11Z4CvNlDI-kkqJEBpSnvDf0pZF2DvBlvYmeVYL_ChqIe8E9GyHUmLMdtO_jifSgOqE5b8vILwi1kZnJR7N857uEnbWM9YTeevi_RZ-E_hr4frW2NKJ78YGvCzwLKG2GgtSjj0zuTLnSaK8fCGzqXgy6paXNhgHUSZgGwvO0YItpMlyJeqEj1wGTWz1IyA1sguF1cC7K0fojPbPoBwrhvaAeoGRPLraE0rrBsQv8iiLwnRBIez9B1j0NiUG8Iad953Y7UzaKOAw8crIEK45NIK_yxHUpxqcHLjPIcRyIyJGioRyGK7cp-_7iPLOCutQc-u46mom1_ZQ","e":"AQAB","x5c":["MIICmzCCAYMCBgF4BJRpbzANBgkqhkiG9w0BAQsFADARMQ8wDQYDVQQDDAZtYXN0ZXIwHhcNMjEwMzA1MjI0NzE4WhcNMzEwMzA1MjI0ODU4WjARMQ8wDQYDVQQDDAZtYXN0ZXIwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCNr33JsOsu/DXVngK82UMj6SSokQGlKe8N/SlkXYO8GW9iZ5Vgv8KGoh7wT0bIdSYsx207+OJ9KA6oTlvy8gvCLWRmclHs3znu4SdtYz1hN56+L9Fn4T+Gvh+tbY0onvxga8LPAsobYaC1KOPTO5MudJorx8IbOpeDLqlpc2GAdRJmAbC87Rgi2kyXIl6oSPXAZNbPUjIDWyC4XVwLsrR+iM9s+gHCuG9oB6gZE8utoTSusGxC/yKIvCdEEh7P0HWPQ2JQbwhp33ndjtTNoo4DDxysgQrjk0gr/LEdSnGpwcuM8hxHIjIkaKhHIYrtyn7/uI8s4K61Bz67jqaibX9lAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAHrGJFhVNiQupIwkn2jiW/jBobm9CHUxOwQL5E7WdRz5uaOJ0v62PrynOQE9xim9Qk8bT3q7DThZs66U9bpIk3msKVRgXRfn5FZy1H5RKOlEEFZhGakPqSlC1yPbhUNhHXMs3GTzdGMLtYaGvSy6XM/8/zqVqVwgh6BpbAR9RfiSdyaiNTSBriu+n/tHW934G9J8UIzdfpVcb0Yt9y4o0UgIXt64NtGFq7zmNJijH88AxBZFB6eUUmQQCczebzoAjyYbVOes5gGFzboVWcyLe3iyD0vvsAVHJViXeiGoxhpKnc8ryISpRUBzsKngf5uZo3bnrD9PHLYBoGOHgzII1xw="],"x5t":"5GNr3LeRXHWI4YR8-QTSsF98oTI","x5t#S256":"Dgd0_wZZqvRuf4GEISPNHREX-1ixTMIsrPeGzk0bCxs"}]}`)

// Create the JWKS from the resource at the given URL.
jwks, err := keyfunc.NewJSON(jwksJSON)
if err != nil {
	log.Fatalf("Failed to create JWKS from JSON.\nError: %s", err)
}
```
Via a given key:
```go
// Get an HMAC key.
key := []byte("example secret")
uniqueKeyID := "myKeyID"

// Create the JWKS from the HMAC key.
jwks := keyfunc.NewGiven(map[string]keyfunc.GivenKey{
	uniqueKeyID: keyfunc.NewGivenHMAC(key),
})
```

Additional options can be passed to the [`keyfunc.Get`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4/keyfunc#Get)
function. See [`keyfunc.Options`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4/keyfunc#Options) and the additional
features mentioned at the bottom of this `README.md`.

### Step 2: Use the [`JWKS.Keyfunc`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4/keyfunc#JWKS.Keyfunc) method as the [`jwt.Keyfunc`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4#Keyfunc) when parsing tokens

```go
// Parse the JWT.
token, err := jwt.Parse(jwtB64, jwks.Keyfunc)
if err != nil {
	return nil, fmt.Errorf("failed to parse token: %w", err)
}
```

The [`JWKS.Keyfunc`](https://pkg.go.dev/github.com/MicahParks/keyfunc#JWKS.Keyfunc) method will automatically select the
key with the matching `kid` (if present) and return its public key as the correct Go type to its caller.

## Test coverage

Test coverage is currently `>85%`.

Testing could be improved by signing all JWTs during the tests themselves. Alternatively, using JWTs that do not expire
would accomplish the same purpose. There are some hard-coded JWTs which are expired. This means the tests with hard
coded JWTs cannot check for parsing and validation errors, just errors within the `jwt.Keyfunc` itself.

## Additional features
These features can be configured by populating fields in the
[`keyfunc.Options`](https://pkg.go.dev/github.com/MicahParks/keyfunc#Options) argument to the
[`keyfunc.Get`](https://pkg.go.dev/github.com/MicahParks/keyfunc#Get) function.
* A background refresh of the JWKS keys can be performed.
	* A custom background refresh interval can be specified. For an example, please see the `examples/interval`
	  directory.
	* A custom background refresh request context timeout can be specified. Defaults to one minute. For an example,
	  please see the `examples/ctx` directory.
	* A custom background refresh error handling function can be specified. If none is specified, errors go unhandled
	  silently. For an example, please see the `examples/recommended_options` directory.
	* A custom rate limit can be specified to prevent too many requests for a JWKS refresh. For an example, please see
	  the `examples/recommended_options` directory.
	* JWTs with a previously unseen `kid` can prompt an automatic refresh of the remote JWKS resource. This should be
	  paired with `RefreshRateLimit` to prevent abuse. For an example, please see the `examples/recommended_options`
	  directory.
* A custom HTTP client can be used.
* A custom HTTP request factory can be provided to create HTTP requests for the remote JWKS resource. For example, an
  HTTP header can be added to indicate a User-Agent.
* A custom HTTP response extractor can be provided to get the raw JWKS JSON from the `*http.Response`. For example, the
  HTTP response code could be checked. Implementations are responsible for closing the response body.
	* By default,
	  the [`keyfunc.ResponseExtractorStatusOK`](https://pkg.go.dev/github.com/MicahParks/keyfunc#ResponseExtractorStatusOK)
	  function is used. The default behavior changed in `v1.4.0`.
* A custom whitelist of acceptable JSON Web Key `"use"` parameter values can be specified. Values not whitelisted will
  cause an error from the [`.Keyfunc`](https://pkg.go.dev/github.com/MicahParks/keyfunc#JWKS.Keyfunc) method. This
  whitelist can be disabled with the `JWKUseNoWhitelist` option.
	* By default, only JSON Web Keys with a `"use"` parameter value of `"sig"`, an empty string `""`, or a completely
	  omitted `"use"` parameter will be returned. The default behavior changed in `v1.5.0`.
	* This `"use"` whitelisting behavior is only available with `keyfunc.Get`. It is not available with
	  `keyfunc.NewJSON` or `keyfunc.NewGiven`. Please open a GitHub issue if you would like this feature added to the
	  other creation methods.
* A map of JWT key IDs (`kid`) to keys can be given and used for the `jwt.Keyfunc`. For an example, see
  the `examples/given` directory.
* A copy of the latest raw JWKS `[]byte` can be returned.
* Custom cryptographic algorithms can be used. Make sure to
  use [`jwt.RegisterSigningMethod`](https://pkg.go.dev/github.com/golang-jwt/jwt/v4#RegisterSigningMethod) before
  parsing JWTs. For an example, see the `examples/custom` directory.
* The remote JWKS resource can be refreshed manually using the `.Refresh` method. This can bypass the rate limit, if the
  option is set.
* There is support for creating one `jwt.Keyfunc` from multiple JWK Sets through the use of the `keyfunc.GetMultiple`.

## Notes
Trailing padding is required to be removed from base64url encoded keys inside a JWKS. This is because RFC 7517 defines
base64url the same as RFC 7515 Section 2:
* https://datatracker.ietf.org/doc/html/rfc7517#section-1.1
* https://datatracker.ietf.org/doc/html/rfc7515#section-2

However, this package will remove trailing padding on base64url encoded keys to account for improper implementations of
JWKS.

This package will check the `alg` in each JWK. If present, it will confirm the same `alg` is in a given JWT's header
before returning the key for signature verification. If the `alg`s do not match, `keyfunc.ErrJWKAlgMismatch` will
prevent the key being used for signature verification. If the `alg` is not present in the JWK, this check will not
occur.

## References
This project was built and tested using various RFCs and services. The services are listed below:
* [Keycloak](https://www.keycloak.org/)
* [Sample JWKS Service](https://jwks-service.appspot.com/)
* connect2id's [Server JWKSet Gen](https://connect2id.com/products/server/docs/config/jwk-set)
  ([Source](https://bitbucket.org/connect2id/server-jwkset-gen/src/master/))
