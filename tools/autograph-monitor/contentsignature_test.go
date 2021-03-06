package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestVerifyContentSignature(t *testing.T) {
	go func() {
		http.HandleFunc("/normandychain", func(w http.ResponseWriter, r *http.Request) {
			chain, err := ioutil.ReadFile(os.Getenv("GOPATH") + `/src/go.mozilla.org/autograph/docs/statics/normandy.content-signature.mozilla.org-20210705.dev.chain`)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(w, "%s", chain)
		})
		log.Fatal(http.ListenAndServe(":64320", nil))
	}()
	err := verifyContentSignature(ValidMonitoringContentSignature)
	if err != nil {
		t.Fatalf("Failed to verify monitoring content signature: %v", err)
	}
}

func TestVerifyExpiredCertChain(t *testing.T) {
	go func() {
		http.HandleFunc("/expiredcertchain", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, ExpiredEndEntityChain)
		})
		log.Fatal(http.ListenAndServe(":64321", nil))
	}()
	chain, err := getX5U("http://localhost:64321/expiredcertchain")
	if err != nil {
		t.Fatalf("Failed to retrieved certificate chain: %v", err)
	}
	err = verifyCertChain(chain)
	if err == nil {
		t.Fatal("Expected to fail chain verification with expired end-entity, but succeeded")
	}
	log.Printf("Chain verification failed with: %v", err)
	if !strings.Contains(err.Error(), "expires in less than 15 days") {
		t.Fatalf("Expected to failed with expired end-entity but failed with: %v", err)
	}
}

func TestVerifyWronglyOrderedChain(t *testing.T) {
	go func() {
		http.HandleFunc("/wronglyorderedchain", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, WronglyOrderedChain)
		})
		log.Fatal(http.ListenAndServe(":64322", nil))
	}()
	chain, err := getX5U("http://localhost:64322/wronglyorderedchain")
	if err != nil {
		t.Fatalf("Failed to retrieved certificate chain: %v", err)
	}
	err = verifyCertChain(chain)
	if err == nil {
		t.Fatal("Expected to fail chain verification with cert not signed by parent, but succeeded")
	}
	log.Printf("Chain verification failed with: %v", err)
	if !strings.Contains(err.Error(), "is not signed by parent certificate") {
		t.Fatalf("Expected to failed with certificate not being signed by parent, but failed with: %v", err)
	}
}

func TestVerifyFirefoxRoot(t *testing.T) {
	conf.RootHash = "97:E8:BA:9C:F1:2F:B3:DE:53:CC:42:A4:E6:57:7E:D6:4D:F4:93:C2:47:B4:14:FE:A0:36:81:8D:38:23:56:0E"
	block, _ := pem.Decode(FirefoxPKIRootPEM)
	if block == nil {
		t.Fatalf("Failed to parse certificate PEM")
	}
	certX509, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Could not parse X.509 certificate: %v", err)
	}
	err = verifyRoot(certX509)
	if err != nil {
		t.Fatalf("Failed to verify valid Firefox root certificate: %v", err)
	}
}

func TestVerifyFirefoxRootWithBadHash(t *testing.T) {
	conf.RootHash = "foo"
	block, _ := pem.Decode(FirefoxPKIRootPEM)
	if block == nil {
		t.Fatalf("Failed to parse certificate PEM")
	}
	certX509, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Could not parse X.509 certificate: %v", err)
	}
	err = verifyRoot(certX509)
	if err == nil {
		t.Fatalf("Expected to fail with incorrect hash, but succeeded.")
	}
	if !strings.Contains(err.Error(), "hash does not match expected root") {
		t.Fatalf("Expected to fail with hash mismatch, but failed with: %v", err)
	}
}

func TestVerifyFirefoxStagingRoot(t *testing.T) {
	conf.RootHash = "DB:74:CE:58:E4:F9:D0:9E:E0:42:36:BE:6C:C5:C4:F6:6A:E7:74:7D:C0:21:42:7A:03:BC:2F:57:0C:8B:9B:90"
	block, _ := pem.Decode(FirefoxPKIStagingRootPEM)
	if block == nil {
		t.Fatalf("Failed to parse certificate PEM")
	}
	certX509, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Could not parse X.509 certificate: %v", err)
	}
	err = verifyRoot(certX509)
	if err != nil {
		t.Fatalf("Failed to verify valid Firefox root certificate: %v", err)
	}
}

// fixtures -----------------------------------------------------------------

var ValidMonitoringContentSignature = signatureresponse{
	Ref:       "1881ks1du39bi26cfmfczu6pf3",
	Type:      "contentsignature",
	Mode:      "p384ecdsa",
	SignerID:  "normankey",
	PublicKey: "MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEVEKiCAIkwRg1VFsP8JOYdSF6a3qvgbRPoEK9eTuLbrB6QixozscKR4iWJ8ZOOX6RPCRgFdfVDoZqjFBFNJN9QtRBk0mVtHbnErx64d2vMF0oWencS1hyLW2whgOgOz7p",
	Signature: "9M26T-1RCEzTAlCzDZk6CkEZxkVZkt-wUJfA4s4altKx3Vw-MfuE08bXy1TenbR0I87PzuuA9c1CNOZ8hzRbVuYvKnOH0z4kIbGzAMWzyOxwRgufaODHpcnSAKv2q3JM",
	X5U:       "http://127.0.0.1:64320/normandychain",
}

// this is the trusted root ca for the firefox pki
var FirefoxPKIRootPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIGYTCCBEmgAwIBAgIBATANBgkqhkiG9w0BAQwFADB9MQswCQYDVQQGEwJVUzEc
MBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0GA1UECxMmTW96aWxsYSBB
TU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAdBgNVBAMTFnJvb3QtY2Et
cHJvZHVjdGlvbi1hbW8wHhcNMTUwMzE3MjI1MzU3WhcNMjUwMzE0MjI1MzU3WjB9
MQswCQYDVQQGEwJVUzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0G
A1UECxMmTW96aWxsYSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAd
BgNVBAMTFnJvb3QtY2EtcHJvZHVjdGlvbi1hbW8wggIgMA0GCSqGSIb3DQEBAQUA
A4ICDQAwggIIAoICAQC0u2HXXbrwy36+MPeKf5jgoASMfMNz7mJWBecJgvlTf4hH                                                                                                                                                                                               
JbLzMPsIUauzI9GEpLfHdZ6wzSyFOb4AM+D1mxAWhuZJ3MDAJOf3B1Rs6QorHrl8                                                                                                                                                                                               
qqlNtPGqepnpNJcLo7JsSqqE3NUm72MgqIHRgTRsqUs+7LIPGe7262U+N/T0LPYV                                                                                                                                                                                               
Le4rZ2RDHoaZhYY7a9+49mHOI/g2YFB+9yZjE+XdplT2kBgA4P8db7i7I0tIi4b0                                                                                                                                                                                               
B0N6y9MhL+CRZJyxdFe2wBykJX14LsheKsM1azHjZO56SKNrW8VAJTLkpRxCmsiT                                                                                                                                                                                               
r08fnPyDKmaeZ0BtsugicdipcZpXriIGmsZbI12q5yuwjSELdkDV6Uajo2n+2ws5                                                                                                                                                                                               
uXrP342X71WiWhC/dF5dz1LKtjBdmUkxaQMOP/uhtXEKBrZo1ounDRQx1j7+SkQ4                                                                                                                                                                                               
BEwjB3SEtr7XDWGOcOIkoJZWPACfBLC3PJCBWjTAyBlud0C5n3Cy9regAAnOIqI1                                                                                                                                                                                               
t16GU2laRh7elJ7gPRNgQgwLXeZcFxw6wvyiEcmCjOEQ6PM8UQjthOsKlszMhlKw                                                                                                                                                                                               
vjyOGDoztkqSBy/v+Asx7OW2Q7rlVfKarL0mREZdSMfoy3zTgtMVCM0vhNl6zcvf                                                                                                                                                                                               
5HNNopoEdg5yuXo2chZ1p1J+q86b0G5yJRMeT2+iOVY2EQ37tHrqUURncCy4uwIB                                                                                                                                                                                               
A6OB7TCB6jAMBgNVHRMEBTADAQH/MA4GA1UdDwEB/wQEAwIBBjAWBgNVHSUBAf8E                                                                                                                                                                                               
DDAKBggrBgEFBQcDAzCBkgYDVR0jBIGKMIGHoYGBpH8wfTELMAkGA1UEBhMCVVMx                                                                                                                                                                                               
HDAaBgNVBAoTE01vemlsbGEgQ29ycG9yYXRpb24xLzAtBgNVBAsTJk1vemlsbGEg                                                                                                                                                                                               
QU1PIFByb2R1Y3Rpb24gU2lnbmluZyBTZXJ2aWNlMR8wHQYDVQQDExZyb290LWNh                                                                                                                                                                                               
LXByb2R1Y3Rpb24tYW1vggEBMB0GA1UdDgQWBBSzvOpYdKvhbngqsqucIx6oYyyX                                                                                                                                                                                               
tzANBgkqhkiG9w0BAQwFAAOCAgEAaNSRYAaECAePQFyfk12kl8UPLh8hBNidP2H6
KT6O0vCVBjxmMrwr8Aqz6NL+TgdPmGRPDDLPDpDJTdWzdj7khAjxqWYhutACTew5
eWEaAzyErbKQl+duKvtThhV2p6F6YHJ2vutu4KIciOMKB8dslIqIQr90IX2Usljq
8Ttdyf+GhUmazqLtoB0GOuESEqT4unX6X7vSGu1oLV20t7t5eCnMMYD67ZBn0YIU
/cm/+pan66hHrja+NeDGF8wabJxdqKItCS3p3GN1zUGuJKrLykxqbOp/21byAGog
Z1amhz6NHUcfE6jki7sM7LHjPostU5ZWs3PEfVVgha9fZUhOrIDsyXEpCWVa3481
LlAq3GiUMKZ5DVRh9/Nvm4NwrTfB3QkQQJCwfXvO9pwnPKtISYkZUqhEqvXk5nBg
QCkDSLDjXTx39naBBGIVIqBtKKuVTla9enngdq692xX/CgO6QJVrwpqdGjebj5P8
5fNZPABzTezG3Uls5Vp+4iIWVAEDkK23cUj3c/HhE+Oo7kxfUeu5Y1ZV3qr61+6t
ZARKjbu1TuYQHf0fs+GwID8zeLc2zJL7UzcHFwwQ6Nda9OJN4uPAuC/BKaIpxCLL
26b24/tRam4SJjqpiq20lynhUrmTtt6hbG3E1Hpy3bmkt2DYnuMFwEx2gfXNcnbT
wNuvFqc=
-----END CERTIFICATE-----`)

// this is the trusted root ca for the staging firefox pki
var FirefoxPKIStagingRootPEM = []byte(`-----BEGIN CERTIFICATE-----
MIIHYzCCBUugAwIBAgIBATANBgkqhkiG9w0BAQwFADCBqDELMAkGA1UEBhMCVVMx
CzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1Nb3VudGFpbiBWaWV3MRwwGgYDVQQKExNB
ZGRvbnMgVGVzdCBTaWduaW5nMSQwIgYDVQQDExt0ZXN0LmFkZG9ucy5zaWduaW5n
LnJvb3QuY2ExMDAuBgkqhkiG9w0BCQEWIW9wc2VjK3N0YWdlcm9vdGFkZG9uc0Bt
b3ppbGxhLmNvbTAeFw0xNTAyMTAxNTI4NTFaFw0yNTAyMDcxNTI4NTFaMIGoMQsw
CQYDVQQGEwJVUzELMAkGA1UECBMCQ0ExFjAUBgNVBAcTDU1vdW50YWluIFZpZXcx
HDAaBgNVBAoTE0FkZG9ucyBUZXN0IFNpZ25pbmcxJDAiBgNVBAMTG3Rlc3QuYWRk
b25zLnNpZ25pbmcucm9vdC5jYTEwMC4GCSqGSIb3DQEJARYhb3BzZWMrc3RhZ2Vy
b290YWRkb25zQG1vemlsbGEuY29tMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIIC
CgKCAgEAv/OSHh5uUMMKKuBh83kikuJ+BW4fQCHVZvADZh2qHNH8pSaME/YqMItP
5XQ1N5oLq1tRQO77AKn+eYPDAQkg+9VV+ct4u76YctcU/gvjieGKQ0fvuDH18QLD
hqa4DHgDmpCa/w+Eqzd54HaFj7ew9Bb7GZPHuZfk7Ct9fcN6kHneEj3KeuLiqzSV
VCRFV9RTlrUdsc1/VwF4A97JTXc3HJeWJO3azOlFpaJ8QHhmgXLLmB59HPeZ10Sf
9QwVGaKcn7yLuwtIA+wDhs8iwGZWcgmknW4DkkRDbQo7L+//4kVK+Yqq0HamZArm
vE4xENvbwOze4XYkCO3PwgmCotU7K5D3sMUUxkOaodlemO9OqRW8vJOJH3b6mhST
aunQR9/GOJ7sl4egrn2fOVZhBvM29lyBCKBffeQgtIMcKpeEKa4TNx4nTrWu1J9k
jHlvNeVL3FzMzJXRPl0RV71cYak+G6GnQ4fg3+4ZSSPxTvbwRJAO2xajkURxFSZo
sXcjYG8iPTSrDazj4LN2+882t4Q2/rMYpkowwLGbvJqHiw2tg9/hpLn1K4W18vcC
vFgzNRrTdKaJ/KjD17eJl8s8oPA7TiophPeezy1WzAc4mdlXS6A85b0mKDDU2A/4
3YmltjsSmizR2LnfeNs125EsCWxSUrAsnUYRO+lJOyNr7GGKGscCAwZVN6OCAZQw
ggGQMAwGA1UdEwQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMBYGA1UdJQEB/wQMMAoG
CCsGAQUFBwMDMCwGCWCGSAGG+EIBDQQfFh1PcGVuU1NMIEdlbmVyYXRlZCBDZXJ0
aWZpY2F0ZTAzBglghkgBhvhCAQQEJhYkaHR0cDovL2FkZG9ucy5tb3ppbGxhLm9y
Zy9jYS9jcmwucGVtMB0GA1UdDgQWBBSE6l/Nb0ySL+rR9PXIo7LCDLqm9jCB1QYD
VR0jBIHNMIHKgBSE6l/Nb0ySL+rR9PXIo7LCDLqm9qGBrqSBqzCBqDELMAkGA1UE
BhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1Nb3VudGFpbiBWaWV3MRwwGgYD
VQQKExNBZGRvbnMgVGVzdCBTaWduaW5nMSQwIgYDVQQDExt0ZXN0LmFkZG9ucy5z
aWduaW5nLnJvb3QuY2ExMDAuBgkqhkiG9w0BCQEWIW9wc2VjK3N0YWdlcm9vdGFk
ZG9uc0Btb3ppbGxhLmNvbYIBATANBgkqhkiG9w0BAQwFAAOCAgEAck21RaAcTzbT
vmqqcCezBd5Gej6jV53HItXfF06tLLzAxKIU1loLH/330xDdOGyiJdvUATDVn8q6
5v4Kae2awON6ytWZp9b0sRdtlLsRo8EWOoRszCqiMWdl1gnGMaV7e2ycz/tR+PoK
GxHCh8rbOtG0eiVJIyRijLDjtExW8Eg+uz6Zkg1IWXqInj7Gqr23FOqD76uAfE82
YTWW3lzxpP3gL7pmV5G7ob/tIyAfrPEB4w0Nt2HEl9h7NDtKPMprrOLPkrI9eAVU
QeeI3RpAKnXOFQkqPYPXIlAaJ6qxtYa6tWHOqRyS1xKnvy/uWjEtU3tYJ5eUL1+2
vzNTdakJgkZDRdDNg0V3NYwza6BwL80VPSfqc1H6R8CU1uj+kjTlCEsoTPLeW7k5
t+lKHFMj0HZLNymgDD5f9UpI7yiOAIF0z4WKAMv/f12vnAPwmOPuOikRNOv0nNuL
RIpKO53Cd7aV5PdB0pNSPNjc6V+5IPrepALNQhKIpzoHA4oG+LlVVy4R3csPcj4e
zQQ9gt3NC2OXF4hveHfKZdCnb+BBl4S71QMYYCCTe+EDCsIGuyXWD/K2hfLD8TPW
thPX5WNsS8bwno2ccqncVLQ4PZxOIB83DFBFmAvTuBiAYWq874rneTXqInHyeCq+
819l9s72pDsFaGevmm0Us9bYuufTS5U=
-----END CERTIFICATE-----`)

// This chain has an expired end-entity certificate
var ExpiredEndEntityChain = `-----BEGIN CERTIFICATE-----
MIIEnTCCBCSgAwIBAgIEAQAAFzAKBggqhkjOPQQDAzCBpjELMAkGA1UEBhMCVVMx
HDAaBgNVBAoTE01vemlsbGEgQ29ycG9yYXRpb24xLzAtBgNVBAsTJk1vemlsbGEg
QU1PIFByb2R1Y3Rpb24gU2lnbmluZyBTZXJ2aWNlMSUwIwYDVQQDExxDb250ZW50
IFNpZ25pbmcgSW50ZXJtZWRpYXRlMSEwHwYJKoZIhvcNAQkBFhJmb3hzZWNAbW96
aWxsYS5jb20wHhcNMTcwNTA5MTQwMjM3WhcNMTcxMTA3MTQwMjM3WjCBrzELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExHDAaBgNVBAoTE01vemlsbGEg
Q29ycG9yYXRpb24xFzAVBgNVBAsTDkNsb3VkIFNlcnZpY2VzMS8wLQYDVQQDEyZu
b3JtYW5keS5jb250ZW50LXNpZ25hdHVyZS5tb3ppbGxhLm9yZzEjMCEGCSqGSIb3
DQEJARYUc2VjdXJpdHlAbW96aWxsYS5vcmcwdjAQBgcqhkjOPQIBBgUrgQQAIgNi
AAShRFsGyg6DkUX+J2mMDM6cLK8V6HawjGVlQ/w5H5fHiGJDMrkl4ktnN+O37mSs
dReHcVxxpPNEpIfkWQ2TFmJgOUzqi/CzO06APlAJ9mnIcaobgdqRQxoTchFEyzUx
nTijggIWMIICEjAdBgNVHQ4EFgQUKnGLJ9po8ea5qUNjJyV/c26VZfswgaoGA1Ud
IwSBojCBn4AUiHVymVvwUPJguD2xCZYej3l5nu6hgYGkfzB9MQswCQYDVQQGEwJV
UzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0GA1UECxMmTW96aWxs
YSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAdBgNVBAMTFnJvb3Qt
Y2EtcHJvZHVjdGlvbi1hbW+CAxAABjAMBgNVHRMBAf8EAjAAMA4GA1UdDwEB/wQE
AwIHgDAWBgNVHSUBAf8EDDAKBggrBgEFBQcDAzBFBgNVHR8EPjA8MDqgOKA2hjRo
dHRwczovL2NvbnRlbnQtc2lnbmF0dXJlLmNkbi5tb3ppbGxhLm5ldC9jYS9jcmwu
cGVtMEMGCWCGSAGG+EIBBAQ2FjRodHRwczovL2NvbnRlbnQtc2lnbmF0dXJlLmNk
bi5tb3ppbGxhLm5ldC9jYS9jcmwucGVtME8GCCsGAQUFBwEBBEMwQTA/BggrBgEF
BQcwAoYzaHR0cHM6Ly9jb250ZW50LXNpZ25hdHVyZS5jZG4ubW96aWxsYS5uZXQv
Y2EvY2EucGVtMDEGA1UdEQQqMCiCJm5vcm1hbmR5LmNvbnRlbnQtc2lnbmF0dXJl
Lm1vemlsbGEub3JnMAoGCCqGSM49BAMDA2cAMGQCMGeeyXYM3+r1fcaXzd90PwGb
h9nrl1fZNXrCu17lCPn2JntBVh7byT3twEbr+Hmv8gIwU9klAW6yHLG/ZpAZ0jdf
38Rciz/FDEAdrzH2QlYAOw+uDdpcmon9oiRgIxzwNlUe
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIFfjCCA2agAwIBAgIDEAAGMA0GCSqGSIb3DQEBDAUAMH0xCzAJBgNVBAYTAlVT
MRwwGgYDVQQKExNNb3ppbGxhIENvcnBvcmF0aW9uMS8wLQYDVQQLEyZNb3ppbGxh
IEFNTyBQcm9kdWN0aW9uIFNpZ25pbmcgU2VydmljZTEfMB0GA1UEAxMWcm9vdC1j
YS1wcm9kdWN0aW9uLWFtbzAeFw0xNzA1MDQwMDEyMzlaFw0xOTA1MDQwMDEyMzla
MIGmMQswCQYDVQQGEwJVUzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEv
MC0GA1UECxMmTW96aWxsYSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2Ux
JTAjBgNVBAMTHENvbnRlbnQgU2lnbmluZyBJbnRlcm1lZGlhdGUxITAfBgkqhkiG
9w0BCQEWEmZveHNlY0Btb3ppbGxhLmNvbTB2MBAGByqGSM49AgEGBSuBBAAiA2IA
BMCmt4C33KfMzsyKokc9SXmMSxozksQglhoGAA1KjlgqEOzcmKEkxtvnGWOA9FLo
A6U7Wmy+7sqmvmjLboAPQc4G0CEudn5Nfk36uEqeyiyKwKSAT+pZsqS4/maXIC7s
DqOCAYkwggGFMAwGA1UdEwQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMBYGA1UdJQEB
/wQMMAoGCCsGAQUFBwMDMB0GA1UdDgQWBBSIdXKZW/BQ8mC4PbEJlh6PeXme7jCB
qAYDVR0jBIGgMIGdgBSzvOpYdKvhbngqsqucIx6oYyyXt6GBgaR/MH0xCzAJBgNV
BAYTAlVTMRwwGgYDVQQKExNNb3ppbGxhIENvcnBvcmF0aW9uMS8wLQYDVQQLEyZN
b3ppbGxhIEFNTyBQcm9kdWN0aW9uIFNpZ25pbmcgU2VydmljZTEfMB0GA1UEAxMW
cm9vdC1jYS1wcm9kdWN0aW9uLWFtb4IBATAzBglghkgBhvhCAQQEJhYkaHR0cDov
L2FkZG9ucy5hbGxpem9tLm9yZy9jYS9jcmwucGVtME4GA1UdHgRHMEWgQzAggh4u
Y29udGVudC1zaWduYXR1cmUubW96aWxsYS5vcmcwH4IdY29udGVudC1zaWduYXR1
cmUubW96aWxsYS5vcmcwDQYJKoZIhvcNAQEMBQADggIBAKWhLjJB8XmW3VfLvyLF
OOUNeNs7Aju+EZl1PMVXf+917LB//FcJKUQLcEo86I6nC3umUNl+kaq4d3yPDpMV
4DKLHgGmegRsvAyNFQfd64TTxzyfoyfNWH8uy5vvxPmLvWb+jXCoMNF5FgFWEVon
5GDEK8hoHN/DMVe0jveeJhUSuiUpJhMzEf6Vbo0oNgfaRAZKO+VOY617nkTOPnVF
LSEcUPIdE8pcd+QP1t/Ysx+mAfkxAbt+5K298s2bIRLTyNUj1eBtTcCbBbFyWsly
rSMkJihFAWU2MVKqvJ74YI3uNhFzqJ/AAUAPoet14q+ViYU+8a1lqEWj7y8foF3r
m0ZiQpuHULiYCO4y4NR7g5ijj6KsbruLv3e9NyUAIRBHOZEKOA7EiFmWJgqH1aZv
/eS7aQ9HMtPKrlbEwUjV0P3K2U2ljs0rNvO8KO9NKQmocXaRpLm+s8PYBGxby92j
5eelLq55028BSzhJJc6G+cRT9Hlxf1cg2qtqcVJa8i8wc2upCaGycZIlBSX4gj/4
k9faY4qGuGnuEdzAyvIXWMSkb8jiNHQfZrebSr00vShkUEKOLmfFHbkwIaWNK0+2
2c3RL4tDnM5u0kvdgWf0B742JskkxqqmEeZVofsOZJLOhXxO9NO/S0hM16/vf/tl
Tnsnhv0nxUR0B9wxN7XmWmq4
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIGYTCCBEmgAwIBAgIBATANBgkqhkiG9w0BAQwFADB9MQswCQYDVQQGEwJVUzEc
MBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0GA1UECxMmTW96aWxsYSBB
TU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAdBgNVBAMTFnJvb3QtY2Et
cHJvZHVjdGlvbi1hbW8wHhcNMTUwMzE3MjI1MzU3WhcNMjUwMzE0MjI1MzU3WjB9
MQswCQYDVQQGEwJVUzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0G
A1UECxMmTW96aWxsYSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAd
BgNVBAMTFnJvb3QtY2EtcHJvZHVjdGlvbi1hbW8wggIgMA0GCSqGSIb3DQEBAQUA
A4ICDQAwggIIAoICAQC0u2HXXbrwy36+MPeKf5jgoASMfMNz7mJWBecJgvlTf4hH
JbLzMPsIUauzI9GEpLfHdZ6wzSyFOb4AM+D1mxAWhuZJ3MDAJOf3B1Rs6QorHrl8
qqlNtPGqepnpNJcLo7JsSqqE3NUm72MgqIHRgTRsqUs+7LIPGe7262U+N/T0LPYV
Le4rZ2RDHoaZhYY7a9+49mHOI/g2YFB+9yZjE+XdplT2kBgA4P8db7i7I0tIi4b0
B0N6y9MhL+CRZJyxdFe2wBykJX14LsheKsM1azHjZO56SKNrW8VAJTLkpRxCmsiT
r08fnPyDKmaeZ0BtsugicdipcZpXriIGmsZbI12q5yuwjSELdkDV6Uajo2n+2ws5
uXrP342X71WiWhC/dF5dz1LKtjBdmUkxaQMOP/uhtXEKBrZo1ounDRQx1j7+SkQ4
BEwjB3SEtr7XDWGOcOIkoJZWPACfBLC3PJCBWjTAyBlud0C5n3Cy9regAAnOIqI1
t16GU2laRh7elJ7gPRNgQgwLXeZcFxw6wvyiEcmCjOEQ6PM8UQjthOsKlszMhlKw
vjyOGDoztkqSBy/v+Asx7OW2Q7rlVfKarL0mREZdSMfoy3zTgtMVCM0vhNl6zcvf
5HNNopoEdg5yuXo2chZ1p1J+q86b0G5yJRMeT2+iOVY2EQ37tHrqUURncCy4uwIB
A6OB7TCB6jAMBgNVHRMEBTADAQH/MA4GA1UdDwEB/wQEAwIBBjAWBgNVHSUBAf8E
DDAKBggrBgEFBQcDAzCBkgYDVR0jBIGKMIGHoYGBpH8wfTELMAkGA1UEBhMCVVMx
HDAaBgNVBAoTE01vemlsbGEgQ29ycG9yYXRpb24xLzAtBgNVBAsTJk1vemlsbGEg
QU1PIFByb2R1Y3Rpb24gU2lnbmluZyBTZXJ2aWNlMR8wHQYDVQQDExZyb290LWNh
LXByb2R1Y3Rpb24tYW1vggEBMB0GA1UdDgQWBBSzvOpYdKvhbngqsqucIx6oYyyX
tzANBgkqhkiG9w0BAQwFAAOCAgEAaNSRYAaECAePQFyfk12kl8UPLh8hBNidP2H6
KT6O0vCVBjxmMrwr8Aqz6NL+TgdPmGRPDDLPDpDJTdWzdj7khAjxqWYhutACTew5
eWEaAzyErbKQl+duKvtThhV2p6F6YHJ2vutu4KIciOMKB8dslIqIQr90IX2Usljq
8Ttdyf+GhUmazqLtoB0GOuESEqT4unX6X7vSGu1oLV20t7t5eCnMMYD67ZBn0YIU
/cm/+pan66hHrja+NeDGF8wabJxdqKItCS3p3GN1zUGuJKrLykxqbOp/21byAGog
Z1amhz6NHUcfE6jki7sM7LHjPostU5ZWs3PEfVVgha9fZUhOrIDsyXEpCWVa3481
LlAq3GiUMKZ5DVRh9/Nvm4NwrTfB3QkQQJCwfXvO9pwnPKtISYkZUqhEqvXk5nBg
QCkDSLDjXTx39naBBGIVIqBtKKuVTla9enngdq692xX/CgO6QJVrwpqdGjebj5P8
5fNZPABzTezG3Uls5Vp+4iIWVAEDkK23cUj3c/HhE+Oo7kxfUeu5Y1ZV3qr61+6t
ZARKjbu1TuYQHf0fs+GwID8zeLc2zJL7UzcHFwwQ6Nda9OJN4uPAuC/BKaIpxCLL
26b24/tRam4SJjqpiq20lynhUrmTtt6hbG3E1Hpy3bmkt2DYnuMFwEx2gfXNcnbT
wNuvFqc=
-----END CERTIFICATE-----`

// This chain is in the wrong order: the intermediate cert is
// placed first, followed by the EE then the root
var WronglyOrderedChain = `-----BEGIN CERTIFICATE-----
MIIFfjCCA2agAwIBAgIDEAAGMA0GCSqGSIb3DQEBDAUAMH0xCzAJBgNVBAYTAlVT
MRwwGgYDVQQKExNNb3ppbGxhIENvcnBvcmF0aW9uMS8wLQYDVQQLEyZNb3ppbGxh
IEFNTyBQcm9kdWN0aW9uIFNpZ25pbmcgU2VydmljZTEfMB0GA1UEAxMWcm9vdC1j
YS1wcm9kdWN0aW9uLWFtbzAeFw0xNzA1MDQwMDEyMzlaFw0xOTA1MDQwMDEyMzla
MIGmMQswCQYDVQQGEwJVUzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEv
MC0GA1UECxMmTW96aWxsYSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2Ux
JTAjBgNVBAMTHENvbnRlbnQgU2lnbmluZyBJbnRlcm1lZGlhdGUxITAfBgkqhkiG
9w0BCQEWEmZveHNlY0Btb3ppbGxhLmNvbTB2MBAGByqGSM49AgEGBSuBBAAiA2IA
BMCmt4C33KfMzsyKokc9SXmMSxozksQglhoGAA1KjlgqEOzcmKEkxtvnGWOA9FLo
A6U7Wmy+7sqmvmjLboAPQc4G0CEudn5Nfk36uEqeyiyKwKSAT+pZsqS4/maXIC7s
DqOCAYkwggGFMAwGA1UdEwQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMBYGA1UdJQEB
/wQMMAoGCCsGAQUFBwMDMB0GA1UdDgQWBBSIdXKZW/BQ8mC4PbEJlh6PeXme7jCB
qAYDVR0jBIGgMIGdgBSzvOpYdKvhbngqsqucIx6oYyyXt6GBgaR/MH0xCzAJBgNV
BAYTAlVTMRwwGgYDVQQKExNNb3ppbGxhIENvcnBvcmF0aW9uMS8wLQYDVQQLEyZN
b3ppbGxhIEFNTyBQcm9kdWN0aW9uIFNpZ25pbmcgU2VydmljZTEfMB0GA1UEAxMW
cm9vdC1jYS1wcm9kdWN0aW9uLWFtb4IBATAzBglghkgBhvhCAQQEJhYkaHR0cDov
L2FkZG9ucy5hbGxpem9tLm9yZy9jYS9jcmwucGVtME4GA1UdHgRHMEWgQzAggh4u
Y29udGVudC1zaWduYXR1cmUubW96aWxsYS5vcmcwH4IdY29udGVudC1zaWduYXR1
cmUubW96aWxsYS5vcmcwDQYJKoZIhvcNAQEMBQADggIBAKWhLjJB8XmW3VfLvyLF
OOUNeNs7Aju+EZl1PMVXf+917LB//FcJKUQLcEo86I6nC3umUNl+kaq4d3yPDpMV
4DKLHgGmegRsvAyNFQfd64TTxzyfoyfNWH8uy5vvxPmLvWb+jXCoMNF5FgFWEVon
5GDEK8hoHN/DMVe0jveeJhUSuiUpJhMzEf6Vbo0oNgfaRAZKO+VOY617nkTOPnVF
LSEcUPIdE8pcd+QP1t/Ysx+mAfkxAbt+5K298s2bIRLTyNUj1eBtTcCbBbFyWsly
rSMkJihFAWU2MVKqvJ74YI3uNhFzqJ/AAUAPoet14q+ViYU+8a1lqEWj7y8foF3r
m0ZiQpuHULiYCO4y4NR7g5ijj6KsbruLv3e9NyUAIRBHOZEKOA7EiFmWJgqH1aZv
/eS7aQ9HMtPKrlbEwUjV0P3K2U2ljs0rNvO8KO9NKQmocXaRpLm+s8PYBGxby92j
5eelLq55028BSzhJJc6G+cRT9Hlxf1cg2qtqcVJa8i8wc2upCaGycZIlBSX4gj/4
k9faY4qGuGnuEdzAyvIXWMSkb8jiNHQfZrebSr00vShkUEKOLmfFHbkwIaWNK0+2
2c3RL4tDnM5u0kvdgWf0B742JskkxqqmEeZVofsOZJLOhXxO9NO/S0hM16/vf/tl
Tnsnhv0nxUR0B9wxN7XmWmq4
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIEnTCCBCSgAwIBAgIEAQAAFzAKBggqhkjOPQQDAzCBpjELMAkGA1UEBhMCVVMx
HDAaBgNVBAoTE01vemlsbGEgQ29ycG9yYXRpb24xLzAtBgNVBAsTJk1vemlsbGEg
QU1PIFByb2R1Y3Rpb24gU2lnbmluZyBTZXJ2aWNlMSUwIwYDVQQDExxDb250ZW50
IFNpZ25pbmcgSW50ZXJtZWRpYXRlMSEwHwYJKoZIhvcNAQkBFhJmb3hzZWNAbW96
aWxsYS5jb20wHhcNMTcwNTA5MTQwMjM3WhcNMTcxMTA3MTQwMjM3WjCBrzELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExHDAaBgNVBAoTE01vemlsbGEg
Q29ycG9yYXRpb24xFzAVBgNVBAsTDkNsb3VkIFNlcnZpY2VzMS8wLQYDVQQDEyZu
b3JtYW5keS5jb250ZW50LXNpZ25hdHVyZS5tb3ppbGxhLm9yZzEjMCEGCSqGSIb3
DQEJARYUc2VjdXJpdHlAbW96aWxsYS5vcmcwdjAQBgcqhkjOPQIBBgUrgQQAIgNi
AAShRFsGyg6DkUX+J2mMDM6cLK8V6HawjGVlQ/w5H5fHiGJDMrkl4ktnN+O37mSs
dReHcVxxpPNEpIfkWQ2TFmJgOUzqi/CzO06APlAJ9mnIcaobgdqRQxoTchFEyzUx
nTijggIWMIICEjAdBgNVHQ4EFgQUKnGLJ9po8ea5qUNjJyV/c26VZfswgaoGA1Ud
IwSBojCBn4AUiHVymVvwUPJguD2xCZYej3l5nu6hgYGkfzB9MQswCQYDVQQGEwJV
UzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0GA1UECxMmTW96aWxs
YSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAdBgNVBAMTFnJvb3Qt
Y2EtcHJvZHVjdGlvbi1hbW+CAxAABjAMBgNVHRMBAf8EAjAAMA4GA1UdDwEB/wQE
AwIHgDAWBgNVHSUBAf8EDDAKBggrBgEFBQcDAzBFBgNVHR8EPjA8MDqgOKA2hjRo
dHRwczovL2NvbnRlbnQtc2lnbmF0dXJlLmNkbi5tb3ppbGxhLm5ldC9jYS9jcmwu
cGVtMEMGCWCGSAGG+EIBBAQ2FjRodHRwczovL2NvbnRlbnQtc2lnbmF0dXJlLmNk
bi5tb3ppbGxhLm5ldC9jYS9jcmwucGVtME8GCCsGAQUFBwEBBEMwQTA/BggrBgEF
BQcwAoYzaHR0cHM6Ly9jb250ZW50LXNpZ25hdHVyZS5jZG4ubW96aWxsYS5uZXQv
Y2EvY2EucGVtMDEGA1UdEQQqMCiCJm5vcm1hbmR5LmNvbnRlbnQtc2lnbmF0dXJl
Lm1vemlsbGEub3JnMAoGCCqGSM49BAMDA2cAMGQCMGeeyXYM3+r1fcaXzd90PwGb
h9nrl1fZNXrCu17lCPn2JntBVh7byT3twEbr+Hmv8gIwU9klAW6yHLG/ZpAZ0jdf
38Rciz/FDEAdrzH2QlYAOw+uDdpcmon9oiRgIxzwNlUe
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIGYTCCBEmgAwIBAgIBATANBgkqhkiG9w0BAQwFADB9MQswCQYDVQQGEwJVUzEc
MBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0GA1UECxMmTW96aWxsYSBB
TU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAdBgNVBAMTFnJvb3QtY2Et
cHJvZHVjdGlvbi1hbW8wHhcNMTUwMzE3MjI1MzU3WhcNMjUwMzE0MjI1MzU3WjB9
MQswCQYDVQQGEwJVUzEcMBoGA1UEChMTTW96aWxsYSBDb3Jwb3JhdGlvbjEvMC0G
A1UECxMmTW96aWxsYSBBTU8gUHJvZHVjdGlvbiBTaWduaW5nIFNlcnZpY2UxHzAd
BgNVBAMTFnJvb3QtY2EtcHJvZHVjdGlvbi1hbW8wggIgMA0GCSqGSIb3DQEBAQUA
A4ICDQAwggIIAoICAQC0u2HXXbrwy36+MPeKf5jgoASMfMNz7mJWBecJgvlTf4hH
JbLzMPsIUauzI9GEpLfHdZ6wzSyFOb4AM+D1mxAWhuZJ3MDAJOf3B1Rs6QorHrl8
qqlNtPGqepnpNJcLo7JsSqqE3NUm72MgqIHRgTRsqUs+7LIPGe7262U+N/T0LPYV
Le4rZ2RDHoaZhYY7a9+49mHOI/g2YFB+9yZjE+XdplT2kBgA4P8db7i7I0tIi4b0
B0N6y9MhL+CRZJyxdFe2wBykJX14LsheKsM1azHjZO56SKNrW8VAJTLkpRxCmsiT
r08fnPyDKmaeZ0BtsugicdipcZpXriIGmsZbI12q5yuwjSELdkDV6Uajo2n+2ws5
uXrP342X71WiWhC/dF5dz1LKtjBdmUkxaQMOP/uhtXEKBrZo1ounDRQx1j7+SkQ4
BEwjB3SEtr7XDWGOcOIkoJZWPACfBLC3PJCBWjTAyBlud0C5n3Cy9regAAnOIqI1
t16GU2laRh7elJ7gPRNgQgwLXeZcFxw6wvyiEcmCjOEQ6PM8UQjthOsKlszMhlKw
vjyOGDoztkqSBy/v+Asx7OW2Q7rlVfKarL0mREZdSMfoy3zTgtMVCM0vhNl6zcvf
5HNNopoEdg5yuXo2chZ1p1J+q86b0G5yJRMeT2+iOVY2EQ37tHrqUURncCy4uwIB
A6OB7TCB6jAMBgNVHRMEBTADAQH/MA4GA1UdDwEB/wQEAwIBBjAWBgNVHSUBAf8E
DDAKBggrBgEFBQcDAzCBkgYDVR0jBIGKMIGHoYGBpH8wfTELMAkGA1UEBhMCVVMx
HDAaBgNVBAoTE01vemlsbGEgQ29ycG9yYXRpb24xLzAtBgNVBAsTJk1vemlsbGEg
QU1PIFByb2R1Y3Rpb24gU2lnbmluZyBTZXJ2aWNlMR8wHQYDVQQDExZyb290LWNh
LXByb2R1Y3Rpb24tYW1vggEBMB0GA1UdDgQWBBSzvOpYdKvhbngqsqucIx6oYyyX
tzANBgkqhkiG9w0BAQwFAAOCAgEAaNSRYAaECAePQFyfk12kl8UPLh8hBNidP2H6
KT6O0vCVBjxmMrwr8Aqz6NL+TgdPmGRPDDLPDpDJTdWzdj7khAjxqWYhutACTew5
eWEaAzyErbKQl+duKvtThhV2p6F6YHJ2vutu4KIciOMKB8dslIqIQr90IX2Usljq
8Ttdyf+GhUmazqLtoB0GOuESEqT4unX6X7vSGu1oLV20t7t5eCnMMYD67ZBn0YIU
/cm/+pan66hHrja+NeDGF8wabJxdqKItCS3p3GN1zUGuJKrLykxqbOp/21byAGog
Z1amhz6NHUcfE6jki7sM7LHjPostU5ZWs3PEfVVgha9fZUhOrIDsyXEpCWVa3481
LlAq3GiUMKZ5DVRh9/Nvm4NwrTfB3QkQQJCwfXvO9pwnPKtISYkZUqhEqvXk5nBg
QCkDSLDjXTx39naBBGIVIqBtKKuVTla9enngdq692xX/CgO6QJVrwpqdGjebj5P8
5fNZPABzTezG3Uls5Vp+4iIWVAEDkK23cUj3c/HhE+Oo7kxfUeu5Y1ZV3qr61+6t
ZARKjbu1TuYQHf0fs+GwID8zeLc2zJL7UzcHFwwQ6Nda9OJN4uPAuC/BKaIpxCLL
26b24/tRam4SJjqpiq20lynhUrmTtt6hbG3E1Hpy3bmkt2DYnuMFwEx2gfXNcnbT
wNuvFqc=
-----END CERTIFICATE-----`
