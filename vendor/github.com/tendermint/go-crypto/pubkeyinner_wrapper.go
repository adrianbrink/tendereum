// Generated by: main
// TypeWriter: wrapper
// Directive: +gen on PubKeyInner

package crypto

import (
	"github.com/tendermint/go-wire/data"
)

// Auto-generated adapters for happily unmarshaling interfaces
// Apache License 2.0
// Copyright (c) 2017 Ethan Frey (ethan.frey@tendermint.com)

type PubKey struct {
	PubKeyInner "json:\"unwrap\""
}

var PubKeyMapper = data.NewMapper(PubKey{})

func (h PubKey) MarshalJSON() ([]byte, error) {
	return PubKeyMapper.ToJSON(h.PubKeyInner)
}

func (h *PubKey) UnmarshalJSON(data []byte) (err error) {
	parsed, err := PubKeyMapper.FromJSON(data)
	if err == nil && parsed != nil {
		h.PubKeyInner = parsed.(PubKeyInner)
	}
	return err
}

// Unwrap recovers the concrete interface safely (regardless of levels of embeds)
func (h PubKey) Unwrap() PubKeyInner {
	hi := h.PubKeyInner
	for wrap, ok := hi.(PubKey); ok; wrap, ok = hi.(PubKey) {
		hi = wrap.PubKeyInner
	}
	return hi
}

func (h PubKey) Empty() bool {
	return h.PubKeyInner == nil
}

/*** below are bindings for each implementation ***/

func init() {
	PubKeyMapper.RegisterImplementation(PubKeyEd25519{}, "ed25519", 0x1)
}

func (hi PubKeyEd25519) Wrap() PubKey {
	return PubKey{hi}
}

func init() {
	PubKeyMapper.RegisterImplementation(PubKeySecp256k1{}, "secp256k1", 0x2)
}

func (hi PubKeySecp256k1) Wrap() PubKey {
	return PubKey{hi}
}
