package ec2utils

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Client is our new type. Uses embedding to create our own methods on the client.
type Client struct {
	*ec2.EC2
}

// NewClient creates our own EC2 client wrapped around the SDK EC2
func NewClient() *Client {
	// Initialize a session that the SDK will use to load configuration,
	// credentials, and region from the shared config file. (~/.aws/config).
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ec2.New(sess)

	cl := &Client{svc}
	return cl
}
