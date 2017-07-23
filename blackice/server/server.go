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
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/amongil/blackice/blackice/ec2utils"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/julienschmidt/httprouter"
)

type instanceStruct struct {
	Name             string
	PrivateIPAddress string
}
type scanResponse struct {
	AllowedInstances *[]instanceStruct
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func fingerprint(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := r.FormValue("identity")
	fp, err := GetFingerprint([]byte(key))
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}
	fmt.Fprintf(w, "%s\n", fp)
}

func instances(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := r.FormValue("keyname")
	cl := ec2utils.NewClient("eu-central-1")
	instances, err := cl.GetInstancesByKeyPair(key)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Convert our structures to json
	res, err := json.MarshalIndent(instances, "", "\t")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, string(res))
}

func keypairs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cl := ec2utils.NewClient("eu-central-1")
	keyPairs, err := cl.GetKeyPairs()
	if err != nil {
		fmt.Println(err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	res, err := json.MarshalIndent(keyPairs, "", "\t")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, string(res))
}

func scan(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	identity := r.FormValue("identity")
	cl := ec2utils.NewClient("eu-central-1")
	keyPairs, err := cl.GetKeyPairs()
	if err != nil {
		fmt.Println(err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	keyName, err := findKeyNameByFingerprint(keyPairs, identity)
	if err != nil {
		fmt.Println(err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}
	instances, err := cl.GetInstancesByKeyPair(keyName)
	if err != nil {
		fmt.Println(err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}
	scanResponse := new(scanResponse)
	var allowedInstances []instanceStruct
	for _, instance := range instances {
		allowedInstances = append(allowedInstances,
			instanceStruct{
				Name:             *instance.InstanceId,
				PrivateIPAddress: *instance.PrivateIpAddress,
			})
	}
	scanResponse.AllowedInstances = &allowedInstances
	// Convert our structures to json
	res, err := json.MarshalIndent(scanResponse, "", "\t")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, string(res))
}

// New creates a new http server that can be started and stopped
func New() *http.Server {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/hello/:name", hello)
	router.POST("/keypairs", keypairs)
	router.POST("/instances", instances)
	router.POST("/scan", scan)
	router.POST("/fingerprint", fingerprint)
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

// GetFingerprint returns fingerprint of ssh-key
func GetFingerprint(pemFile []byte) (string, error) {
	block, _ := pem.Decode(pemFile)

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	keyPKCS8, err := ec2utils.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", err
	}

	sha := fmt.Sprintf("% x", sha1.Sum(keyPKCS8))
	sha = strings.Replace(sha, " ", ":", -1)
	return sha, nil
}

func findKeyNameByFingerprint(keyPairs []*ec2.KeyPairInfo, fingerprint string) (string, error) {
	for _, keyPair := range keyPairs {
		if *keyPair.KeyFingerprint == fingerprint {
			return *keyPair.KeyName, nil
		}
	}
	return "", errors.New("IdentityNotFoundError")
}
