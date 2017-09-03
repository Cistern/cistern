package main

type ConfigCloudWatchLogGroup struct {
	Name    string `json:"name"`
	FlowLog bool   `json:"flowlog"`
}

type Config struct {
	CloudWatchLogs []ConfigCloudWatchLogGroup `json:"cloudwatch_logs"`
	Retention      int                        `json:"retention"`
}
