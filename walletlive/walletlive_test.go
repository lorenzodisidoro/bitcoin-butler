package walletlive

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcutil/hdkeychain"
)

func TestDeriveAddress(t *testing.T) {
	masterPubKey1 := "xpub6ERApfZwUNrhLCkDtcHTcxd75RbzS1ed54G1LkBUHQVHQKqhMkhgbmJbZRkrgZw4koxb5JaHWkY4ALHY2grBGRjaDMzQLcgJvLJuZZvRcEL"
	network := "mainnet"
	address1Case := "3AyAF4gizs179HV1dwdKH6ZGMffkpBK3LD"
	address2Case := "3EnYNCHDnfPcgzC927JwDhKVZGYfPvg763"
	derivationBasePath := "m/44/0/1/0"

	wl := &WalletLive{}
	err := wl.New(masterPubKey1, derivationBasePath, network)
	if err != nil {
		t.Fatal(err)
	}

	address1, err := wl.DeriveAddress(1)
	if err != nil {
		t.Fatal(err)
	}

	address2, err := wl.DeriveAddress(2)
	if err != nil {
		t.Fatal(err)
	}

	if address1 == address2 {
		t.Fatal("Should be expected different address")
	}

	if address1 != address1Case || address2 != address2Case {
		t.Fatal("Should be expected the same address")
	}
}

func TestParseDerivationPath(t *testing.T) {
	// hardened paths
	derivationPathHardStr := "m/44'/1/2'"
	wantHardDerivationPath := []uint32{hdkeychain.HardenedKeyStart + 44, 1, hdkeychain.HardenedKeyStart + 2}

	derivationHardPath, err := ParseDerivationPath(derivationPathHardStr)
	if err != nil {
		t.Fatal(err)
	}

	for i, element := range derivationHardPath {
		if element != wantHardDerivationPath[i] {
			t.Fatal("Should be expected the same path")
		}
	}

	// not hardened paths
	derivationPathStr := "m/44/0/1/0/1"
	wantDerivationPath := []uint32{44, 0, 1, 0, 1}

	derivationPath, err := ParseDerivationPath(derivationPathStr)
	if err != nil {
		t.Fatal(err)
	}

	for i, element := range derivationPath {
		if element != wantDerivationPath[i] {
			t.Fatal("Should be expected the same path")
		}
	}
}

func TestParsedPath(t *testing.T) {
	masterPrivateKey := "xprv9s21ZrQH143K3QTDL4LXw2F7HEK3wJUD2nW2nRk4stbPy6cq3jPPqjiChkVvvNKmPGJxWUtg6LnF5kejMRNNU3TGtRBeJgk33yuGBxrMPHi"
	wantExtendedKeyAcc0 := "xprv9wTYmMFdV23N21MM6dLNavSQV7Sj7meSPXx6AV5eTdqqGLjycVjb115Ec5LgRAXscPZgy5G4jQ9csyyZLN3PZLxoM1h3BoPuEJzsgeypdKj" // account 0 external chain

	masterKey, err := hdkeychain.NewKeyFromString(masterPrivateKey)
	if err != nil {
		fmt.Println(err)
	}

	path := "m/0'/0" // account 0 external chain
	derivationPath, err := ParseDerivationPath(path)
	if err != nil {
		t.Fatal(err)
	}

	acct0, err := masterKey.Child(derivationPath[0])
	if err != nil {
		t.Fatal(err)
	}

	acct0Ext, err := acct0.Child(derivationPath[1])
	if err != nil {
		t.Fatal(err)
	}

	if wantExtendedKeyAcc0 != acct0Ext.String() {
		t.Fatal("Should be expected the same account address")
	}
}
