package skport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kyoukaya/endfield-daily/notify"
)

// RunAccount authenticates, fetches roles, and checks in for one account.
func RunAccount(token string, index int, notifier notify.Notifier, notifyNoOps bool) error {
	creds, err := Authenticate(token)
	if err != nil {
		msg := fmt.Sprintf("Account %d: %s", index, err)
		fmt.Println(msg)
		return err
	}
	fmt.Printf("Account %d: obtained cred and salt\n", index)

	roles, err := getPlayerRoles(creds.Cred, creds.Salt)
	if err != nil {
		msg := fmt.Sprintf("Account %d: %s", index, err)
		fmt.Println(msg)
		return err
	}
	fmt.Printf("Account %d: Found %d role(s)\n", index, len(roles))

	for i, role := range roles {
		label := fmt.Sprintf("%s (Lv.%d) [%s]", role.Nickname, role.Level, role.Server)
		checkInRole(creds.Cred, creds.Salt, role, label, index, notifier, notifyNoOps)
		if i < len(roles)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}

// doWithRetry executes an HTTP request created by makeReq, retrying up to
// maxRetries times on network errors or 5xx responses using exponential backoff.
// The makeReq factory is called on every attempt so that timestamps/signatures
// embedded in headers are always fresh.
func doWithRetry(makeReq func() *http.Request) ([]byte, error) {
	const maxRetries = 3
	baseDelay := time.Second

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * (1 << (attempt - 1))
			fmt.Printf("  → Retry %d/%d after %s\n", attempt, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err := http.DefaultClient.Do(makeReq())
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error HTTP %d: %s", resp.StatusCode, body)
			continue
		}
		return body, nil
	}
	return nil, lastErr
}

func getPlayerRoles(cred, salt string) ([]Role, error) {
	body, err := doWithRetry(func() *http.Request {
		path := "/api/v1/game/player/binding"
		ts := Timestamp()
		headers := BuildHeaders(cred, "", ts)
		headers.Set("Sign", ComputeSignV2(path, ts, salt))
		req, _ := http.NewRequest("GET", BindingURL, nil)
		req.Header = headers
		return req
	})
	if err != nil {
		return nil, fmt.Errorf("binding request failed: %w", err)
	}

	var result bindingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("binding parse failed: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("binding API error: %s", firstNonEmpty(result.Message, string(body)))
	}

	if result.Data == nil {
		return nil, fmt.Errorf("no Endfield account binding found")
	}

	var endfieldApp *bindingApp
	for i := range result.Data.List {
		if result.Data.List[i].AppCode == "endfield" {
			endfieldApp = &result.Data.List[i]
			break
		}
	}
	if endfieldApp == nil || len(endfieldApp.BindingList) == 0 {
		return nil, fmt.Errorf("no Endfield account binding found")
	}

	var roles []Role
	for _, binding := range endfieldApp.BindingList {
		for _, r := range binding.Roles {
			roles = append(roles, Role{
				GameRole: GameID + "_" + r.RoleID + "_" + r.ServerID,
				Nickname: r.Nickname,
				Level:    r.Level,
				Server:   r.ServerName,
				ServerID: r.ServerID,
				RoleID:   r.RoleID,
			})
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("no roles found in binding")
	}
	return roles, nil
}

func checkInRole(cred, salt string, role Role, label string, accountIndex int, notifier notify.Notifier, notifyNoOps bool) {
	log := &notify.MessageLog{}
	log.Info(fmt.Sprintf("Account %d - %s", accountIndex, label))

	// Check attendance status
	body, err := doWithRetry(func() *http.Request {
		path := "/web/v1/game/endfield/attendance"
		ts := Timestamp()
		h := BuildHeaders(cred, role.GameRole, ts)
		h.Set("Sign", ComputeSignV2(path, ts, salt))
		req, _ := http.NewRequest("GET", AttendanceURL, nil)
		req.Header = h
		return req
	})
	if err != nil {
		msg := fmt.Sprintf("  → Attendance check request failed: %s", err)
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}

	var status attendanceStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		msg := fmt.Sprintf("  → Attendance check parse failed: %s", err)
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}
	if status.Code != 0 {
		msg := fmt.Sprintf("  → Attendance status check failed: %s", firstNonEmpty(status.Message, string(body)))
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}

	if status.Data != nil && status.Data.HasToday {
		log.Info("  → Already checked in today")
		// Only notify for no-ops if explicitly enabled
		if notifyNoOps {
			sendNotification(log, notifier)
		}
		return
	}

	// Claim attendance
	claimBody, err := doWithRetry(func() *http.Request {
		path := "/web/v1/game/endfield/attendance"
		ts := Timestamp()
		h := BuildHeaders(cred, role.GameRole, ts)
		h.Set("Sign", ComputeSignV2(path, ts, salt))
		req, _ := http.NewRequest("POST", AttendanceURL, nil)
		req.Header = h
		return req
	})
	if err != nil {
		msg := fmt.Sprintf("  → Attendance claim request failed: %s", err)
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}

	var claim attendanceClaimResponse
	if err := json.Unmarshal(claimBody, &claim); err != nil {
		msg := fmt.Sprintf("  → Attendance claim parse failed: %s", err)
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}
	if claim.Code != 0 {
		msg := fmt.Sprintf("  → Claim failed: %s", firstNonEmpty(claim.Message, string(claimBody)))
		log.Error(msg)
		sendNotification(log, notifier)
		return
	}

	var rewards []string
	if claim.Data != nil {
		for _, award := range claim.Data.AwardIds {
			if info, ok := claim.Data.ResourceInfoMap[award.ID]; ok {
				rewards = append(rewards, fmt.Sprintf("%s x%d", info.Name, info.Count))
			}
		}
	}

	if len(rewards) > 0 {
		log.Info(fmt.Sprintf("  → Checked in! Rewards: %s", strings.Join(rewards, ", ")))
	} else {
		log.Info("  → Successfully checked in!")
	}
	sendNotification(log, notifier)
}

func sendNotification(log *notify.MessageLog, notifier notify.Notifier) {
	if notifier == nil {
		return
	}
	if err := notifier.Send(log); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send notification: %s\n", err)
	}
}
