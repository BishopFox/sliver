Prelude
=======

The `prelude` package handles the connection with [Prelude's Operator platform](https://www.prelude.org/).

## BOF support

To execute BOF TTPs, use the custom `bof` executor:

```json
bof.[{"type": "int", "value": #{json.T1234.ProcessId}}]
```