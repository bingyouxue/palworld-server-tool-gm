// Package mods handles downloading and installing PalDefender and UE4SS
// from their GitHub latest releases into the server's Pal/Binaries/Win64 dir.
package mods

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// httpClient is for short API/version queries (20 s is plenty for JSON).
var httpClient = &http.Client{Timeout: 20 * time.Second}

// dlClient is for file downloads — no global Timeout so that large zips
// can transfer over slow mainland connections without being cut off mid-body.
var dlClient = &http.Client{}

type Component string

const (
	ComponentPalDefender Component = "paldefender"
	ComponentUE4SS       Component = "ue4ss"
)

type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelBeta   Channel = "beta"
)

var ghRepos = map[Component]struct {
	Repo    string
	Asset   string // substring match
	EnvURL  string
}{
	ComponentPalDefender: {
		Repo:   "Ultimeit/PalDefender",
		Asset:  "PalDefender",
		EnvURL: "PALSERVER_PALDEFENDER_URL",
	},
	ComponentUE4SS: {
		Repo:   "UE4SS-RE/RE-UE4SS",
		Asset:  "UE4SS",
		EnvURL: "PALSERVER_UE4SS_URL",
	},
}

type ModsMarker struct {
	PalDefender string              `json:"paldefender,omitempty"`
	UE4SS       string              `json:"ue4ss,omitempty"`
	Files       map[string][]string `json:"files,omitempty"`
}

// Win64Dir returns the Pal/Binaries/Win64 path under serverRoot.
func Win64Dir(serverRoot string) string {
	return filepath.Join(serverRoot, "Pal", "Binaries", "Win64")
}

func markerPath(serverRoot string) string {
	return filepath.Join(Win64Dir(serverRoot), ".palserver-mods.json")
}

func ReadMarker(serverRoot string) ModsMarker {
	var m ModsMarker
	data, err := os.ReadFile(markerPath(serverRoot))
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

func writeMarker(serverRoot string, component Component, version string, files []string) {
	m := ReadMarker(serverRoot)
	if m.Files == nil {
		m.Files = map[string][]string{}
	}
	switch component {
	case ComponentPalDefender:
		m.PalDefender = version
	case ComponentUE4SS:
		m.UE4SS = version
	}
	m.Files[string(component)] = files
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(markerPath(serverRoot), data, 0644)
}

// InstalledVersion returns the marker-recorded version or "" if not installed.
func InstalledVersion(serverRoot string, component Component) string {
	m := ReadMarker(serverRoot)
	switch component {
	case ComponentPalDefender:
		return m.PalDefender
	case ComponentUE4SS:
		return m.UE4SS
	}
	return ""
}

// IsInstalled checks the actual DLL files on disk.
func IsInstalled(serverRoot string, component Component) bool {
	w64 := Win64Dir(serverRoot)
	switch component {
	case ComponentPalDefender:
		_, err := os.Stat(filepath.Join(w64, "PalDefender.dll"))
		return err == nil
	case ComponentUE4SS:
		_, err1 := os.Stat(filepath.Join(w64, "ue4ss", "UE4SS.dll"))
		_, err2 := os.Stat(filepath.Join(w64, "UE4SS.dll"))
		return err1 == nil || err2 == nil
	}
	return false
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func resolveDownload(component Component, channel Channel, progressFn ProgressFunc) (version, url string, err error) {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	info := ghRepos[component]
	if v := os.Getenv(info.EnvURL); v != "" {
		return "custom", v, nil
	}

	// GitHub API mirrors tried in order; first success wins.
	// Override entirely with PALSERVER_GH_API env var.
	apiMirrors := []string{
		"https://api.github.com",
		"https://ghapi.huchen.dev",
		"https://api.github.com", // retry direct
	}
	if base := os.Getenv("PALSERVER_GH_API"); base != "" {
		apiMirrors = []string{base}
	}

	var release ghRelease
	var resolveErr error

	for _, base := range apiMirrors {
		var apiURL string
		if channel == ChannelBeta {
			apiURL = fmt.Sprintf("%s/repos/%s/releases?per_page=15", base, info.Repo)
		} else {
			apiURL = fmt.Sprintf("%s/repos/%s/releases/latest", base, info.Repo)
		}
		progress(fmt.Sprintf("查询版本: %s ...", base))

		req, _ := http.NewRequest("GET", apiURL, nil)
		req.Header.Set("User-Agent", "pst-gm")
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, httpErr := httpClient.Do(req)
		if httpErr != nil {
			resolveErr = fmt.Errorf("API %s 失败: %v", base, httpErr)
			progress(fmt.Sprintf("[跳过] %v", resolveErr))
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			resolveErr = fmt.Errorf("API %s 返回 HTTP %d", base, resp.StatusCode)
			progress(fmt.Sprintf("[跳过] %v", resolveErr))
			continue
		}

		if channel == ChannelBeta {
			var releases []ghRelease
			if e := json.Unmarshal(body, &releases); e != nil {
				resolveErr = fmt.Errorf("解析 releases: %v", e)
				progress(fmt.Sprintf("[跳过] %v", resolveErr))
				continue
			}
			for _, r := range releases {
				if !r.Draft {
					release = r
					break
				}
			}
		} else {
			if e := json.Unmarshal(body, &release); e != nil {
				resolveErr = fmt.Errorf("解析 release: %v", e)
				progress(fmt.Sprintf("[跳过] %v", resolveErr))
				continue
			}
		}
		if release.TagName != "" {
			resolveErr = nil
			break
		}
		resolveErr = fmt.Errorf("未找到 %s 的发布版本", info.Repo)
		progress(fmt.Sprintf("[跳过] %v", resolveErr))
	}

	if resolveErr != nil {
		return "", "", resolveErr
	}
	if release.TagName == "" {
		return "", "", fmt.Errorf("未找到 %s 的发布版本", info.Repo)
	}

	assetLower := strings.ToLower(info.Asset)
	for _, a := range release.Assets {
		nameLower := strings.ToLower(a.Name)
		if strings.Contains(nameLower, assetLower) && strings.HasSuffix(nameLower, ".zip") {
			return release.TagName, a.BrowserDownloadURL, nil
		}
	}
	return "", "", fmt.Errorf("在 %s@%s 中未找到匹配的 .zip 附件", info.Repo, release.TagName)
}

// ProgressFunc is called with progress messages during installation.
type ProgressFunc func(msg string)

// Install downloads and extracts a mod component into serverRoot's Win64 dir.
// progressFn may be nil.
func Install(serverRoot string, component Component, channel Channel, progressFn ProgressFunc) (version string, err error) {
	w64 := Win64Dir(serverRoot)
	if err := os.MkdirAll(w64, 0755); err != nil {
		return "", fmt.Errorf("create Win64 dir: %w", err)
	}

	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	progress("正在查询最新版本...")
	version, dlURL, err := resolveDownload(component, channel, progressFn)
	if err != nil {
		return "", err
	}
	progress(fmt.Sprintf("发现版本 %s，开始下载...", version))

	// Download zip with mirror fallback.
	// Mirrors rewrite the github.com download URL prefix.
	// PALSERVER_GH_DOWNLOAD env var can provide a single override prefix.
	// Download mirror candidates — tried in order until one succeeds.
	// Direct GitHub first, then mainland-accessible mirrors.
	// Override with PALSERVER_GH_DOWNLOAD env var (a single prefix URL).
	ghDownloadMirrors := []string{
		"",                           // direct github.com
		"https://ghproxy.com/",
		"https://mirror.ghproxy.com/",
		"https://ghproxy.net/",
		"https://gh.api.99988866.xyz/",
	}
	if envMirror := os.Getenv("PALSERVER_GH_DOWNLOAD"); envMirror != "" {
		ghDownloadMirrors = []string{envMirror}
	}

	// Try each mirror in order; retry the full download+body on each failure.
	// Using dlClient (no global Timeout) so large zips don't get cut off mid-body.
	var tmpPath string
	var dlErr error
	var written int64
	for i, mirror := range ghDownloadMirrors {
		candidate := dlURL
		if mirror != "" {
			candidate = mirror + dlURL
		}
		progress(fmt.Sprintf("尝试下载 (%d/%d): %s", i+1, len(ghDownloadMirrors), candidate))

		var resp *http.Response
		resp, dlErr = dlClient.Get(candidate)
		if dlErr != nil {
			progress(fmt.Sprintf("[跳过] 连接失败: %v", dlErr))
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			dlErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			progress(fmt.Sprintf("[跳过] %v", dlErr))
			continue
		}
		progress("连接成功，正在下载...")

		// Write body to a temp file so we can retry if the stream dies.
		var tf *os.File
		tf, dlErr = os.CreateTemp("", string(component)+"_*.zip")
		if dlErr != nil {
			resp.Body.Close()
			return "", fmt.Errorf("create temp file: %w", dlErr)
		}
		curPath := tf.Name()
		written, dlErr = io.Copy(tf, resp.Body)
		tf.Close()
		resp.Body.Close()
		if dlErr != nil {
			os.Remove(curPath)
			progress(fmt.Sprintf("[跳过] 下载中断: %v", dlErr))
			continue
		}
		// Success
		tmpPath = curPath
		dlErr = nil
		break
	}
	if dlErr != nil || tmpPath == "" {
		if dlErr == nil {
			dlErr = fmt.Errorf("所有下载地址均失败")
		}
		return "", fmt.Errorf("download failed: %w", dlErr)
	}
	defer os.Remove(tmpPath)
	progress(fmt.Sprintf("下载完成 (%.1f MB)，开始解压...", float64(written)/1024/1024))

	// Extract
	files, err := extractZip(tmpPath, w64)
	if err != nil {
		return "", fmt.Errorf("extract zip: %w", err)
	}
	writeMarker(serverRoot, component, version, files)
	progress(fmt.Sprintf("安装完成！版本 %s，共释放 %d 个顶层条目。", version, len(files)))
	return version, nil
}

// extractZip extracts a zip archive to destDir and returns the top-level
// entry names that were extracted (for marker tracking).
func extractZip(zipPath, destDir string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	topSet := map[string]struct{}{}
	for _, f := range r.File {
		// normalise path separators
		name := strings.ReplaceAll(f.Name, "\\", "/")
		name = strings.TrimPrefix(name, "./")
		if name == "" {
			continue
		}
		// record top-level entry
		seg := strings.SplitN(name, "/", 2)[0]
		if seg != "" {
			topSet[seg] = struct{}{}
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(name))
		// safety: must stay inside destDir
		if !strings.HasPrefix(filepath.Clean(destPath)+string(os.PathSeparator), filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, err
		}
		out, err := os.Create(destPath)
		if err != nil {
			return nil, err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return nil, err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return nil, err
		}
	}

	tops := make([]string, 0, len(topSet))
	for k := range topSet {
		tops = append(tops, k)
	}
	return tops, nil
}

// Remove uninstalls a component by deleting the files recorded in the marker.
func Remove(serverRoot string, component Component) error {
	w64 := Win64Dir(serverRoot)
	m := ReadMarker(serverRoot)

	other := ComponentUE4SS
	if component == ComponentUE4SS {
		other = ComponentPalDefender
	}
	keepSet := map[string]struct{}{}
	if m.Files != nil {
		for _, f := range m.Files[string(other)] {
			keepSet[f] = struct{}{}
		}
	}

	var targets []string
	if m.Files != nil && len(m.Files[string(component)]) > 0 {
		targets = m.Files[string(component)]
	} else {
		// fallback defaults
		switch component {
		case ComponentPalDefender:
			targets = []string{"PalDefender.dll", "PalDefender", "d3d9.dll"}
		case ComponentUE4SS:
			targets = []string{"UE4SS.dll", "UE4SS-settings.ini", "ue4ss", "Mods", "dwmapi.dll"}
		}
	}

	for _, rel := range targets {
		if _, ok := keepSet[rel]; ok {
			continue
		}
		p := filepath.Join(w64, rel)
		_ = os.RemoveAll(p)
	}

	// update marker
	if m.Files == nil {
		m.Files = map[string][]string{}
	}
	delete(m.Files, string(component))
	switch component {
	case ComponentPalDefender:
		m.PalDefender = ""
	case ComponentUE4SS:
		m.UE4SS = ""
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(markerPath(serverRoot), data, 0644)
	return nil
}
