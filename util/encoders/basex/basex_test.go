package basex

import (
	h "encoding/hex"
	"testing"
)

var testCases = []struct {
	alphabet string
	dec      string
	enc      string
}{
	{"01", "000f", "01111"},
	{"01", "00ff", "011111111"},
	{"01", "0fff", "111111111111"},
	{"01", "ff00ff00", "11111111000000001111111100000000"},
	{"01", "fb6f9ac3", "11111011011011111001101011000011"},
	{"01", "179eea7a", "10111100111101110101001111010"},
	{"01", "6db825db", "1101101101110000010010111011011"},
	{"01", "93976aa7", "10010011100101110110101010100111"},
	{"0123456789abcdef", "0000000f", "000f"},
	{"0123456789abcdef", "000fff", "0fff"},
	{"0123456789abcdef", "ffff", "ffff"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "", ""},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "61", "2g"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "626262", "a3gV"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "636363", "aPEr"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "73696d706c792061206c6f6e6720737472696e67", "2cFupjhnEsSn59qHXstmK2ffpLv2"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "00eb15231dfceb60925886b67d065299925915aeb172c06647", "1NS17iag9jJgTHD1VXjvLCEnZuQ3rJDE9L"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "516b6fcd0f", "ABnLTmg"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "bf4f89001e670274dd", "3SEo3LWLoPntC"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "572e4794", "3EFU7m"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ecac89cad93923c02321", "EJDM8drfXA6uyA"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "10c8511e", "Rt5zm"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "00000000000000000000", "1111111111"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "801184cd2cdd640ca42cfc3a091c51d549b2f016d454b2774019c2b2d2e08529fd206ec97e", "5Hx15HFGyep2CfPxsJKe2fXJsCVn5DEiyoeGGF6JZjGbTRnqfiD"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "003c176e659bea0f29a3e9bf7880c112b1b31b4dc826268187", "16UjcYNBG9GTK4uq2f7yYEbuifqCzoLMGS"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ffffffffffffffffffff", "FPBt6CHo3fovdL"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ffffffffffffffffffffffffff", "NKioeUVktgzXLJ1B3t"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ffffffffffffffffffffffffffffffff", "YcVfxkQb6JRzqk5kF2tNLv"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ef41b9ce7e830af7", "h26E62FyLQN"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "606cbc791036d2e9", "H8Sa62HVULG"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "bdcb0ea69c2c8ec8", "YkESUPpnfoD"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "1a2358ba67fb71d5", "5NaBN89ajtQ"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "e6173f0f4d5fb5d7", "fVAoezT1ZkS"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "91c81cbfdd58bbd2", "RPGNSU3bqTX"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "329e0bf0e388dbfe", "9U41ZkwwysT"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "30b10393210fa65b", "99NMW3WHjjY"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ab3bdd18e3623654", "VeBbqBb4rCT"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "fe29d1751ec4af8a", "jWhmYLN9dUm"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "c1273ab5488769807d", "3Tbh4kL3WKW6g"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6c7907904de934f852", "2P5jNYhfpTJxy"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "05f0be055db47a0dc9", "5PN768Kr5oEp"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "3511e6206829b35b12", "gBREojGaJ6DF"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "d1c7c2ddc4a459d503", "3fsekq5Esq2KC"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "1f88efd17ab073e9a1", "QHJbmW9ZY7jn"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "0f45dadf4e64c5d5c2", "CGyVUMmCKLRf"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "de1e5c5f718bb7fafa", "3pyy8U7w3KUa5"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "123190b93e9a49a46c", "ES3DeFrG1zbd"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "8bee94a543e7242e5a", "2nJnuWyLpGf6y"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "9fd5f2285362f5cfd834", "9yqFhqeewcW3pF"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6987bac63ad23828bb31", "6vskE5Y1LhS3U4"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "19d4a0f9d459cc2a08b0", "2TAsHPuaLhh5Aw"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "a1e47ffdbea5a807ab26", "A6XzPgSUJDf1W5"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "35c231e5b3a86a9b83db", "42B8reRwPAAoAa"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "b2351012a48b8347c351", "B1hPyomGx4Vhqa"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "71d402694dd9517ea653", "7Pv2SyAQx2Upu8"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "55227c0ec7955c2bd6e8", "5nR64BkskyjHMq"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "17b3d8ee7907c1be34df", "2LEg7TxosoxTGS"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "7e7bba7b68bb8e95827f", "879o2ATGnmYyAW"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "db9c13f5ba7654b01407fb", "wTYfxjDVbiks874"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6186449d20f5fd1e6c4393", "RBeiWhzZNL6VtMG"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "5248751cebf4ad1c1a83c3", "MQSVNnc8ehFCqtW"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "32090ef18cd479fc376a74", "DQdu351ExDaeYeX"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "7cfa5d6ed1e467d986c426", "XzW67T5qfEnFcaZ"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "9d8707723c7ede51103b6d", "g4eTCg6QJnB1UU4"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6f4d1e392d6a9b4ed8b223", "Ubo7kZY5aDpAJp2"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "38057d98797cd39f80a0c9", "EtjQ2feamJvuqse"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "de7e59903177e20880e915", "xB2N7yRBnDYEoT2"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "b2ea24a28bc4a60b5c4b8d", "mNFMpJ2P3TGYqhv"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "cf84938958589b6ffba6114d", "4v8ZbsGh2ePz5sipt"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "dee13be7b8d8a08c94a3c02a", "5CwmE9jQqwtHkTF45"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "14cb9c6b3f8cd2e02710f569", "Pm85JHVAAdeUdxtp"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ca3f2d558266bdcc44c79cb5", "4pMwomBAQHuUnoLUC"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "c031215be44cbad745f38982", "4dMeTrcxiVw9RWvj3"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "1435ab1dbc403111946270a5", "P7wX3sCWNrbqhBEC"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "d8c6e4d775e7a66a0d0f9f41", "56GLoRDGWGuGJJwPN"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "dcee35e74f0fd74176fce2f4", "5Ap1zyuYiJJFwWcMR"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "bfcc0ca4b4855d1cf8993fc0", "4cvafQW4PEhARKv9D"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "e02a3ac25ece7b54584b670a", "5EMM28xkpxZ1kkVUM"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "fe4d938fc3719f064cabb4bfff", "NBXKkbHwrAsiWTLAk6"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "9289cb4f6b15c57e6086b87ea5", "DCvDpjEXEbHjZqskKv"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "fc266f35626b3612bfe978537b", "N186PVoBWrNre35BGE"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "33ff08c06d92502bf258c07166", "5LC4SoW6jmTtbkbePw"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6a81cac1f3666bc59dc67b1c3c", "9sXgUySUzwiqDU5WHy"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "9dfb8e7e744c544c0f323ea729", "EACsmGmkgcwsrPFzLg"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "1e7a1e284f70838b38442b682b", "3YEVk9bE7rw5qExMkv"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "2a862ad57901a8235f5dc74eaf", "4YS259nuTLfeXa5Wuc"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "74c82096baef21f9d3089e5462", "AjAcKEhUfrqm8smvM7"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "7a3edbc23d7b600263920261cc", "BBZXyRgey5S5DDZkcK"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "20435664c357d25a9c8df751cf4f", "CrwNL6Fbv4pbRx1zd9g"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "51a7aa87cf5cb1c12d045ec3422d", "X27NHGgKXmGzzQvDtpC"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "344d2e116aa26f1062a2cb6ebbef", "LEDLDvL1Hg4qt1efVXt"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6941add7be4c0b5c7163e4928f8e", "fhMyN6gwoxE3uYraVzV"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "10938fcbb7c4ab991649734a14bf", "76TPrSDxzGQfSzMu974"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "eafe04d944ba504e9af9117b07de", "2VPgov563ryfe4L2Bj6M"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "58d0aeed4d35da20b6f052127edf", "ZenZhXF9YwP8nQvNtNz"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "d734984e2f5aecf25f7a3e353f8a", "2N7n3jFsTdyN49Faoq6h"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "57d873fdb405b7daf4bafa62068a", "ZJ7NwoP4wHvwyZg3Wjs"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "bda4ec7b40d0d65ca95dec4c4d3b", "2CijxjsNyvqTwPCfDcpA"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "826c4abdceb1b91f0d4ad665f86d2e", "4edfvuDQu9KzVxLuXHfMo"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "e7ecb35d07e65b960cb10574a4f51a", "7VLRYdB4cToipp2J2p3v9"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "4f2d72ead87b31d6869fba39eac6dc", "3DUjqJRcfdWhpsrLrGcQs"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "8b4f5788d60030950d5dfbf94c585d", "4u44JSRH5jP5X39YhPsmE"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "ee4c0a0025d1a74ace9fe349355cc5", "7fgACjABRQUGUEpN6VBBA"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "58ac05b9a0b4b66083ff1d489b8d84", "3UtJPyTwGXapcxHx8Rom5"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "1aa35c05e1132e8e049aafaef035d8", "kE2eSU7gM2619pT82iGP"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "771b0c28608484562a292e5d5d2b30", "4LGYeWhyfrjUByibUqdVR"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "78ff9a0e56f9e88dc1cd654b40d019", "4PLggs66qAdbmZgkaPihe"},
	{"123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", "6d691bdd736346aa5a0a95b373b2ab", "44Y6qTgSvRMkdqpQ5ufkN"},
}

func Test_AmbiguousAlphabet(t *testing.T) {
	_, err := NewEncoding("01gh1")
	expect(err.Error(), ErrAmbiguousAlphabet.Error(), t)
}

func Test_Encode(t *testing.T) {
	for _, cs := range testCases {
		enc, _ := NewEncoding(cs.alphabet)
		expect(enc.Encode(hex(cs.dec)), cs.enc, t)
	}
}

func Test_Decode(t *testing.T) {
	for _, cs := range testCases {
		enc, _ := NewEncoding(cs.alphabet)
		dec, err := enc.Decode(cs.enc)
		if err != nil {
			t.Fatal(err)
		}
		expect(h.EncodeToString(dec), cs.dec, t)
	}
}

func Test_NonDecodable(t *testing.T) {
	enc, _ := NewEncoding("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
	_, err := enc.Decode("invalid")
	expect(err.Error(), ErrNonBaseChar.Error(), t)
	_, err = enc.Decode("c2F0b3NoaQo=")
	expect(err.Error(), ErrNonBaseChar.Error(), t)
	_, err = enc.Decode(" 1111111111")
	expect(err.Error(), ErrNonBaseChar.Error(), t)
	_, err = enc.Decode("1111111111 ")
	expect(err.Error(), ErrNonBaseChar.Error(), t)
	_, err = enc.Decode(" \t\n\u000b\f\r skip \r\f\u000b\n\t a")
	expect(err.Error(), ErrNonBaseChar.Error(), t)
}

func hex(in string) []byte {
	dec, err := h.DecodeString(in)
	if err != nil {
		panic(err)
	}
	return dec
}

func expect(cur, expected string, t *testing.T) {
	if cur != expected {
		t.Fatalf("Expected {%s} got {%s}.", expected, cur)
	}
}
