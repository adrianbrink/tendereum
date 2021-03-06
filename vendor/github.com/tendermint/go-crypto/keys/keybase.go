package keys

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	crypto "github.com/tendermint/go-crypto"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/tendermint/go-crypto/keys/words"
	"github.com/tendermint/go-crypto/nano"
)

// XXX Lets use go-crypto/bcrypt and ascii encoding directly in here without
// further wrappers around a store or DB.
// Copy functions from: https://github.com/tendermint/mintkey/blob/master/cmd/mintkey/common.go
//
// dbKeybase combines encyption and storage implementation to provide
// a full-featured key manager
type dbKeybase struct {
	db    dbm.DB
	codec words.Codec
}

func New(db dbm.DB, codec words.Codec) dbKeybase {
	return dbKeybase{
		db:    db,
		codec: codec,
	}
}

var _ Keybase = dbKeybase{}

// Create generates a new key and persists it storage, encrypted using the passphrase.
// It returns the generated seedphrase (mnemonic) and the key Info.
// It returns an error if it fails to generate a key for the given algo type,
// or if another key is already stored under the same name.
func (kb dbKeybase) Create(name, passphrase, algo string) (string, Info, error) {
	// NOTE: secret is SHA256 hashed by secp256k1 and ed25519.
	// 16 byte secret corresponds to 12 BIP39 words.
	// XXX: Ledgers use 24 words now - should we ?
	secret := crypto.CRandBytes(16)
	key, err := generate(algo, secret)
	if err != nil {
		return "", Info{}, err
	}

	// encrypt and persist the key
	public := kb.writeKey(key, name, passphrase)

	// return the mnemonic phrase
	words, err := kb.codec.BytesToWords(secret)
	seedphrase := strings.Join(words, " ")
	return seedphrase, public, err
}

// Recover converts a seedphrase to a private key and persists it, encrypted with the given passphrase.
// Functions like Create, but seedphrase is input not output.
func (kb dbKeybase) Recover(name, passphrase, algo string, seedphrase string) (Info, error) {

	key, err := kb.SeedToPrivKey(algo, seedphrase)
	if err != nil {
		return Info{}, err
	}

	// Valid seedphrase. Encrypt key and persist to disk.
	public := kb.writeKey(key, name, passphrase)
	return public, nil
}

// SeedToPrivKey returns the private key corresponding to a seedphrase
// without persisting the private key.
// TODO: enable the keybase to just hold these in memory so we can sign without persisting (?)
func (kb dbKeybase) SeedToPrivKey(algo, seedphrase string) (crypto.PrivKey, error) {
	words := strings.Split(strings.TrimSpace(seedphrase), " ")
	secret, err := kb.codec.WordsToBytes(words)
	if err != nil {
		return crypto.PrivKey{}, err
	}

	key, err := generate(algo, secret)
	if err != nil {
		return crypto.PrivKey{}, err
	}
	return key, nil
}

// List returns the keys from storage in alphabetical order.
func (kb dbKeybase) List() ([]Info, error) {
	var res []Info
	iter := kb.db.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if isPub(key) {
			info, err := readInfo(iter.Value())
			if err != nil {
				return nil, err
			}
			res = append(res, info)
		}
	}
	return res, nil
}

// Get returns the public information about one key.
func (kb dbKeybase) Get(name string) (Info, error) {
	bs := kb.db.Get(pubName(name))
	return readInfo(bs)
}

// Sign signs the msg with the named key.
// It returns an error if the key doesn't exist or the decryption fails.
// TODO: what if leddger fails ?
func (kb dbKeybase) Sign(name, passphrase string, msg []byte) (sig crypto.Signature, pk crypto.PubKey, err error) {
	var key crypto.PrivKey
	armorStr := kb.db.Get(privName(name))
	key, err = unarmorDecryptPrivKey(string(armorStr), passphrase)
	if err != nil {
		return
	}

	sig = key.Sign(msg)
	pk = key.PubKey()
	return
}

// Export decodes the private key with the current password, encrypts
// it with a secure one-time password and generates an armored private key
// that can be Imported by another dbKeybase.
//
// This is designed to copy from one device to another, or provide backups
// during version updates.
func (kb dbKeybase) Export(name, oldpass, transferpass string) ([]byte, error) {
	armorStr := kb.db.Get(privName(name))
	key, err := unarmorDecryptPrivKey(string(armorStr), oldpass)
	if err != nil {
		return nil, err
	}

	if transferpass == "" {
		return key.Bytes(), nil
	}
	armorBytes := encryptArmorPrivKey(key, transferpass)
	return []byte(armorBytes), nil
}

// Import accepts bytes generated by Export along with the same transferpass.
// If they are valid, it stores the password under the given name with the
// new passphrase.
func (kb dbKeybase) Import(name, newpass, transferpass string, data []byte) (err error) {
	var key crypto.PrivKey
	if transferpass == "" {
		key, err = crypto.PrivKeyFromBytes(data)
	} else {
		key, err = unarmorDecryptPrivKey(string(data), transferpass)
	}
	if err != nil {
		return err
	}

	kb.writeKey(key, name, newpass)
	return nil
}

// Delete removes key forever, but we must present the
// proper passphrase before deleting it (for security).
func (kb dbKeybase) Delete(name, passphrase string) error {
	// verify we have the proper password before deleting
	bs := kb.db.Get(privName(name))
	_, err := unarmorDecryptPrivKey(string(bs), passphrase)
	if err != nil {
		return err
	}
	kb.db.DeleteSync(pubName(name))
	kb.db.DeleteSync(privName(name))
	return nil
}

// Update changes the passphrase with which an already stored key is encrypted.
//
// oldpass must be the current passphrase used for encryption, newpass will be
// the only valid passphrase from this time forward.
func (kb dbKeybase) Update(name, oldpass, newpass string) error {
	bs := kb.db.Get(privName(name))
	key, err := unarmorDecryptPrivKey(string(bs), oldpass)
	if err != nil {
		return err
	}

	// Generate the public bytes and the encrypted privkey
	public := info(name, key)
	private := encryptArmorPrivKey(key, newpass)

	// We must delete first, as Putting over an existing name returns an error.
	// Must be done atomically with the write or we could lose the key.
	batch := kb.db.NewBatch()
	batch.Delete(pubName(name))
	batch.Delete(privName(name))
	batch.Set(pubName(name), public.bytes())
	batch.Set(privName(name), []byte(private))
	batch.Write()

	return nil
}

//---------------------------------------------------------------------------------------

func (kb dbKeybase) writeKey(priv crypto.PrivKey, name, passphrase string) Info {
	// Generate the public bytes and the encrypted privkey
	public := info(name, priv)
	private := encryptArmorPrivKey(priv, passphrase)

	// Write them both
	kb.db.SetSync(pubName(name), public.bytes())
	kb.db.SetSync(privName(name), []byte(private))

	return public
}

// TODO: use a `type TypeKeyAlgo string` (?)
func generate(algo string, secret []byte) (crypto.PrivKey, error) {
	switch algo {
	case crypto.NameEd25519:
		return crypto.GenPrivKeyEd25519FromSecret(secret).Wrap(), nil
	case crypto.NameSecp256k1:
		return crypto.GenPrivKeySecp256k1FromSecret(secret).Wrap(), nil
	case nano.NameLedgerEd25519:
		return nano.NewPrivKeyLedgerEd25519()
	default:
		err := errors.Errorf("Cannot generate keys for algorithm: %s", algo)
		return crypto.PrivKey{}, err
	}
}

func pubName(name string) []byte {
	return []byte(fmt.Sprintf("%s.pub", name))
}

func privName(name string) []byte {
	return []byte(fmt.Sprintf("%s.priv", name))
}

func isPub(name []byte) bool {
	return strings.HasSuffix(string(name), ".pub")
}
