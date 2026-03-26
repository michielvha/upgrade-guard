package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/vhco/upgrade-guard/internal/analyst"
)

// ADOConfig holds Azure DevOps connection details.
type ADOConfig struct {
	OrgURL  string // e.g. https://dev.azure.com/myorg
	Project string
	RepoID  string
	PRID    int
	Token   string
}

// PostADOComment posts (or updates) a PR comment thread in Azure DevOps.
func PostADOComment(cfg ADOConfig, result *analyst.AnalysisResult) error {
	body := formatPRComment(result)

	// Check for existing comment thread to update
	existingThreadID, existingCommentID, err := findExistingADOThread(cfg)
	if err != nil {
		return fmt.Errorf("searching for existing ADO thread: %w", err)
	}

	if existingThreadID != 0 {
		return updateADOComment(cfg, existingThreadID, existingCommentID, body)
	}

	return createADOThread(cfg, body)
}

func adoBaseURL(cfg ADOConfig) string {
	orgURL := strings.TrimSuffix(cfg.OrgURL, "/")
	return fmt.Sprintf("%s/%s/_apis/git/repositories/%s/pullRequests/%d",
		orgURL, cfg.Project, cfg.RepoID, cfg.PRID)
}

func adoRequest(method, url, token string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("", token)
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}

type adoThreadsResponse struct {
	Value []adoThread `json:"value"`
}

type adoThread struct {
	ID       int          `json:"id"`
	Comments []adoComment `json:"comments"`
}

type adoComment struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

func findExistingADOThread(cfg ADOConfig) (int, int, error) {
	url := adoBaseURL(cfg) + "/threads?api-version=7.1"

	resp, err := adoRequest("GET", url, cfg.Token, nil)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("ADO API returned status %d", resp.StatusCode)
	}

	var threads adoThreadsResponse
	if err := json.NewDecoder(resp.Body).Decode(&threads); err != nil {
		return 0, 0, err
	}

	for _, thread := range threads.Value {
		for _, comment := range thread.Comments {
			if strings.Contains(comment.Content, commentMarker) {
				return thread.ID, comment.ID, nil
			}
		}
	}

	return 0, 0, nil
}

func createADOThread(cfg ADOConfig, body string) error {
	url := adoBaseURL(cfg) + "/threads?api-version=7.1"

	payload := map[string]any{
		"comments": []map[string]any{
			{
				"parentCommentId": 0,
				"content":         body,
				"commentType":     1, // text
			},
		},
		"status": 1, // active
	}

	resp, err := adoRequest("POST", url, cfg.Token, payload)
	if err != nil {
		return fmt.Errorf("creating ADO thread: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ADO API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func updateADOComment(cfg ADOConfig, threadID, commentID int, body string) error {
	url := fmt.Sprintf("%s/threads/%d/comments/%d?api-version=7.1",
		adoBaseURL(cfg), threadID, commentID)

	payload := map[string]any{
		"content": body,
	}

	resp, err := adoRequest("PATCH", url, cfg.Token, payload)
	if err != nil {
		return fmt.Errorf("updating ADO comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ADO API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
