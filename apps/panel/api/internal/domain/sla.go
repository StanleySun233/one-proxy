package domain

type NodeSLAMinute struct {
	ScenarioID         string `json:"scenarioId"`
	ScenarioName       string `json:"scenarioName"`
	NodeID             string `json:"nodeId"`
	NodeName           string `json:"nodeName"`
	WindowStart        string `json:"windowStart"`
	ExpectedHeartbeats int    `json:"expectedHeartbeats"`
	ReceivedHeartbeats int    `json:"receivedHeartbeats"`
	Success            int    `json:"success"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

type NodeSLAMinuteInput struct {
	NodeID             string
	WindowStart        string
	ExpectedHeartbeats int
	ReceivedHeartbeats int
	Success            int
}
