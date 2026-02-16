package skport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kyoukaya/endfield-daily/notify"
)

// RunAccount authenticates, fetches roles, and checks in for one account.
func RunAccount(token string, index int, log *notify.MessageLog) error {
	creds, err := Authenticate(token)
	if err != nil {
		log.Error(fmt.Sprintf("Account %d: %s", index, err))
		return err
	}
	log.Info(fmt.Sprintf("Account %d: obtained cred and salt", index))

	roles, err := getPlayerRoles(creds.Cred, creds.Salt)
	if err != nil {
		log.Error(fmt.Sprintf("Account %d: %s", index, err))
		return err
	}
	log.Info(fmt.Sprintf("Account %d: Found %d role(s)", index, len(roles)))

	for i, role := range roles {
		label := fmt.Sprintf("%s (Lv.%d) [%s]", role.Nickname, role.Level, role.Server)
		err := checkInRole(creds.Cred, creds.Salt, role, label, log)
		if err != nil {
			log.Error(fmt.Sprintf("  → %s: %s", label, err))
		}
		if i < len(roles)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return nil
}

func getPlayerRoles(cred, salt string) ([]Role, error) {
	path := "/api/v1/game/player/binding"
	ts := Timestamp()
	headers := BuildHeaders(cred, "", ts)
	headers.Set("Sign", ComputeSignV2(path, ts, salt))

	req, _ := http.NewRequest("GET", BindingURL, nil)
	req.Header = headers

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binding request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

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

func checkInRole(cred, salt string, role Role, label string, log *notify.MessageLog) error {
	path := "/web/v1/game/endfield/attendance"
	ts := Timestamp()
	headers := BuildHeaders(cred, role.GameRole, ts)
	headers.Set("Sign", ComputeSignV2(path, ts, salt))

	// Check attendance status
	req, _ := http.NewRequest("GET", AttendanceURL, nil)
	req.Header = headers.Clone()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("attendance check request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var status attendanceStatusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("attendance check parse failed: %w", err)
	}
	if status.Code != 0 {
		return fmt.Errorf("attendance status check failed: %s", firstNonEmpty(status.Message, string(body)))
	}

	if status.Data != nil && status.Data.HasToday {
		log.Info(fmt.Sprintf("  → %s: Already checked in today", label))
		return nil
	}

	// Claim attendance
	claimReq, _ := http.NewRequest("POST", AttendanceURL, nil)
	claimReq.Header = headers.Clone()

	claimResp, err := http.DefaultClient.Do(claimReq)
	if err != nil {
		return fmt.Errorf("attendance claim request failed: %w", err)
	}
	defer claimResp.Body.Close()
	claimBody, _ := io.ReadAll(claimResp.Body)

	var claim attendanceClaimResponse
	if err := json.Unmarshal(claimBody, &claim); err != nil {
		return fmt.Errorf("attendance claim parse failed: %w", err)
	}
	if claim.Code != 0 {
		return fmt.Errorf("claim failed: %s", firstNonEmpty(claim.Message, string(claimBody)))
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
		log.Info(fmt.Sprintf("  → %s: Checked in! Rewards: %s", label, strings.Join(rewards, ", ")))
	} else {
		log.Info(fmt.Sprintf("  → %s: Successfully checked in!", label))
	}
	return nil
}
