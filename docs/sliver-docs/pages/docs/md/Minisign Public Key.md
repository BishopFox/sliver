This [Minisign](https://jedisct1.github.io/minisign/) key is used to sign binary artifacts from the Sliver Authors specifically. As anyone is welcome to modify and redistribute the project under the terms of the GPLv3 license an artifact that is not signed by this specific key is not necessarily illegitimate or malicious, it simply means it's not from the original project authors.


## Public Key

```
RWTZPg959v3b7tLG7VzKHRB1/QT+d3c71Uzetfa44qAoX5rH7mGoQTTR
```

[https://sliver.sh/minisign.pub](/minisign.pub)

## Verification

Minisign is available in most Linux package managers, there is also a [Go implementation](https://github.com/aead/minisign).

```
minisign -Vm sliver-server -P RWTZPg959v3b7tLG7VzKHRB1/QT+d3c71Uzetfa44qAoX5rH7mGoQTTR
```