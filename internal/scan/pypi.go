package scan

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

// pypiInfo fetches the latest version, summary and homepage for one PyPI
// package. Empty strings on error.
func pypiInfo(ctx context.Context, name string) (latest, desc, home string) {
	client := &http.Client{Timeout: 5 * time.Second}
	u := "https://pypi.org/pypi/" + url.PathEscape(name) + "/json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", ""
	}
	var d struct {
		Info struct {
			Version  string `json:"version"`
			Summary  string `json:"summary"`
			HomePage string `json:"home_page"`
		} `json:"info"`
	}
	if json.Unmarshal(body, &d) != nil {
		return "", "", ""
	}
	return d.Info.Version, d.Info.Summary, d.Info.HomePage
}
