package main

type Event map[string]interface{}

type QueryResult struct {
	Summary []Event     `json:"summary,omitempty"`
	Series  []Event     `json:"series,omitempty"`
	Events  []Event     `json:"events,omitempty"`
	Query   interface{} `json:"query"`
}
