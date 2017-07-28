package main

import "github.com/aws/aws-sdk-go/service/cloudwatchlogs"

// CloudWatchLog is a CloudWatch Logs log group.
type CloudWatchLog struct {
	svc          *cloudwatchlogs.CloudWatchLogs
	logGroupName string
}

// NewCloudWatchLog returns a CloudWatchLog for the given log group name.
func NewCloudWatchLog(svc *cloudwatchlogs.CloudWatchLogs, logGroupName string) *CloudWatchLog {
	return &CloudWatchLog{
		svc:          svc,
		logGroupName: logGroupName,
	}
}

func (cwl *CloudWatchLog) GetLogEvents(start, end int64) ([]*cloudwatchlogs.FilteredLogEvent, error) {
	result := []*cloudwatchlogs.FilteredLogEvent{}
	limit := int64(10000)
	var token *string
	interleaved := true
	for {
		output, err := cwl.svc.FilterLogEvents(&cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: &cwl.logGroupName,
			Limit:        &limit,
			NextToken:    token,
			StartTime:    &start,
			EndTime:      &end,
			Interleaved:  &interleaved,
		})
		if err != nil {
			return nil, err
		}

		result = append(result, output.Events...)

		if output.NextToken == nil {
			break
		}

		token = output.NextToken
	}

	return result, nil
}
