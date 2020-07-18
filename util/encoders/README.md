Encoders
=========

General purpose data encoders, these are the server-side versions. Due to the build process of implants it's tricky for `server/encoders` and `sliver/encoders` to share the same code, however it's imperative these two packages are interoperable, though some encode/decode operations may only be used on one side (e.g., the implant can decode English but not encode to it). This is because certain encoders (like the English encoder) may require external files that we don't want to bundle with the implant.

There are two interfaces, both can encode arbitrary binary data but the outputs differ:
 * `BinaryEncoder` - Encodes binary data and outputs other binary data formats (e.g., encoding data into a PNGs/images)
 * `ASCIIEncoder` - Encode/decode binary data and only outputs valid ASCII (e.g., `base64` or english text)

## Encoders

#### `Base64` 

Encodes data using `base64` encoding with a custom alphabet so that it's not interoperable with standard Base64 encoding.

#### `Hex` 

Encodes data to ASCII/hex

#### `English`

Encodes arbitrary binary data into English text, this requires a dictionary file when encoding (one is provided in `assets/`). This is designed to be an inefficient encoder but the amount of inefficiency can be adjusted based off of the length of the words in the dictionary used during the encoding process.
