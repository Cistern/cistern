package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type LogState struct {
	LastTimestamp int64 `json:"last_timestamp"`
	filename      string
}

func (s *LogState) Store() error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.filename, data, 0600)
	return err
}

func (s *LogState) Load() error {
	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, s)
	return err
}

func NewLogState(filename string) (*LogState, error) {
	state := &LogState{
		filename: filename,
	}
	state.Load()
	return state, state.Store()
}

// CloudWatchLog is a CloudWatch Logs log group.
type CloudWatchLog struct {
	svc          *cloudwatchlogs.CloudWatchLogs
	logGroupName string
	logState     *LogState
}

// NewCloudWatchLog returns a CloudWatchLog for the given log group name.
func NewCloudWatchLog(svc *cloudwatchlogs.CloudWatchLogs, logGroupName string) (*CloudWatchLog, error) {
	logState, err := NewLogState(filepath.Join(DataDir, logGroupName+".state"))
	if err != nil {
		return nil, err
	}
	return &CloudWatchLog{
		svc:          svc,
		logGroupName: logGroupName,
		logState:     logState,
	}, nil
}

// GetLogEvents gets log events from the log group.
func (cwl *CloudWatchLog) GetLogEvents(start int64) ([]*cloudwatchlogs.FilteredLogEvent, error) {
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

func (cwl *CloudWatchLog) LastTimestamp() int64 {
	return cwl.logState.LastTimestamp
}

func (cwl *CloudWatchLog) SetLastTimestamp(t int64) error {
	cwl.logState.LastTimestamp = t
	return cwl.logState.Store()
}
