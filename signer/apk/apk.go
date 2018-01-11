package apk

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"time"

	"github.com/pkg/errors"
	"go.mozilla.org/autograph/signer"
	"go.mozilla.org/pkcs7"
)

const (
	// Type of this signer is "apk"
	Type = "apk"
)

// An APKSigner is configured to issue PKCS7 detached signatures
// for Android application packages.
type APKSigner struct {
	signer.Configuration
	signingKey  crypto.PrivateKey
	signingCert *x509.Certificate
}

// New initializes an apk signer using a configuration
func New(conf signer.Configuration) (s *APKSigner, err error) {
	s = new(APKSigner)
	if conf.Type != Type {
		return nil, errors.Errorf("apk: invalid type %q, must be %q", conf.Type, Type)
	}
	s.Type = conf.Type
	if conf.ID == "" {
		return nil, errors.New("apk: missing signer ID in signer configuration")
	}
	s.ID = conf.ID
	if conf.PrivateKey == "" {
		return nil, errors.New("apk: missing private key in signer configuration")
	}
	s.PrivateKey = conf.PrivateKey
	s.signingKey, err = signer.ParsePrivateKey([]byte(conf.PrivateKey))
	if err != nil {
		return nil, errors.Wrap(err, "apk: failed to parse private key")
	}
	block, _ := pem.Decode([]byte(conf.Certificate))
	if block == nil {
		return nil, errors.New("apk: failed to parse certificate PEM")
	}
	s.signingCert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "apk: could not parse X.509 certificate")
	}
	// APK signing certs typically have a 30y expiration, which is fairly useless
	// but requires the validity to be correct
	if time.Now().Before(s.signingCert.NotBefore) || time.Now().After(s.signingCert.NotAfter) {
		return nil, errors.New("apk: signer certificate is not currently valid")
	}
	return
}

// Config returns the configuration of the current signer
func (s *APKSigner) Config() signer.Configuration {
	return signer.Configuration{
		ID:          s.ID,
		Type:        s.Type,
		PrivateKey:  s.PrivateKey,
		Certificate: s.Certificate,
	}
}

// SignData takes input data and returns a PKCS7 detached signature
func (s *APKSigner) SignData(input []byte, options interface{}) (signer.Signature, error) {
	p7sig := new(Signature)
	toBeSigned, err := pkcs7.NewSignedData(input)
	if err != nil {
		return nil, errors.Wrap(err, "apk: cannot initialize signed data")
	}
	err = toBeSigned.AddSigner(s.signingCert, s.signingKey, pkcs7.SignerInfoConfig{})
	if err != nil {
		return nil, errors.Wrap(err, "apk: cannot sign")
	}
	toBeSigned.Detach()
	p7sig.Data, err = toBeSigned.Finish()
	if err != nil {
		return nil, errors.Wrap(err, "apk: cannot finish signing data")
	}
	p7sig.Finished = true
	return p7sig, nil
}

// Signature is a PKCS7 detached signature
type Signature struct {
	p7       *pkcs7.PKCS7
	Data     []byte
	Finished bool
}

// Marshal returns the base64 representation of a PKCS7 detached signature
func (sig *Signature) Marshal() (string, error) {
	if !sig.Finished {
		return "", errors.New("apk: cannot marshal unfinished signature")
	}
	if len(sig.Data) == 0 {
		return "", errors.New("apk: cannot marshal empty signature data")
	}
	return base64.StdEncoding.EncodeToString(sig.Data), nil
}

// Unmarshal takes the base64 representation of a PKCS7 detached signature
// and the content of the signed data, and returns a PKCS7 struct
func Unmarshal(signature string, content []byte) (sig *Signature, err error) {
	sig = new(Signature)
	sig.Data, err = base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return sig, errors.Wrap(err, "apk.Unmarshal: failed to decode base64 signature")
	}
	sig.p7, err = pkcs7.Parse(sig.Data)
	if err != nil {
		return sig, errors.Wrap(err, "apk.Unmarshal: failed to parse pkcs7 signature")
	}
	sig.p7.Content = content
	sig.Finished = true
	return
}

// String returns a PEM encoded PKCS7 block
func (sig *Signature) String() string {
	var buf bytes.Buffer
	pem.Encode(&buf, &pem.Block{Type: "PKCS7", Bytes: sig.Data})
	return string(buf.Bytes())
}
