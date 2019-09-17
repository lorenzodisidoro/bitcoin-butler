package walletlive

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
)

// is the root path to use for default derivation
// the first account will be at m/44/0/0/0 (m/44/0/ACCOUNT/0)
var (
	defaultPath = []uint32{44, 0, 0, 0}

	// public errors
	ErrDerivationPathNotFound      = errors.New("Derivation path not providded")
	ErrDerivationPathInvalidPrefix = errors.New("Use 'm/' prefix for absolute paths")
	ErrDerivationPathMalformed     = errors.New("Malformedm or empty derivation path")
)

// WalletLive wrap btcutil/hdkeychain/extendedkey package
// It's contain fields needed to support HD wallet (BIP0032)
// Reference: https://github.com/btcsuite/btcutil/blob/master/hdkeychain/extendedkey_test.go
type WalletLive struct {
	MasterPublicKey *hdkeychain.ExtendedKey
	Network         *chaincfg.Params
	Path            []uint32
}

// New create instance of WalletLive
// the param 'xPub' represented by the hexadecimal string of xPub,
// param 'path' is derivation path, eg. m/44/0/ACCOUNT/0/INDEX is []uint32{44, 0, ACCOUNT, 0, INDEX}
// and param network is bitcoin network to use during address derivation
func (wl *WalletLive) New(xPub, path, network string) error {
	masterPublicKey, err := hdkeychain.NewKeyFromString(xPub)
	if err != nil {
		return err
	}

	networkParams, err := parseNetwork(network)
	if err != nil {
		return err
	}

	wl.MasterPublicKey = masterPublicKey
	wl.Network = networkParams
	wl.Path, err = ParseDerivationPath(path)
	if err != nil {
		return err
	}

	return nil
}

// DeriveAddress derive public keys from other public key.
// function returns new address script hash or an error
func (wl *WalletLive) DeriveAddress(index uint32) (string, error) {
	var addressScript string
	var err error

	wl.Path = append(wl.Path, index)

	derivedPublicKey := wl.MasterPublicKey
	for _, childNum := range wl.Path {
		newDerivedKey, err := derivedPublicKey.Child(childNum)
		if err != nil {
			return addressScript, err
		}

		derivedPublicKey = newDerivedKey
	}

	witnessProgram, err := wl.GetPayToAddrScript(derivedPublicKey)
	addressScriptHash, err := btcutil.NewAddressScriptHash(witnessProgram, wl.Network)
	if err != nil {
		return addressScript, err
	}

	addressScript = addressScriptHash.String()

	return addressScript, nil
}

// GetPayToAddrScript creates a new script to pay a transaction output to a the specified address
// create double hash (HASH160) and execute base58 check encode
func (wl *WalletLive) GetPayToAddrScript(extendedKey *hdkeychain.ExtendedKey) ([]byte, error) {
	publicKey, err := extendedKey.ECPubKey()
	if err != nil {
		return nil, err
	}

	publicKeyBytes := publicKey.SerializeCompressed()
	publicKeyHash := btcutil.Hash160(publicKeyBytes)

	var verstion byte // 0x00 version prefix
	_, _, err = base58.CheckDecode(base58.CheckEncode(publicKeyHash, verstion))
	if err != nil {
		return nil, err
	}

	addressWitnessPubKeyHash, _ := btcutil.NewAddressWitnessPubKeyHash(publicKeyHash, wl.Network)
	witnessProgram, err := txscript.PayToAddrScript(addressWitnessPubKeyHash)
	if err != nil {
		return nil, err
	}

	return witnessProgram, nil
}

// ParseDerivationPath converts a user specified derivation path string to the uint32 array.
// Derivation paths need to start with the 'm/' prefix.
func ParseDerivationPath(path string) ([]uint32, error) {
	var result []uint32
	pathElements := strings.Split(path, "/")

	switch {
	case len(pathElements) == 0:
		return nil, ErrDerivationPathNotFound

	case strings.TrimSpace(pathElements[0]) == "":
		return nil, ErrDerivationPathInvalidPrefix

	case strings.TrimSpace(pathElements[0]) == "m":
		pathElements = pathElements[1:]

	default:
		result = append(result, defaultPath...)
	}

	if len(pathElements) == 0 {
		return nil, ErrDerivationPathMalformed
	}

	for _, element := range pathElements {
		element = strings.TrimSpace(element)
		var value uint32

		// hardened paths
		if strings.HasSuffix(element, "'") {
			value = hdkeychain.HardenedKeyStart
			element = strings.TrimSpace(strings.TrimSuffix(element, "'"))
		}

		bigval, ok := new(big.Int).SetString(element, 0)
		if !ok {
			return nil, fmt.Errorf("invalid element: %s", element)
		}

		max := math.MaxUint32 - value
		if bigval.Sign() < 0 || bigval.Cmp(big.NewInt(int64(max))) > 0 {
			if value == 0 {
				return nil, fmt.Errorf("Element %v out of allowed range [0, %d]", bigval, max)
			}

			return nil, fmt.Errorf("Element %v out of allowed hardened range [0, %d]", bigval, max)
		}

		value += uint32(bigval.Uint64())

		// append and repeat
		result = append(result, value)
	}
	return result, nil
}

// parseNetwork return chaincfg.Params by network string
func parseNetwork(network string) (*chaincfg.Params, error) {
	switch network {
	case "mainnet":
		return &chaincfg.MainNetParams, nil
	case "testnet":
		return &chaincfg.TestNet3Params, nil
	case "regtest":
		return &chaincfg.RegressionNetParams, nil
	default:
		return nil, fmt.Errorf("Unrecognized network")
	}
}
