package domain

type ClientDirectSessionValidateInput struct {
	SessionID  string `json:"sessionId"`
	PunchToken string `json:"punchToken"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}

type ClientDirectSessionValidateResult struct {
	Valid      bool   `json:"valid"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}
