// Copyright 2020 yiya1989. All rights reserved.
// Apache License Version 2.0
package krb5forssh

import (
	"encoding/binary"
	"fmt"
	"strings"

	"gopkg.in/jcmturner/gokrb5.v7/client"
	"gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/crypto"
	"gopkg.in/jcmturner/gokrb5.v7/gssapi"
	"gopkg.in/jcmturner/gokrb5.v7/iana/chksumtype"
	"gopkg.in/jcmturner/gokrb5.v7/iana/flags"
	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/messages"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"
	"gopkg.in/jcmturner/gokrb5.v7/types"
)

type Krb5ClientState int

const (
	ContextFlagREADY = 128
	/* initiator states */
	InitiatorStart Krb5ClientState = iota
	InitiatorRestart
	InitiatorWaitForMutal
	InitiatorReady
)

func NewKrb5InitiatorClient(krb5Conf, user, realm string, keytabConf []byte) (kcl Krb5InitiatorClient, err error) {
	c, _ := config.NewConfigFromString(krb5Conf)

	// Set to lookup KDCs in DNS
	c.LibDefaults.DNSLookupKDC = true

	// Blank out the KDCs to ensure they are not being used
	c.Realms = []config.Realm{}

	// Init keytab from conf
	kt := &keytab.Keytab{}
	if err = kt.Unmarshal([]byte(keytabConf)); err != nil {
		return kcl, fmt.Errorf("unmarshal keytabConf failed: %w", err)
	}

	// Init krb5 client and login
	cl := client.NewClientWithKeytab(user, realm, kt, c)

	err = cl.Login()
	if err != nil {
		fmt.Printf("error on logging in using DNS lookup of KDCs: %v\n", err)
		return
	}
	err = cl.AffirmLogin()
	if err != nil {
		fmt.Println(err)
		return
	}

	return Krb5InitiatorClient{
		client: cl,
		state:  InitiatorStart,
	}, nil
}

type Krb5InitiatorClient struct {
	state  Krb5ClientState
	client *client.Client
	subkey types.EncryptionKey
}

// Create new authenticator checksum for kerberos MechToken
func (k *Krb5InitiatorClient) newAuthenticatorChksum(flags []int) []byte {
	a := make([]byte, 24)
	binary.LittleEndian.PutUint32(a[:4], 16)
	for _, i := range flags {
		if i == gssapi.ContextFlagDeleg {
			x := make([]byte, 28-len(a))
			a = append(a, x...)
		}
		f := binary.LittleEndian.Uint32(a[20:24])
		f |= uint32(i)
		binary.LittleEndian.PutUint32(a[20:24], f)
	}
	return a
}

func (k *Krb5InitiatorClient) InitSecContext(target string, token []byte, isGSSDelegCreds bool) ([]byte, bool, error) {
	GSSAPIFlags := []int{
		ContextFlagREADY,
		gssapi.ContextFlagInteg,
		gssapi.ContextFlagMutual,
	}
	if isGSSDelegCreds {
		GSSAPIFlags = append(GSSAPIFlags, gssapi.ContextFlagDeleg)
	}
	APOptions := []int{flags.APOptionMutualRequired}

	switch k.state {
	case InitiatorStart, InitiatorRestart:
		newTarget := strings.ReplaceAll(target, "@", "/")

		tkt, sessionKey, err := k.client.GetServiceTicket(newTarget)
		if err != nil {
			return []byte{}, false, err
		}

		krb5Token, err := spnego.NewKRB5TokenAPREQ(k.client, tkt, sessionKey, GSSAPIFlags, APOptions)
		if err != nil {
			return nil, false, fmt.Errorf("error generating new kerberos 5 token: %w", err)
		}
		creds := k.client.Credentials
		auth, err := types.NewAuthenticator(creds.Domain(), creds.CName())
		if err != nil {
			return nil, false, fmt.Errorf("error generating new authenticator: %w", err)
		}
		auth.Cksum = types.Checksum{
			CksumType: chksumtype.GSSAPI,
			Checksum:  k.newAuthenticatorChksum(GSSAPIFlags),
		}
		etype, _ := crypto.GetEtype(sessionKey.KeyType)
		if err := auth.GenerateSeqNumberAndSubKey(sessionKey.KeyType, etype.GetKeyByteSize()); err != nil {
			return nil, false, err
		}
		k.subkey = auth.SubKey

		APReq, err := messages.NewAPReq(
			tkt,
			sessionKey,
			auth,
		)
		if err != nil {
			return nil, false, fmt.Errorf("error generating NewAPReq: %w", err)
		}
		for _, o := range APOptions {
			types.SetFlag(&APReq.APOptions, o)
		}
		krb5Token.APReq = APReq

		outToken, err := krb5Token.Marshal()
		if err != nil {
			fmt.Println(err)
			return []byte{}, false, err
		}
		k.state = InitiatorWaitForMutal
		return outToken, true, nil
	case InitiatorWaitForMutal:
		var krb5Token spnego.KRB5Token
		if err := krb5Token.Unmarshal(token); err != nil {
			err := fmt.Errorf("unmarshal APRep token failed: %w", err)
			return []byte{}, false, err
		}
		//var enc messages.EncAPRepPart
		//err2 := enc.Unmarshal(krb5Token.APRep.EncPart.Cipher)
		//fmt.Printf("err2: %#v, enc: %#v\n", err2, enc)

		k.state = InitiatorReady
		return []byte{}, false, nil
	case InitiatorReady:
		return nil, false, fmt.Errorf("called one time too many, client has already been %d", k.state)
	default:
		return nil, false, fmt.Errorf("invalid state %d", k.state)
	}
}

func (k *Krb5InitiatorClient) GetMIC(micFiled []byte) ([]byte, error) {
	micToken, err := gssapi.NewInitiatorMICToken(micFiled, k.subkey)
	if err != nil {
		return nil, err
	}
	token, err := micToken.Marshal()
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (k *Krb5InitiatorClient) DeleteSecContext() error {
	k.client.Destroy()
	return nil
}
