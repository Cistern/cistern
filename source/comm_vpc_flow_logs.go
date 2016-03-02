package source

import (
	"fmt"

	"github.com/Cistern/cistern/clock"
	"github.com/Cistern/cistern/config"
	"github.com/Cistern/cistern/message"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const CommVPCFlowLogsClassName = "vpcflowlog"

type CommVPCFlowLogsClass struct {
	outbound chan *message.Message
}

func NewCommVPCFlowLogsClass(
	conf config.AWSFlowLogConfig,
	outbound chan *message.Message) *CommVPCFlowLogsClass {
	c := &CommVPCFlowLogsClass{
		outbound: outbound,
	}

	svc := cloudwatchlogs.New(session.New(&aws.Config{
		Region:      aws.String(conf.Region),
		Credentials: credentials.NewStaticCredentials(conf.AccessKeyID, conf.SecretAccessKey, ""),
	}))

	params := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(conf.LogGroupName),  // Required
		LogStreamName: aws.String(conf.LogStreamName), // Required
		EndTime:       aws.Int64(clock.Time() * 1000),
		Limit:         aws.Int64(10),
		StartFromHead: aws.Bool(true),
		StartTime:     aws.Int64((clock.Time() - 86400) * 1000),
	}
	resp, err := svc.GetLogEvents(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
	}

	// Pretty-print the response data.
	fmt.Println(resp)

	return c
}

func (c *CommVPCFlowLogsClass) Name() string {
	return CommVPCFlowLogsClassName
}

func (c *CommVPCFlowLogsClass) Category() string {
	return "comm"
}

func (c *CommVPCFlowLogsClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *CommVPCFlowLogsClass) generateMessages() {

}
