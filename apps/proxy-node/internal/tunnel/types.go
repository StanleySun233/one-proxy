package tunnel

type Message struct {
	Type                string   `json:"type"`
	RequestID           string   `json:"requestId,omitempty"`
	StreamID            string   `json:"streamId,omitempty"`
	NodeID              string   `json:"nodeId,omitempty"`
	ParentID            string   `json:"parentId,omitempty"`
	Timestamp           string   `json:"timestamp,omitempty"`
	Status              string   `json:"status,omitempty"`
	Message             string   `json:"message,omitempty"`
	Data                string   `json:"data,omitempty"`
	RemainingHopNodeIDs []string `json:"remainingHopNodeIds,omitempty"`
	TargetHost          string   `json:"targetHost,omitempty"`
	TargetPort          int      `json:"targetPort,omitempty"`
}
