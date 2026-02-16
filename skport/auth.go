package skport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Authenticate performs the 3-step OAuth flow to obtain credentials.
func Authenticate(accountToken string) (*Credentials, error) {
	if accountToken == "" {
		return nil, fmt.Errorf("no account token supplied for OAuth flow")
	}

	// Step 1: basic info (validate token)
	infoURL := BasicInfoURL + "?token=" + url.QueryEscape(accountToken)
	resp, err := http.Get(infoURL)
	if err != nil {
		return nil, fmt.Errorf("OAuth step 1 request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var infoResp basicInfoResponse
	if err := json.Unmarshal(body, &infoResp); err != nil {
		return nil, fmt.Errorf("OAuth step 1 parse failed: %w", err)
	}
	if infoResp.Status != 0 {
		return nil, fmt.Errorf("OAuth step 1 failed: %s", firstNonEmpty(infoResp.Msg, string(body)))
	}

	// Step 2: grant OAuth code
	grantBody, _ := json.Marshal(oauthGrantRequest{
		Token:   accountToken,
		AppCode: AppCode,
		Type:    0,
	})
	resp2, err := http.Post(OAuthGrantURL, "application/json", bytes.NewReader(grantBody))
	if err != nil {
		return nil, fmt.Errorf("OAuth step 2 request failed: %w", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)

	var grantResp oauthGrantResponse
	if err := json.Unmarshal(body2, &grantResp); err != nil {
		return nil, fmt.Errorf("OAuth step 2 parse failed: %w", err)
	}
	if grantResp.Status != 0 || grantResp.Data == nil || grantResp.Data.Code == "" {
		return nil, fmt.Errorf("OAuth step 2 failed: %s", firstNonEmpty(grantResp.Msg, string(body2)))
	}

	// Step 3: exchange code for cred
	credBody, _ := json.Marshal(generateCredRequest{
		Code: grantResp.Data.Code,
		Kind: 1,
	})
	req, _ := http.NewRequest("POST", GenerateCredURL, bytes.NewReader(credBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Platform", Platform)
	req.Header.Set("Referer", "https://www.skport.com/")
	req.Header.Set("Origin", "https://www.skport.com")

	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OAuth step 3 request failed: %w", err)
	}
	defer resp3.Body.Close()
	body3, _ := io.ReadAll(resp3.Body)

	var credResp generateCredResponse
	if err := json.Unmarshal(body3, &credResp); err != nil {
		return nil, fmt.Errorf("OAuth step 3 parse failed: %w", err)
	}
	if credResp.Code != 0 || credResp.Data == nil || credResp.Data.Cred == "" {
		return nil, fmt.Errorf("OAuth step 3 failed: %s", firstNonEmpty(credResp.Message, string(body3)))
	}

	return &Credentials{
		Cred:   credResp.Data.Cred,
		Salt:   credResp.Data.Token,
		UserID: credResp.Data.UserID,
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "unknown error"
}
