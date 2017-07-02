package main

import (
	"fmt"

	"github.com/amongil/blackice/ec2utils"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	cl := ec2utils.NewClient()

	input := &ec2.DescribeInstancesInput{}
	err := cl.DescribeInstancesPages(
		input,
		func(p *ec2.DescribeInstancesOutput, lastPage bool) (shouldContinue bool) {
			fmt.Println(p)
			if p.NextToken == nil {
				return false
			}
			return true
		})
	if err != nil {
		aerr, _ := err.(awserr.Error)
		fmt.Println(aerr)
		return
	}
}
