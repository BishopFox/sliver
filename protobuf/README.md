Protobuf
=========

This directory contains the protobuf message definitions and is organized in two packages:

 * `client` - Generally refered to as `clientpb` in the code, these messages should _only_ be sent from the client to server or vice versa.
 * `sliver` - Referred to as `sliverpb` in the `/server/` code, or simply `pb` in the `/sliver/` code, these message may be sent from the client to the server or from the server to the implant and vice versa.
 