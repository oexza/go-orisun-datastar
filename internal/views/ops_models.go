package views

import "fmt"

type OpsDashboardSnapshot struct {
	Handlers []OpsEventHandlerMetric `json:"handlers"`
}

type OpsEventHandlerMetric struct {
	Name                 string `json:"name"`
	Running              bool   `json:"running"`
	LastEventID          string `json:"lastEventId"`
	LastEventType        string `json:"lastEventType"`
	LastCommitPosition   int64  `json:"lastCommitPosition"`
	LastCheckpointCommit int64  `json:"lastCheckpointCommit"`
	ProcessedCount       int64  `json:"processedCount"`
	FailureCount         int64  `json:"failureCount"`
	LastError            string `json:"lastError"`
	UpdatedAt            string `json:"updatedAt"`
}

func int64Label(count int64) string {
	return fmt.Sprint(count)
}
