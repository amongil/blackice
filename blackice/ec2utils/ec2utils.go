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

package ec2utils

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Client is our new type. Uses embedding to create our own methods on the client.
type Client struct {
	*ec2.EC2
}

// NewClient creates our own EC2 client wrapped around the SDK EC2
func NewClient(region string) *Client {
	// Initialize a session that the SDK will use to load configuration,
	// credentials, and region from the shared config file. (~/.aws/config).
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ec2.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	cl := &Client{svc}
	return cl
}

// GetInstancesByKeyPair returns a list of string containing instance ids that match
func (cl *Client) GetInstancesByKeyPair(k string) ([]*ec2.Instance, error) {
	var keyNames []*string
	keyName := k
	keyNames = append(keyNames, &keyName)

	var filters []*ec2.Filter
	keyNameFilter := "key-name"
	keyPairFilter := &ec2.Filter{
		Name:   &keyNameFilter,
		Values: keyNames,
	}
	filters = append(filters, keyPairFilter)

	var result []*ec2.Instance
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}
	err := cl.DescribeInstancesPages(
		input,
		func(p *ec2.DescribeInstancesOutput, lastPage bool) (shouldContinue bool) {
			for _, reservation := range p.Reservations {
				for _, instance := range reservation.Instances {
					result = append(result, instance)
				}
			}
			if p.NextToken == nil {
				return false
			}
			return true
		})
	if err != nil {
		aerr, _ := err.(awserr.Error)
		return nil, aerr
	}
	return result, err
}

// GetKeyPairs returns all the Key Pairs registered on the account
func (cl *Client) GetKeyPairs() ([]*ec2.KeyPairInfo, error) {
	result, err := cl.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
	if err != nil {
		aerr, _ := err.(awserr.Error)
		return nil, aerr
	}

	return result.KeyPairs, err
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

// MarshalPKCS8PrivateKey converts a private key to PKCS#8 encoded form.
// See http://www.rsa.com/rsalabs/node.asp?id=2130 and RFC5208.
func MarshalPKCS8PrivateKey(key interface{}) ([]byte, error) {
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
