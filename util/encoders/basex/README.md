# basex

Package basex provides fast base encoding / decoding of any given alphabet using
bitcoin style leading zero compression. It is a GO port of
https://github.com/cryptocoinjs/base-x

## Usage

#### type Encoding

```go
type Encoding struct {
}
```

Encoding is a custom base encoding defined by an alphabet. It should bre created
using NewEncoding function

#### func  NewEncoding

```go
func NewEncoding(alphabet string) (*Encoding, error)
```
NewEncoding returns a custom base encoder defined by the alphabet string. The
alphabet should contain non-repeating characters. Ordering is important. Example
alphabets:

    - base2: 01
    - base16: 0123456789abcdef
    - base32: 0123456789ABCDEFGHJKMNPQRSTVWXYZ
    - base62: 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ

#### func (*Encoding) Decode

```go
func (e *Encoding) Decode(source string) ([]byte, error)
```
Decode function decodes a string previously obtained from Encode, using the same
alphabet and returns a byte slice In case the input is not valid an arror will
be returned

#### func (*Encoding) Encode

```go
func (e *Encoding) Encode(source []byte) string
```
Encode function receives a byte slice and encodes it to a string using the
alphabet provided
