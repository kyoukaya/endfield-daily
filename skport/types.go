package skport

import "encoding/json"

// Credentials holds the auth credentials from the OAuth flow.
type Credentials struct {
	Cred   string
	Salt   string
	UserID string
}

// Role represents a player's game role binding.
type Role struct {
	GameRole string
	Nickname string
	Level    int
	Server   string
	ServerID string
	RoleID   string
}

// basicInfoResponse is the response from the basic info endpoint.
type basicInfoResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
}

// oauthGrantRequest is the request body for the OAuth grant endpoint.
type oauthGrantRequest struct {
	Token   string `json:"token"`
	AppCode string `json:"appCode"`
	Type    int    `json:"type"`
}

// oauthGrantResponse is the response from the OAuth grant endpoint.
type oauthGrantResponse struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   *struct {
		Code string `json:"code"`
	} `json:"data"`
}

// generateCredRequest is the request body for the generate cred endpoint.
type generateCredRequest struct {
	Code string `json:"code"`
	Kind int    `json:"kind"`
}

// generateCredResponse is the response from the generate cred endpoint.
type generateCredResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		Cred   string `json:"cred"`
		Token  string `json:"token"`
		UserID string `json:"userId"`
	} `json:"data"`
}

// bindingResponse is the response from the player binding endpoint.
type bindingResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		List []bindingApp `json:"list"`
	} `json:"data"`
}

type bindingApp struct {
	AppCode     string          `json:"appCode"`
	BindingList []bindingEntry  `json:"bindingList"`
}

type bindingEntry struct {
	Roles []bindingRole `json:"roles"`
}

type bindingRole struct {
	Nickname   string `json:"nickname"`
	Level      int    `json:"level"`
	ServerName string `json:"serverName"`
	ServerID   string `json:"serverId"`
	RoleID     string `json:"roleId"`
}

// attendanceStatusResponse is the response from the attendance GET endpoint.
type attendanceStatusResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		HasToday bool              `json:"hasToday"`
		Records  []json.RawMessage `json:"records"`
	} `json:"data"`
}

// attendanceClaimResponse is the response from the attendance POST endpoint.
type attendanceClaimResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *struct {
		AwardIds []struct {
			ID string `json:"id"`
		} `json:"awardIds"`
		ResourceInfoMap map[string]struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"resourceInfoMap"`
	} `json:"data"`
}
