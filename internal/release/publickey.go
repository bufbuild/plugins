package release

import (
	_ "embed"

	"aead.dev/minisign"
)

//go:embed minisign.pub
var minisignPublicKey []byte

func DefaultPublicKey() (minisign.PublicKey, error) {
	var publicKey minisign.PublicKey
	if err := publicKey.UnmarshalText(minisignPublicKey); err != nil {
		return minisign.PublicKey{}, err
	}
	return publicKey, nil
}
