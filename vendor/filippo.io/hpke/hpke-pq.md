# HPKE Hybrid KEMs

[filippo.io/hpke-pq](https://filippo.io/hpke-pq)

This document is a simplified and self-contained implementation reference for
the MLKEM768-X25519, MLKEM768-P256, and MLKEM1024-P384 hybrid HPKE KEMs,
specified in [draft-ietf-hpke-pq-03][], [draft-irtf-cfrg-hybrid-kems-07][],
[draft-irtf-cfrg-concrete-hybrid-kems-02][], and [draft-ietf-hpke-hpke-02].

It compensates for the need to cross-reference four documents, with different
nomenclature (including functions and components with the same name but
different behavior), alternative irrelevant definitions (the UG, UK, and CK
frameworks), and multiple KEM abstraction layers.

## Conventions used in this document

`||` denotes concatenation. `[N:M]` denotes the byte slice from index N (inclusive)
to index M (exclusive). Strings quoted with `""` are encoded as ASCII. Values in
code blocks are hex encoded byte strings. `random(N)` denotes N bytes of CSPRNG
output. All lengths are in bytes.

ML-KEM.KeyGen_internal, ML-KEM.Encaps, and ML-KEM.Decaps are defined in
[FIPS 203][]. `SHAKE256(s, L)` is an invocation of `SHAKE256(s, 8*L)` defined in
[FIPS 202][]. SHA3-256 is defined in [FIPS 202][].

## KEM definitions

| Parameter         | MLKEM768-X25519    | MLKEM768-P256      | MLKEM1024-P384     |
| ----------------- | ------------------ | ------------------ | ------------------ |
| ML-KEM parameters | ML-KEM-768         | ML-KEM-768         | ML-KEM-1024        |
| Group             | Curve25519         | P-256              | P-384              |
| KEM identifier    | 0x647a             | 0x0050             | 0x0051             |
| Nsecret           | 32                 | 32                 | 32                 |
| Nenc              | 1120               | 1153               | 1665               |
| Npk               | 1216               | 1249               | 1665               |
| Nsk               | 32                 | 32                 | 32                 |
| Label             | `"\.//^\"`         | `"MLKEM768-P256"`  | `"MLKEM1024-P384"` |
| KEM.Nct           | 1088               | 1088               | 1568               |
| KEM.Nek           | 1184               | 1184               | 1568               |    
| KEM.Nseed         | 64                 | 64                 | 64                 |
| Group.Nelem       | 32                 | 65                 | 97                 |
| Group.Nseed       | 32                 | 128                | 48                 |
| Group.Nscalar     | N/A                | 32                 | 48                 |

The MLKEM768-X25519 Label is alternatively encoded as

    5c2e2f2f5e5c

## KEM functions

```
def GenerateKeyPair():
    seed = random(32)

    ek_PQ, ek_T, _, _ = expandKey(seed)
    ek = ek_PQ || ek_T

    return (seed, ek)

def DeriveKeyPair(ikm):
    # SHAKE256.LabeledDerive is part of the single-stage KDF described in
    # draft-ietf-hpke-hpke-02 and defined in draft-ietf-hpke-pq-03, but is
    # reproduced below for convenience.
    seed = SHAKE256.LabeledDerive(ikm, "DeriveKeyPair", "", 32)

    ek_PQ, ek_T, _, _ = expandKey(seed)
    ek = ek_PQ || ek_T

    return (seed, ek)

def Encaps(ek):
    ek_PQ = ek[0 : KEM.Nek]
    ek_T = ek[KEM.Nek : KEM.Nek + Group.Nelem]

    ss_PQ, ct_PQ = ML-KEM.Encaps(ek_PQ)

    sk_E = Group.RandomScalar(random(Group.Nseed))
    ct_T = Group.Exp(Group.G, sk_E)
    ss_T = Group.ElementToSharedSecret(Group.Exp(ek_T, sk_E))

    ss = SHA3-256(ss_PQ || ss_T || ct_T || ek_T || Label)
    ct = ct_PQ || ct_T

    return (ss, ct)

def Decaps(seed, ct):
    ct_PQ = ct[0 : KEM.Nct]
    ct_T = ct[KEM.Nct : KEM.Nct + Group.Nelem]

    ek_PQ, ek_T, dk_PQ, dk_T = expandKey(seed)

    ss_PQ = ML-KEM.Decaps(dk_PQ, ct_PQ)
    ss_T = Group.ElementToSharedSecret(Group.Exp(ct_T, dk_T))

    ss = SHA3-256(ss_PQ || ss_T || ct_T || ek_T || Label)

    return ss

def expandKey(seed):
    seed_full = SHAKE256(seed, KEM.Nseed + Group.Nseed)
    seed_PQ = seed_full[0 : KEM.Nseed]
    seed_T = seed_full[KEM.Nseed : KEM.Nseed + Group.Nseed]

    # Note that even if expandKey returns the semi-expanded ML-KEM decapsulation
    # key dk_PQ to use FIPS 203 definitions, that format should be avoided and
    # instead seed_PQ should be expanded directly into the implementation's
    # internal ML-KEM private representation.
    (ek_PQ, dk_PQ) = ML-KEM.KeyGen_internal(seed_PQ)
    dk_T = Group.RandomScalar(seed_T)
    ek_T = Group.Exp(Group.G, dk_T)

    return (ek_PQ, ek_T, dk_PQ, dk_T)
```

There is no distinction between a private/public key and its serialization:
there is no abstract key format, only byte strings. In practice, implementations
will probably want to load keys into pairs of internal representations, and
serialize them back to their byte string format when needed.

The IETF/IRTF documents lack a specified way to turn a private key into public
key, although it can be inferred from the key generation process. We define such
a process here as `PrivateKeyToPublicKey`.

```
def PrivateKeyToPublicKey(seed):
    ek_PQ, ek_T, _, _ = expandKey(seed)
    ek = ek_PQ || ek_T

    return ek
```

## Group definitions

### Curve25519

Group.Exp is the X25519 function defined in [RFC 7748][].

Group.G is the canonical generator, which encodes to

    0900000000000000000000000000000000000000000000000000000000000000

consistently with [RFC 7748, Section 4.1][] and [RFC 7748, Section 6.1][].

Group.RandomScalar and Group.ElementToSharedSecret are the identity.

### P-256 and P-384

The NIST P-256 and P-384 elliptic curves are defined in [SP800-186][].

`Group.Exp(p, x)` computes scalar multiplication between the input element p and
the scalar x. The input element p and the output element have length Group.Nelem
and are encoded in uncompressed representation using the
Elliptic-Curve-Point-to-Octet-String and Octet-String-to-Elliptic-Curve-Point
functions defined in [SEC 1, Version 2.0][]. The input scalar x has length
Group.Nscalar and is encoded in big-endian representation using the I2OSP and
OS2IP functions defined in [RFC 8017][].

Group.G is the canonical generator, which encodes to

    046b17d1f2e12c4247f8bce6e563a440f277037d812deb33a0f4a13945d898c29
    64fe342e2fe1a7f9b8ee7eb4a7c0f9e162bce33576b315ececbb6406837bf51f5

for P-256, and to

    04aa87ca22be8b05378eb1c71ef320ad746e1d3b628ba79b9859f741e082542a385502f25dbf55296c3a545e3872760ab
    73617de4a96262c6f5d9e98bf9292dc29f8f41dbd289a147ce9da3113b5f0b8c00a60b1ce1d7e819d7a431d7c90ea0e5f

for P-384, consistently with [SP800-186][], Section 3.2.1.

```
def RandomScalar(seed):
    start = 0
    end = Nscalar
    sk = seed[start : end]
    while OS2IP(sk) == 0 || OS2IP(sk) >= OS2IP(Group.N):
        start = end
        end = end + Nscalar
        if end > len(seed):
            # This happens with cryptographically negligible probability.
            # The chance of a single rejection is < 2^-32 for P-256 and
            # < 2^-192 for P-384. The chance of reaching this is thus
            # < 2^-128 for P-256 and < 2^-192 for P-384.
            raise Exception("Rejection sampling failed")
        sk = seed[start : end]
    return sk
```

Group.N is the order of the curve's group, which encodes to

    ffffffff00000000ffffffffffffffffbce6faada7179e84f3b9cac2fc632551

for P-256, and to

    ffffffffffffffffffffffffffffffffffffffffffffffffc7634d81f4372ddf581a0db248b0a77aecec196accc52973

for P-384, consistently with [SP800-186][], Section 3.2.1.

Group.ElementToSharedSecret encodes the input element as an X coordinate using
the Field-Element-to-Octet-String function in [SEC 1, Version 2.0][].

> Note that since the scalar x is always derived uniformly at random, the chance
> of it being zero are cryptographically negligible. Moreover,
> Octet-String-to-Elliptic-Curve-Point never decodes the point at infinity from
> a string of Group.Nelem bytes. Since NIST P curves have prime order, this
> means that the output of Group.Exp and input to Group.ElementToSharedSecret is
> also never the point at infinity.

## SHAKE256.LabeledDerive

SHAKE256.LabeledDerive is used by DeriveKeyPair, and is part of the single-stage
KDF specified across [draft-ietf-hpke-hpke-02][] and [draft-ietf-hpke-pq-03][],
but is reproduced below for convenience.

```
def SHAKE256.LabeledDerive(ikm, label, context, L):
    suite_id = concat("KEM", I2OSP(kem_id, 2))
    prefixed_label = I2OSP(len(label), 2) || label
    labeled_ikm = ikm || "HPKE-v1" || suite_id || prefixed_label || I2OSP(L, 2) || context
    return SHAKE256(labeled_ikm, L)
```

I2OSP is defined in [RFC 8017][], and `kem_id` is the KEM identifier.

[draft-ietf-hpke-hpke-02]: https://datatracker.ietf.org/doc/html/draft-ietf-hpke-hpke-02
[draft-ietf-hpke-pq-03]: https://datatracker.ietf.org/doc/html/draft-ietf-hpke-pq-03
[draft-irtf-cfrg-hybrid-kems-07]: https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-hybrid-kems-07
[draft-irtf-cfrg-concrete-hybrid-kems-02]: https://datatracker.ietf.org/doc/html/draft-irtf-cfrg-concrete-hybrid-kems-02
[RFC 7748]: https://rfc-editor.org/rfc/rfc7748.html
[RFC 7748, Section 4.1]: https://rfc-editor.org/rfc/rfc7748.html#section-4.1
[RFC 7748, Section 6.1]: https://rfc-editor.org/rfc/rfc7748.html#section-6.1
[RFC 8017]: https://datatracker.ietf.org/doc/html/rfc8017
[FIPS 202]: https://doi.org/10.6028/NIST.FIPS.202
[FIPS 203]: https://doi.org/10.6028/NIST.FIPS.203
[SP800-186]: https://doi.org/10.6028/NIST.SP.800-186
[SEC 1, Version 2.0]: https://www.secg.org/sec1-v2.pdf
