package skport

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	Platform  = "3"
	VName     = "1.0.0"
	AppCode   = "6eb76d4e13aa36e6"
	GameID    = "3"
	UserAgent = "Skport/0.7.0 (com.gryphline.skport; build:700089; Android 33; ) Okhttp/5.1.0"

	BindingURL     = "https://zonai.skport.com/api/v1/game/player/binding"
	AttendanceURL  = "https://zonai.skport.com/web/v1/game/endfield/attendance"
	GenerateCredURL = "https://zonai.skport.com/web/v1/user/auth/generate_cred_by_code"
	OAuthGrantURL  = "https://as.gryphline.com/user/oauth2/v2/grant"
	BasicInfoURL   = "https://as.gryphline.com/user/info/v1/basic"
)

// Timestamp returns the current Unix timestamp as a string.
func Timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// BuildHeaders constructs the standard headers for SKPort API requests.
func BuildHeaders(cred, gameRole, timestamp string) http.Header {
	h := http.Header{}
	h.Set("Accept", "application/json, text/plain, */*")
	h.Set("Content-Type", "application/json")
	h.Set("Origin", "https://game.skport.com")
	h.Set("Referer", "https://game.skport.com/")
	h.Set("Cred", cred)
	h.Set("Platform", Platform)
	h.Set("Sk-Language", "en")
	h.Set("Timestamp", timestamp)
	h.Set("Vname", VName)
	h.Set("User-Agent", UserAgent)
	if gameRole != "" {
		h.Set("Sk-Game-Role", gameRole)
	}
	return h
}

// ComputeSignV2 computes the V2 signature: MD5(HMAC-SHA256(path+ts+headerJSON, salt)).
func ComputeSignV2(path, timestamp, salt string) string {
	headerObj := struct {
		Platform  string `json:"platform"`
		Timestamp string `json:"timestamp"`
		DID       string `json:"dId"`
		VName     string `json:"vName"`
	}{
		Platform:  Platform,
		Timestamp: timestamp,
		DID:       "",
		VName:     VName,
	}
	headerJSON, _ := json.Marshal(headerObj)
	s := path + timestamp + string(headerJSON)

	mac := hmac.New(sha256.New, []byte(salt))
	mac.Write([]byte(s))
	hmacHex := fmt.Sprintf("%x", mac.Sum(nil))

	md5sum := md5.Sum([]byte(hmacHex))
	return fmt.Sprintf("%x", md5sum)
}
