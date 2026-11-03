package passkeytest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
)

// Authenticator is a software passkey for tests: it answers the options
// /v1/passkeys/register/begin and /v1/passkeys/login/begin return with the
// JSON a browser would post to the matching finish route. It holds one P-256
// credential, signs with "none" attestation and reports user presence and
// verification on every ceremony.
//
// It does what a platform authenticator does and nothing more careful, so
// use it to drive ceremonies end to end, never as a model of a real device.
type Authenticator struct {
	// Origin is the page origin the browser would report, such as
	// http://localhost:3000. Its host must be the RP ID or under it.
	Origin string
	// RPID is the relying party ID the credential is scoped to.
	RPID string

	key        *ecdsa.PrivateKey
	credID     []byte
	userHandle []byte
	signCount  uint32
}

// NewAuthenticator returns an authenticator with no credential yet.
func NewAuthenticator(origin, rpID string) *Authenticator {
	return &Authenticator{Origin: origin, RPID: rpID}
}

// UserHandle is the user handle the last Create stored, which a
// discoverable sign-in returns.
func (a *Authenticator) UserHandle() []byte { return a.userHandle }

// CredentialID is the ID of the credential Create made, base64url encoded
// the way authsome lists it.
func (a *Authenticator) CredentialID() string {
	return base64.RawURLEncoding.EncodeToString(a.credID)
}

const (
	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttestedData = 0x40
)

// creationOptions is the part of navigator.credentials.create's options
// the authenticator reads. authsome wraps them as {"options": ...}; pass
// what is inside.
type creationOptions struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
		User      struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"publicKey"`
}

// Create makes a new credential for the creation options and returns the
// attestation body for /v1/passkeys/register/finish. It replaces any
// credential the authenticator held.
func (a *Authenticator) Create(options json.RawMessage) ([]byte, error) {
	var opts creationOptions
	if err := json.Unmarshal(options, &opts); err != nil {
		return nil, fmt.Errorf("passkeytest: decode creation options: %w", err)
	}
	if opts.PublicKey.Challenge == "" {
		return nil, errors.New("passkeytest: creation options carry no challenge")
	}
	handle, err := base64.RawURLEncoding.DecodeString(opts.PublicKey.User.ID)
	if err != nil {
		return nil, fmt.Errorf("passkeytest: decode user handle: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	credID := make([]byte, 32)
	if _, err = rand.Read(credID); err != nil {
		return nil, err
	}
	a.key, a.credID, a.userHandle, a.signCount = key, credID, handle, 0

	ecdh, err := key.PublicKey.ECDH()
	if err != nil {
		return nil, err
	}
	point := ecdh.Bytes() // 0x04 || X || Y
	cose, err := webauthncbor.Marshal(webauthncose.EC2PublicKeyData{
		PublicKeyData: webauthncose.PublicKeyData{
			KeyType:   int64(webauthncose.EllipticKey),
			Algorithm: int64(webauthncose.AlgES256),
		},
		Curve:  int64(webauthncose.P256),
		XCoord: point[1:33],
		YCoord: point[33:],
	})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode public key: %w", err)
	}
	authData := a.authData(flagUserPresent | flagUserVerified | flagAttestedData)
	authData = append(authData, make([]byte, 16)...) // AAGUID, none
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(credID)))
	authData = append(authData, credID...)
	authData = append(authData, cose...)

	attestation, err := webauthncbor.Marshal(map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": authData,
	})
	if err != nil {
		return nil, fmt.Errorf("passkeytest: encode attestation: %w", err)
	}
	clientData, err := a.clientData("webauthn.create", opts.PublicKey.Challenge)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"id":    a.CredentialID(),
		"rawId": a.CredentialID(),
		"type":  "public-key",
		"response": map[string]string{
			"clientDataJSON":    b64(clientData),
			"attestationObject": b64(attestation),
		},
	})
}

// requestOptions is the part of navigator.credentials.get's options the
// authenticator reads.
type requestOptions struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
	} `json:"publicKey"`
}

// Get signs the request options' challenge with the credential Create made
// and returns the assertion body for /v1/passkeys/login/finish, user handle
// included, as a discoverable credential answers.
func (a *Authenticator) Get(options json.RawMessage) ([]byte, error) {
	if a.key == nil {
		return nil, errors.New("passkeytest: no credential; call Create first")
	}
	var opts requestOptions
	if err := json.Unmarshal(options, &opts); err != nil {
		return nil, fmt.Errorf("passkeytest: decode request options: %w", err)
	}
	if opts.PublicKey.Challenge == "" {
		return nil, errors.New("passkeytest: request options carry no challenge")
	}
	a.signCount++
	authData := a.authData(flagUserPresent | flagUserVerified)
	clientData, err := a.clientData("webauthn.get", opts.PublicKey.Challenge)
	if err != nil {
		return nil, err
	}
	clientHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte{}, authData...), clientHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"id":    a.CredentialID(),
		"rawId": a.CredentialID(),
		"type":  "public-key",
		"response": map[string]string{
			"clientDataJSON":    b64(clientData),
			"authenticatorData": b64(authData),
			"signature":         b64(sig),
			"userHandle":        b64(a.userHandle),
		},
	})
}

// authData is the fixed head of authenticator data: the RP ID hash, the
// flags and the signature counter.
func (a *Authenticator) authData(flags byte) []byte {
	rpHash := sha256.Sum256([]byte(a.RPID))
	out := append([]byte{}, rpHash[:]...)
	out = append(out, flags)
	return binary.BigEndian.AppendUint32(out, a.signCount)
}

func (a *Authenticator) clientData(kind, challenge string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":        kind,
		"challenge":   challenge,
		"origin":      a.Origin,
		"crossOrigin": false,
	})
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
