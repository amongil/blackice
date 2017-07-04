// Copyright © 2017 Alvaro Mongil
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package server

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/amongil/blackice/blackice/ec2utils"
	"github.com/julienschmidt/httprouter"
)

var key = []byte(`
-----BEGIN RSA PRIVATE KEY-----
-----END RSA PRIVATE KEY-----`)

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func fingerprint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fp, err := GetFingerprint(key)
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	fmt.Fprintf(w, "%s\n", fp)
}

func instances(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cl := ec2utils.NewClient("eu-central-1")
	instances, err := cl.GetInstancesByKeyPair(ps.ByName("keyname"))
	if err != nil {
		fmt.Println(err.Error())
	}
	// Convert our structures to json
	res, err := json.MarshalIndent(instances, "", "\t")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, string(res))
}

// New creates a new http server that can be started and stopped
func New() *http.Server {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/hello/:name", hello)
	router.GET("/instances/:keyname", instances)
	router.GET("/fingerprint", fingerprint)
	addr := "127.0.0.1:8080"
	srv := &http.Server{
		Handler: router,
		Addr:    addr,
	}
	return srv
}

// Start makes the server start listening for requests
func Start(srv *http.Server) {
	log.Fatal(srv.ListenAndServe())
}

// Stop makes the server stop listening for requests
func Stop(srv *http.Server) {
	srv.Shutdown(nil)
}

// GetFingerprint return finerprint of ssh-key
func GetFingerprint(pemFile []byte) (string, error) {
	block, _ := pem.Decode(pemFile)

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	keyPKCS8, err := marshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}

	sha := fmt.Sprintf("% x", sha1.Sum(keyPKCS8))
	sha = strings.Replace(sha, " ", ":", -1)
	return sha, nil
}

// pkcs8 reflects an ASN.1, PKCS#8 PrivateKey. See
// ftp://ftp.rsasecurity.com/pub/pkcs/pkcs-8/pkcs-8v1_2.asn
// and RFC5208.
type pkcs8 struct {
	Version    int
	Algo       pkix.AlgorithmIdentifier
	PrivateKey []byte
	// optional attributes omitted.
}

var (
	oidPublicKeyRSA   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidPublicKeyECDSA = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}

	nullAsn = asn1.RawValue{Tag: 5}
)

// marshalPKCS8PrivateKey converts a private key to PKCS#8 encoded form.
// See http://www.rsa.com/rsalabs/node.asp?id=2130 and RFC5208.
func marshalPKCS8PrivateKey(key interface{}) ([]byte, error) {
	pkcs := pkcs8{
		Version: 0,
	}

	switch key := key.(type) {
	case *rsa.PrivateKey:
		pkcs.Algo = pkix.AlgorithmIdentifier{
			Algorithm:  oidPublicKeyRSA,
			Parameters: nullAsn,
		}
		pkcs.PrivateKey = x509.MarshalPKCS1PrivateKey(key)
	case *ecdsa.PrivateKey:
		bytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, errors.New("x509: failed to marshal to PKCS#8: " + err.Error())
		}

		pkcs.Algo = pkix.AlgorithmIdentifier{
			Algorithm:  oidPublicKeyECDSA,
			Parameters: nullAsn,
		}
		pkcs.PrivateKey = bytes
	default:
		return nil, errors.New("x509: PKCS#8 only RSA and ECDSA private keys supported")
	}

	bytes, err := asn1.Marshal(pkcs)
	if err != nil {
		return nil, errors.New("x509: failed to marshal to PKCS#8: " + err.Error())
	}

	return bytes, nil
}
