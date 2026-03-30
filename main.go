package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	base    = "https://sonarbay.com"
	version = "0.1.0"
	repo    = "sonarbay/news"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func cacheDir() string {
	dir, _ := os.UserCacheDir()
	return filepath.Join(dir, "sonarbay")
}

func checkVersionNotice() {
	cacheFile := filepath.Join(cacheDir(), "version-check")

	if data, err := os.ReadFile(cacheFile); err == nil {
		parts := strings.SplitN(string(data), "\n", 2)
		if len(parts) == 2 {
			if ts, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				if time.Now().Unix()-ts < 86400 {
					if latest := strings.TrimSpace(parts[1]); latest != "" && latest != version {
						printUpdateNotice(latest)
					}
					return
				}
			}
		}
	}

	go func() {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
		if err != nil || resp.StatusCode != 200 {
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var release struct {
			TagName string `json:"tag_name"`
		}
		json.Unmarshal(body, &release)
		latest := strings.TrimPrefix(release.TagName, "v")
		if latest == "" {
			return
		}

		os.MkdirAll(cacheDir(), 0755)
		os.WriteFile(cacheFile, []byte(fmt.Sprintf("%d\n%s", time.Now().Unix(), latest)), 0644)

		if latest != version {
			printUpdateNotice(latest)
		}
	}()
}

func printUpdateNotice(latest string) {
	fmt.Fprintf(os.Stderr, "\n  %s╭───────────────────────────────────────────╮%s\n", dim, reset)
	fmt.Fprintf(os.Stderr, "  %s│%s  Update available: %sv%s%s → %sv%s%s            %s│%s\n", dim, reset, gray, version, reset, green, latest, reset, dim, reset)
	fmt.Fprintf(os.Stderr, "  %s│%s  Run %ssonarbay update%s to upgrade            %s│%s\n", dim, reset, bold, reset, dim, reset)
	fmt.Fprintf(os.Stderr, "  %s╰───────────────────────────────────────────╯%s\n\n", dim, reset)
}

const (
	blue  = "\033[34m"
	green = "\033[32m"
	gray  = "\033[90m"
	bold  = "\033[1m"
	dim   = "\033[2m"
	red   = "\033[31m"
	reset = "\033[0m"
)

func header(text string)                  { fmt.Printf("\n%s%s%s%s\n\n", blue, bold, text, reset) }
func kv(key string, value string)         { fmt.Printf("  %s%s%s  %s\n", gray, key, reset, value) }
func divider()                            { fmt.Printf("%s%s%s\n", dim, strings.Repeat("─", 60), reset) }
func errMsg(msg string)                   { fmt.Fprintf(os.Stderr, "\n%s✗ %s%s\n\n", red, msg, reset) }

func apiGet(path string, params url.Values) ([]byte, error) {
	u := base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func printJSON(data []byte) {
	var v any
	if json.Unmarshal(data, &v) == nil {
		out, _ := json.MarshalIndent(v, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Println(string(data))
	}
}

func usage() {
	fmt.Println(`SonarBay — News Intelligence from your terminal

Usage: sonarbay <command> [flags]

Commands:
  search <query>    Search global news
  trending          See trending entities
  counts <query>    Time-series mention counts
  status            API health check
  update            Update CLI to latest version

Flags:
  --json            Output raw JSON (works with all commands)
  -h, --help        Show this help

Search Flags:
  -n <num>          Results per page (default: 10)
  -p <num>          Page number (default: 1)
  -s <sort>         Sort: relevance, newest, oldest (default: relevance)
  --country <code>  Filter by country code
  --source <name>   Filter by source domain

Trending Flags:
  -t <type>         Entity type: persons, organizations, countries, source (default: persons)
  -w <window>       Time window: 1h, 6h, 12h, 24h, 48h, 7d (default: 24h)
  -n <num>          Number of results (default: 20)

Counts Flags:
  -i <interval>     Bucket interval: 15m, 1h, 6h, 1d (default: 1h)
  -w <hours>        Time window in hours (default: 24)`)
}

func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--json" {
			flags["json"] = "true"
		} else if a == "-h" || a == "--help" {
			flags["help"] = "true"
		} else if strings.HasPrefix(a, "--") && i+1 < len(args) {
			flags[a[2:]] = args[i+1]
			i++
		} else if strings.HasPrefix(a, "-") && len(a) == 2 && i+1 < len(args) {
			flags[string(a[1])] = args[i+1]
			i++
		} else {
			if _, ok := flags["_arg"]; !ok {
				flags["_arg"] = a
			}
		}
	}
	return flags
}

func cmdSearch(flags map[string]string) {
	query := flags["_arg"]
	if query == "" {
		errMsg("Usage: sonarbay search <query>")
		os.Exit(1)
	}

	params := url.Values{"q": {query}}
	if v, ok := flags["n"]; ok {
		params.Set("per_page", v)
	}
	if v, ok := flags["p"]; ok {
		params.Set("page", v)
	}
	if v, ok := flags["s"]; ok {
		params.Set("sort", v)
	}
	if v, ok := flags["country"]; ok {
		params.Set("countries", v)
	}
	if v, ok := flags["source"]; ok {
		params.Set("sources", v)
	}

	raw, err := apiGet("/v1/search", params)
	if err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	if flags["json"] == "true" {
		printJSON(raw)
		return
	}

	var data struct {
		Query      string `json:"query"`
		Found      int    `json:"found"`
		Page       int    `json:"page"`
		TotalPages int    `json:"total_pages"`
		SearchTime int    `json:"search_time_ms"`
		Results    []struct {
			Title     string `json:"title"`
			PageTitle string `json:"pageTitle"`
			Source    string `json:"source"`
			Date      string `json:"date"`
			URL       string `json:"url"`
		} `json:"results"`
	}
	json.Unmarshal(raw, &data)

	header(fmt.Sprintf(`Search: "%s"`, data.Query))
	kv("Found", fmt.Sprintf("%d articles", data.Found))
	kv("Time", fmt.Sprintf("%dms", data.SearchTime))
	kv("Page", fmt.Sprintf("%d/%d", data.Page, data.TotalPages))
	divider()
	fmt.Println()

	for i, a := range data.Results {
		title := a.Title
		if title == "" {
			title = a.PageTitle
		}
		fmt.Printf("%s%2d.%s %s%s%s\n", dim, i+1, reset, bold, title, reset)
		meta := []string{}
		if a.Source != "" {
			meta = append(meta, a.Source)
		}
		if a.Date != "" {
			meta = append(meta, fmtDate(a.Date))
		}
		if len(meta) > 0 {
			fmt.Printf("    %s%s%s\n", gray, strings.Join(meta, "  ·  "), reset)
		}
		fmt.Println()
	}
}

func cmdTrending(flags map[string]string) {
	params := url.Values{}
	if v, ok := flags["t"]; ok {
		params.Set("type", v)
	}
	if v, ok := flags["w"]; ok {
		params.Set("hours", windowToHours(v))
	}
	if v, ok := flags["n"]; ok {
		params.Set("limit", v)
	}

	raw, err := apiGet("/v1/trending", params)
	if err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	if flags["json"] == "true" {
		printJSON(raw)
		return
	}

	var data struct {
		Type         string `json:"type"`
		WindowHours  int    `json:"window_hours"`
		TotalScanned int    `json:"total_articles_scanned"`
		Trending     []struct {
			Value string `json:"value"`
			Count int    `json:"count"`
		} `json:"trending"`
	}
	json.Unmarshal(raw, &data)

	header(fmt.Sprintf("Trending %s (%dh)", data.Type, data.WindowHours))
	kv("Articles scanned", fmt.Sprintf("%d", data.TotalScanned))
	divider()
	fmt.Println()

	maxCount := 1
	if len(data.Trending) > 0 && data.Trending[0].Count > 0 {
		maxCount = data.Trending[0].Count
	}

	for i, item := range data.Trending {
		barLen := int(math.Max(1, math.Round(float64(item.Count)/float64(maxCount)*20)))
		bar := strings.Repeat("█", barLen)
		fmt.Printf("  %s%2d.%s %s%s%s  %s%s%s  %s%d%s\n",
			dim, i+1, reset, bold, item.Value, reset, blue, bar, reset, gray, item.Count, reset)
	}
	fmt.Println()
}

func cmdCounts(flags map[string]string) {
	query := flags["_arg"]
	if query == "" {
		errMsg("Usage: sonarbay counts <query>")
		os.Exit(1)
	}

	params := url.Values{"q": {query}}
	if v, ok := flags["i"]; ok {
		params.Set("interval", v)
	}
	if v, ok := flags["w"]; ok {
		params.Set("hours", v)
	}

	raw, err := apiGet("/v1/counts", params)
	if err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	if flags["json"] == "true" {
		printJSON(raw)
		return
	}

	var data struct {
		Query    string `json:"query"`
		Interval string `json:"interval"`
		Hours    int    `json:"hours"`
		Buckets  []struct {
			Time  string `json:"time"`
			Count int    `json:"count"`
		} `json:"buckets"`
	}
	json.Unmarshal(raw, &data)

	header(fmt.Sprintf(`Counts: "%s" (%s buckets, %dh)`, data.Query, data.Interval, data.Hours))
	divider()
	fmt.Println()

	maxCount := 1
	for _, b := range data.Buckets {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}

	for _, b := range data.Buckets {
		barLen := int(math.Round(float64(b.Count) / float64(maxCount) * 30))
		bar := strings.Repeat("▓", barLen)
		t := b.Time
		if len(t) > 16 {
			t = t[11:16]
		}
		fmt.Printf("  %s%s%s  %s%s%s  %s%d%s\n", dim, t, reset, blue, bar, reset, gray, b.Count, reset)
	}
	fmt.Println()
}

func cmdStatus(flags map[string]string) {
	raw, err := apiGet("/v1/status", nil)
	if err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	if flags["json"] == "true" {
		printJSON(raw)
		return
	}

	var data struct {
		OK            bool   `json:"ok"`
		TotalArticles int    `json:"total_articles"`
		DateRange     struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"date_range"`
		LastSync      string `json:"last_sync"`
		SyncInterval  string `json:"sync_interval"`
		RetentionDays int    `json:"retention_days"`
	}
	json.Unmarshal(raw, &data)

	header("SonarBay Status")
	status := green + "✓ OK" + reset
	if !data.OK {
		status = red + "✗ Down" + reset
	}
	kv("Status", status)
	kv("Articles", strconv.Itoa(data.TotalArticles))
	kv("Date Range", fmt.Sprintf("%s → %s", data.DateRange.From, data.DateRange.To))
	kv("Last Sync", data.LastSync)
	kv("Sync Interval", data.SyncInterval)
	kv("Retention", fmt.Sprintf("%d days", data.RetentionDays))
	fmt.Println()
}

func fmtDate(raw string) string {
	if len(raw) == 14 {
		return raw[0:4] + "-" + raw[4:6] + "-" + raw[6:8] + " " + raw[8:10] + ":" + raw[10:12] + " UTC"
	}
	if len(raw) > 19 && strings.Contains(raw, "T") {
		return raw[:19] + " UTC"
	}
	return raw
}

func windowToHours(w string) string {
	if len(w) < 2 {
		return "24"
	}
	num := w[:len(w)-1]
	unit := w[len(w)-1]
	n, err := strconv.Atoi(num)
	if err != nil {
		return "24"
	}
	if unit == 'd' {
		return strconv.Itoa(n * 24)
	}
	return strconv.Itoa(n)
}

func cmdUpdate() {
	fmt.Println()
	fmt.Printf("  %sSonarBay CLI Update%s\n", bold, reset)
	fmt.Printf("  %s────────────────────%s\n", dim, reset)
	fmt.Printf("  Current version: v%s\n\n", version)

	fmt.Println("  Checking for updates...")
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		errMsg("Failed to check for updates: " + err.Error())
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg("No releases found yet")
		os.Exit(1)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &release)

	latestVer := strings.TrimPrefix(release.TagName, "v")
	if latestVer == version {
		fmt.Printf("  %s✓ Already on latest version (v%s)%s\n\n", green, version, reset)
		return
	}

	fmt.Printf("  New version available: %sv%s%s\n\n", green, latestVer, reset)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	var assetName string
	switch {
	case goos == "windows" && goarch == "amd64":
		assetName = "sonarbay-win-x64.exe"
	case goos == "linux" && goarch == "amd64":
		assetName = "sonarbay-linux-x64"
	case goos == "linux" && goarch == "arm64":
		assetName = "sonarbay-linux-arm64"
	case goos == "darwin" && goarch == "arm64":
		assetName = "sonarbay-darwin-arm64"
	case goos == "darwin" && goarch == "amd64":
		assetName = "sonarbay-darwin-x64"
	default:
		errMsg(fmt.Sprintf("Unsupported platform: %s/%s", goos, goarch))
		os.Exit(1)
	}

	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		errMsg("Binary not found for your platform: " + assetName)
		os.Exit(1)
	}

	fmt.Println("  Downloading...")
	dlResp, err := httpClient.Get(downloadURL)
	if err != nil {
		errMsg("Download failed: " + err.Error())
		os.Exit(1)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		errMsg(fmt.Sprintf("Download failed: HTTP %d", dlResp.StatusCode))
		os.Exit(1)
	}

	newBinary, err := io.ReadAll(dlResp.Body)
	if err != nil {
		errMsg("Failed to read binary: " + err.Error())
		os.Exit(1)
	}

	exe, err := os.Executable()
	if err != nil {
		errMsg("Can't find current binary path: " + err.Error())
		os.Exit(1)
	}

	oldPath := exe + ".old"
	os.Remove(oldPath)
	if err := os.Rename(exe, oldPath); err != nil {
		errMsg("Can't replace binary (try running as admin): " + err.Error())
		os.Exit(1)
	}

	if err := os.WriteFile(exe, newBinary, 0755); err != nil {
		os.Rename(oldPath, exe)
		errMsg("Failed to write new binary: " + err.Error())
		os.Exit(1)
	}

	os.Remove(oldPath)
	fmt.Printf("  %s✓ Updated to v%s%s\n\n", green, latestVer, reset)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		return
	}

	checkVersionNotice()

	cmd := args[0]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		usage()
		return
	}
	if cmd == "--version" || cmd == "-v" {
		fmt.Printf("sonarbay v%s\n", version)
		return
	}

	flags := parseFlags(args[1:])
	if flags["help"] == "true" {
		usage()
		return
	}

	switch cmd {
	case "search":
		cmdSearch(flags)
	case "trending":
		cmdTrending(flags)
	case "counts":
		cmdCounts(flags)
	case "status":
		cmdStatus(flags)
	case "update":
		cmdUpdate()
	default:
		errMsg(fmt.Sprintf("Unknown command: %s", cmd))
		usage()
		os.Exit(1)
	}
}
