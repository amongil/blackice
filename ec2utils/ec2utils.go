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
