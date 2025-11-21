package e2etesting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type teamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type createTeamRequest struct {
	TeamName string       `json:"team_name"`
	Members  []teamMember `json:"members"`
}

type teamEntity struct {
	Name    string       `json:"name"`
	Members []teamMember `json:"members"`
}

type createTeamResponse struct {
	Team teamEntity `json:"team"`
}

type userResponse struct {
	User struct {
		ID       string `json:"id"`
		IsActive bool   `json:"is_active"`
	} `json:"user"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type createPRResponse struct {
	PR struct {
		ID        string   `json:"pull_request_id"`
		AuthorID  string   `json:"author_id"`
		Status    string   `json:"status"`
		Reviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
}

type mergeResponse struct {
	PR struct {
		ID       string     `json:"pull_request_id"`
		Status   string     `json:"status"`
		MergedAt *time.Time `json:"mergedAt"`
	} `json:"pr"`
}

type reassignResponse struct {
	PR struct {
		ID string `json:"pull_request_id"`
	} `json:"pr"`
	ReplacedBy string `json:"replaced_by"`
}

type statsResponse struct {
	Stats struct {
		ReviewerAssignments []struct {
			UserID      string `json:"user_id"`
			Assignments int    `json:"assignments"`
		} `json:"reviewer_assignments"`
		PRStatus struct {
			Total            int     `json:"total"`
			Open             int     `json:"open"`
			Merged           int     `json:"merged"`
			AverageReviewers float64 `json:"average_reviewers"`
		} `json:"pr_status"`
		TeamMembers []struct {
			TeamName string `json:"team_name"`
			Active   int    `json:"active_members"`
			Inactive int    `json:"inactive_members"`
		} `json:"team_members"`
		PRLifetime struct {
			AverageMerge struct {
				Days    int `json:"days"`
				Hours   int `json:"hours"`
				Minutes int `json:"minutes"`
			} `json:"average_merge"`
			OpenOlderThan7Days int `json:"open_older_than_7_days"`
		} `json:"pr_lifetime"`
	} `json:"stats"`
}

type reviewsResponse struct {
	UserID       string `json:"user_id"`
	PullRequests []struct {
		PullRequestID string `json:"pull_request_id"`
	} `json:"pull_requests"`
}

type deactivateResponse struct {
	Result struct {
		Deactivated []string             `json:"deactivated_user_ids"`
		Successful  []reassignmentRecord `json:"successful_reassignments"`
		Failed      []reassignmentRecord `json:"failed_reassignments"`
	} `json:"result"`
}

type reassignmentRecord struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
	NewReviewerID string `json:"new_reviewer_id"`
	Error         string `json:"error"`
}

func TestHealthEndpoint(t *testing.T) {
	baseURL := requireBaseURL(t)
	body := doRequest(t, http.MethodGet, baseURL+"/health", nil, http.StatusOK)
	var resp map[string]string
	decodeJSON(t, body, &resp)
	if resp["status"] != "OK" {
		t.Fatalf("unexpected status: %v", resp)
	}
}

func TestTeamCreateAndGet(t *testing.T) {
	baseURL := requireBaseURL(t)
	members := []teamMember{
		{UserID: "u1", Username: "alice", IsActive: true},
		{UserID: "u2", Username: "bob", IsActive: true},
		{UserID: "u3", Username: "carol", IsActive: true},
	}
	teamName := fmt.Sprintf("backend-%s", randomID("t"))
	req := createTeamRequest{TeamName: teamName, Members: members}
	body := doRequest(t, http.MethodPost, baseURL+"/team/add", req, http.StatusCreated)
	var created createTeamResponse
	decodeJSON(t, body, &created)
	if len(created.Team.Members) != len(members) {
		t.Fatalf("expected %d members, got %d", len(members), len(created.Team.Members))
	}

	teamResp := getTeam(t, baseURL, teamName)
	if len(teamResp.Members) != len(members) {
		t.Fatalf("team get mismatch: %d vs %d", len(teamResp.Members), len(members))
	}
}

func TestTeamCreateDuplicate(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("dup-%s", randomID("team"))
	members := []teamMember{{UserID: randomID("user"), Username: "dup", IsActive: true}}
	req := createTeamRequest{TeamName: teamName, Members: members}
	doRequest(t, http.MethodPost, baseURL+"/team/add", req, http.StatusCreated)
	body := doRequest(t, http.MethodPost, baseURL+"/team/add", req, http.StatusBadRequest)
	var errResp errorResponse
	decodeJSON(t, body, &errResp)
	if errResp.Error.Code != "TEAM_EXISTS" {
		t.Fatalf("expected TEAM_EXISTS, got %s", errResp.Error.Code)
	}
}

func TestUserSetIsActive(t *testing.T) {
	baseURL := requireBaseURL(t)
	userID := randomID("user")
	teamName := fmt.Sprintf("toggle-%s", randomID("team"))
	doRequest(t, http.MethodPost, baseURL+"/team/add", createTeamRequest{
		TeamName: teamName,
		Members:  []teamMember{{UserID: userID, Username: "toggle", IsActive: true}},
	}, http.StatusCreated)
	payload := map[string]any{"user_id": userID, "is_active": false}
	body := doRequest(t, http.MethodPost, baseURL+"/users/setIsActive", payload, http.StatusOK)
	var user userResponse
	decodeJSON(t, body, &user)
	if user.User.IsActive {
		t.Fatalf("expected user to be inactive")
	}
}

func TestUserSetIsActiveNotFound(t *testing.T) {
	baseURL := requireBaseURL(t)
	payload := map[string]any{"user_id": "unknown", "is_active": false}
	body := doRequest(t, http.MethodPost, baseURL+"/users/setIsActive", payload, http.StatusNotFound)
	var errResp errorResponse
	decodeJSON(t, body, &errResp)
	if errResp.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", errResp.Error.Code)
	}
}

func TestPullRequestCreateAssign(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("pr-team-%s", randomID("team"))
	members := make([]teamMember, 0, 4)
	for i := 0; i < 4; i++ {
		members = append(members, teamMember{UserID: randomID("user"), Username: fmt.Sprintf("member_%d", i), IsActive: true})
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/auto-assign",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var resp createPRResponse
	decodeJSON(t, body, &resp)
	if len(resp.PR.Reviewers) == 0 || len(resp.PR.Reviewers) > 2 {
		t.Fatalf("unexpected reviewers count %d", len(resp.PR.Reviewers))
	}
	for _, reviewer := range resp.PR.Reviewers {
		if reviewer == payload["author_id"] {
			t.Fatalf("author should not be reviewer")
		}
		if !containsUser(members, reviewer) {
			t.Fatalf("reviewer %s not in team", reviewer)
		}
	}
}

func TestPullRequestCreateDuplicate(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("dup-pr-%s", randomID("team"))
	members := []teamMember{{UserID: randomID("user"), Username: "author", IsActive: true}, {UserID: randomID("user"), Username: "r1", IsActive: true}}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   "pr-dup",
		"pull_request_name": "feature/dup",
		"author_id":         members[0].UserID,
	}
	doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusConflict)
	var errResp errorResponse
	decodeJSON(t, body, &errResp)
	if errResp.Error.Code != "PR_EXISTS" {
		t.Fatalf("expected PR_EXISTS, got %s", errResp.Error.Code)
	}
}

func TestPullRequestCreateAuthorNotFound(t *testing.T) {
	baseURL := requireBaseURL(t)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/missing-author",
		"author_id":         "unknown-author",
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusNotFound)
	var errResp errorResponse
	decodeJSON(t, body, &errResp)
	if errResp.Error.Code != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %s", errResp.Error.Code)
	}
}

func TestPullRequestMergeFlow(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("merge-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "reviewer", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/merge",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)

	mergeBody := doRequest(t, http.MethodPost, baseURL+"/pullRequest/merge", map[string]string{"pull_request_id": pr.PR.ID}, http.StatusOK)
	var mergeResp mergeResponse
	decodeJSON(t, mergeBody, &mergeResp)
	if mergeResp.PR.Status != "MERGED" || mergeResp.PR.MergedAt == nil {
		t.Fatalf("merge failed: %+v", mergeResp.PR)
	}
	firstMergedAt := *mergeResp.PR.MergedAt

	mergeBody = doRequest(t, http.MethodPost, baseURL+"/pullRequest/merge", map[string]string{"pull_request_id": pr.PR.ID}, http.StatusOK)
	decodeJSON(t, mergeBody, &mergeResp)
	if mergeResp.PR.Status != "MERGED" || mergeResp.PR.MergedAt == nil || !mergeResp.PR.MergedAt.Equal(firstMergedAt) {
		t.Fatalf("merge not idempotent: %v %v", mergeResp.PR.MergedAt, firstMergedAt)
	}

	doRequest(t, http.MethodPost, baseURL+"/pullRequest/merge", map[string]string{"pull_request_id": "unknown"}, http.StatusNotFound)
}

func TestPullRequestReassignSuccess(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("reassign-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "r1", IsActive: true},
		{UserID: randomID("user"), Username: "r2", IsActive: true},
		{UserID: randomID("user"), Username: "r3", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/reassign",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	if len(pr.PR.Reviewers) < 2 {
		t.Skip("not enough reviewers assigned for reassign test")
	}
	old := pr.PR.Reviewers[0]
	reassignBody := doRequest(t, http.MethodPost, baseURL+"/pullRequest/reassign", map[string]string{
		"pull_request_id": pr.PR.ID,
		"old_user_id":     old,
	}, http.StatusOK)
	var reassign reassignResponse
	decodeJSON(t, reassignBody, &reassign)
	if reassign.ReplacedBy == "" || reassign.ReplacedBy == old || !containsUser(members, reassign.ReplacedBy) {
		t.Fatalf("invalid replacement: %+v", reassign)
	}
}

func TestPullRequestReassignMerged(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("reassign-merged-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "r1", IsActive: true},
		{UserID: randomID("user"), Username: "r2", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/reassign-merged",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	doRequest(t, http.MethodPost, baseURL+"/pullRequest/merge", map[string]string{"pull_request_id": pr.PR.ID}, http.StatusOK)
	reassignBody := doRequest(t, http.MethodPost, baseURL+"/pullRequest/reassign", map[string]string{
		"pull_request_id": pr.PR.ID,
		"old_user_id":     members[1].UserID,
	}, http.StatusConflict)
	var errResp errorResponse
	decodeJSON(t, reassignBody, &errResp)
	if errResp.Error.Code != "PR_MERGED" {
		t.Fatalf("expected PR_MERGED, got %s", errResp.Error.Code)
	}
}

func TestPullRequestReassignNotAssigned(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("reassign-miss-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "r1", IsActive: true},
		{UserID: randomID("user"), Username: "r2", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/reassign-miss",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	nonReviewer := members[2].UserID
	for _, r := range pr.PR.Reviewers {
		if r == nonReviewer {
			nonReviewer = randomID("user")
			break
		}
	}
	reassignBody := doRequest(t, http.MethodPost, baseURL+"/pullRequest/reassign", map[string]string{
		"pull_request_id": pr.PR.ID,
		"old_user_id":     nonReviewer,
	}, http.StatusConflict)
	var errResp errorResponse
	decodeJSON(t, reassignBody, &errResp)
	if errResp.Error.Code != "NOT_ASSIGNED" {
		t.Fatalf("expected NOT_ASSIGNED, got %s", errResp.Error.Code)
	}
}

func TestPullRequestReassignNoCandidate(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("reassign-nocand-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "r1", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/reassign-nocand",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	reassignBody := doRequest(t, http.MethodPost, baseURL+"/pullRequest/reassign", map[string]string{
		"pull_request_id": pr.PR.ID,
		"old_user_id":     members[1].UserID,
	}, http.StatusConflict)
	var errResp errorResponse
	decodeJSON(t, reassignBody, &errResp)
	if errResp.Error.Code != "NO_CANDIDATE" {
		t.Fatalf("expected NO_CANDIDATE, got %s", errResp.Error.Code)
	}
}

func TestTeamDeactivateWithReassign(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("deact-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "r1", IsActive: true},
		{UserID: randomID("user"), Username: "r2", IsActive: true},
		{UserID: randomID("user"), Username: "r3", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/deact",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	if len(pr.PR.Reviewers) < 2 {
		t.Skip("not enough reviewers for deactivate test")
	}
	disable := pr.PR.Reviewers[:2]
	deact := deactivateTeam(t, baseURL, teamName, disable, http.StatusOK)
	if len(deact.Result.Deactivated) != len(disable) {
		t.Fatalf("expected %d deactivated, got %d", len(disable), len(deact.Result.Deactivated))
	}
	if len(deact.Result.Successful) == 0 {
		t.Fatalf("expected successful reassignments")
	}

	teamResp := getTeam(t, baseURL, teamName)
	for _, member := range teamResp.Members {
		if contains(disable, member.UserID) && member.IsActive {
			t.Fatalf("member %s should be inactive", member.UserID)
		}
	}
}

func TestTeamDeactivateNoReplacement(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("deact-fail-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "only", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/deact-fail",
		"author_id":         members[0].UserID,
	}
	doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	disable := []string{members[0].UserID, members[1].UserID}
	deact := deactivateTeam(t, baseURL, teamName, disable, http.StatusOK)
	if len(deact.Result.Failed) == 0 {
		t.Fatalf("expected failed reassignments")
	}
	if !strings.Contains(deact.Result.Failed[0].Error, "no active replacement") {
		t.Fatalf("unexpected error: %s", deact.Result.Failed[0].Error)
	}
}

func TestStatsSummary(t *testing.T) {
	baseURL := requireBaseURL(t)
	body := doRequest(t, http.MethodGet, baseURL+"/stats/summary", nil, http.StatusOK)
	var stats statsResponse
	decodeJSON(t, body, &stats)
	if stats.Stats.PRStatus.Total < 0 {
		t.Fatalf("invalid stats: %+v", stats.Stats.PRStatus)
	}
}

func TestUserGetReviewScenarios(t *testing.T) {
	baseURL := requireBaseURL(t)
	teamName := fmt.Sprintf("reviews-%s", randomID("team"))
	members := []teamMember{
		{UserID: randomID("user"), Username: "author", IsActive: true},
		{UserID: randomID("user"), Username: "reviewer", IsActive: true},
	}
	createTeam(t, baseURL, teamName, members)
	payload := map[string]string{
		"pull_request_id":   randomID("pr"),
		"pull_request_name": "feature/review",
		"author_id":         members[0].UserID,
	}
	body := doRequest(t, http.MethodPost, baseURL+"/pullRequest/create", payload, http.StatusCreated)
	var pr createPRResponse
	decodeJSON(t, body, &pr)
	if len(pr.PR.Reviewers) == 0 {
		t.Skip("no reviewers assigned")
	}

	reviewsBody := doRequest(t, http.MethodGet, fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, pr.PR.Reviewers[0]), nil, http.StatusOK)
	var reviews reviewsResponse
	decodeJSON(t, reviewsBody, &reviews)
	if len(reviews.PullRequests) == 0 {
		t.Fatalf("expected PRs for reviewer")
	}

	emptyBody := doRequest(t, http.MethodGet, fmt.Sprintf("%s/users/getReview?user_id=%s", baseURL, "no-pr-user"), nil, http.StatusOK)
	var empty reviewsResponse
	decodeJSON(t, emptyBody, &empty)
	if len(empty.PullRequests) != 0 {
		t.Fatalf("expected empty PR list")
	}
}

// helpers

func requireBaseURL(t *testing.T) string {
	baseURL := os.Getenv("E2E_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if !isAlive(baseURL) {
		t.Skipf("service %s is not reachable", baseURL)
	}
	return baseURL
}

func isAlive(baseURL string) bool {
	resp, err := httpClient.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func randomID(prefix string) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("%s%s", prefix, string(b))
}

func doRequest(t *testing.T, method, url string, payload any, expected int) []byte {
	t.Helper()
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request %s failed: %v", url, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != expected {
		t.Fatalf("unexpected status %d (want %d): %s", resp.StatusCode, expected, string(respBody))
	}
	return respBody
}

func decodeJSON[T any](t *testing.T, data []byte, out *T) {
	t.Helper()
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func createTeam(t *testing.T, baseURL string, teamName string, members []teamMember) {
	doRequest(t, http.MethodPost, baseURL+"/team/add", createTeamRequest{TeamName: teamName, Members: members}, http.StatusCreated)
}

func getTeam(t *testing.T, baseURL, teamName string) teamEntity {
	body := doRequest(t, http.MethodGet, fmt.Sprintf("%s/team/get?team_name=%s", baseURL, teamName), nil, http.StatusOK)
	var team teamEntity
	decodeJSON(t, body, &team)
	return team
}

func deactivateTeam(t *testing.T, baseURL, teamName string, userIDs []string, expected int) deactivateResponse {
	payload := map[string]any{"team_name": teamName, "user_ids": userIDs}
	body := doRequest(t, http.MethodPost, baseURL+"/team/deactivate", payload, expected)
	var resp deactivateResponse
	decodeJSON(t, body, &resp)
	return resp
}

func containsUser(members []teamMember, userID string) bool {
	for _, m := range members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
