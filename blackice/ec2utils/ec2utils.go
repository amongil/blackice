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
