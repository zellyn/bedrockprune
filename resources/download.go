package resources

import (
	"archive/zip"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
)

const LAUNCHER_MANIFEST_URL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
const BEDROCK_DOWNLOAD_PAGE_URL = "https://mcpedl.org/downloading/"

const EXAMPLE_VERSION_MANIFEST_URL = "https://piston-meta.mojang.com/v1/packages/1ea0afa4d4caba3a752fd8f0725b7b83eb879514/1.20.4.json"

type Version struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	Time        time.Time `json:"time"`
	ReleaseTime time.Time `json:"releaseTime"`
}

// Thanks to https://mholt.github.io/json-to-go/
type LauncherManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []Version `json:"versions"`
}

// Thanks to https://mholt.github.io/json-to-go/
type VersionManifest struct {
	Arguments struct {
		Game []any `json:"game"`
		Jvm  []any `json:"jvm"`
	} `json:"arguments"`
	AssetIndex struct {
		ID        string `json:"id"`
		Sha1      string `json:"sha1"`
		Size      int    `json:"size"`
		TotalSize int    `json:"totalSize"`
		URL       string `json:"url"`
	} `json:"assetIndex"`
	Assets          string `json:"assets"`
	ComplianceLevel int    `json:"complianceLevel"`
	Downloads       struct {
		Client struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"client"`
		ClientMappings struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"client_mappings"`
		Server struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"server"`
		ServerMappings struct {
			Sha1 string `json:"sha1"`
			Size int    `json:"size"`
			URL  string `json:"url"`
		} `json:"server_mappings"`
	} `json:"downloads"`
	ID          string `json:"id"`
	JavaVersion struct {
		Component    string `json:"component"`
		MajorVersion int    `json:"majorVersion"`
	} `json:"javaVersion"`
	Libraries []struct {
		Downloads struct {
			Artifact struct {
				Path string `json:"path"`
				Sha1 string `json:"sha1"`
				Size int    `json:"size"`
				URL  string `json:"url"`
			} `json:"artifact"`
		} `json:"downloads"`
		Name  string `json:"name"`
		Rules []struct {
			Action string `json:"action"`
			Os     struct {
				Name string `json:"name"`
			} `json:"os"`
		} `json:"rules,omitempty"`
	} `json:"libraries"`
	Logging struct {
		Client struct {
			Argument string `json:"argument"`
			File     struct {
				ID   string `json:"id"`
				Sha1 string `json:"sha1"`
				Size int    `json:"size"`
				URL  string `json:"url"`
			} `json:"file"`
			Type string `json:"type"`
		} `json:"client"`
	} `json:"logging"`
	MainClass              string    `json:"mainClass"`
	MinimumLauncherVersion int       `json:"minimumLauncherVersion"`
	ReleaseTime            time.Time `json:"releaseTime"`
	Time                   time.Time `json:"time"`
	Type                   string    `json:"type"`
}

func FetchLauncherManifest(ctx context.Context) (LauncherManifest, error) {
	var lm LauncherManifest

	resp, err := Get(ctx, LAUNCHER_MANIFEST_URL)
	if err != nil {
		return lm, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return lm, err
	}

	err = json.Unmarshal(body, &lm)
	return lm, err
}

func (lm LauncherManifest) LatestRelease() (Version, error) {
	versionID := lm.Latest.Release
	if versionID == "" {
		return Version{}, fmt.Errorf("no release version found in latest")
	}

	for _, v := range lm.Versions {
		if v.ID == versionID && v.Type == "release" {
			return v, nil
		}
	}

	return Version{}, fmt.Errorf("version details for version %q not found", versionID)
}

func (lm LauncherManifest) GetLatestReleaseManifest(ctx context.Context) (VersionManifest, error) {
	var vm VersionManifest
	version, err := lm.LatestRelease()
	if err != nil {
		return vm, err
	}

	resp, err := Get(ctx, version.URL)
	if err != nil {
		return vm, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return vm, err
	}

	err = json.Unmarshal(body, &vm)
	return vm, err
}

func (vm VersionManifest) ClientURL() (string, error) {
	if vm.Downloads.Client.URL == "" {
		return "", errors.New("no client URL found")
	}
	return vm.Downloads.Client.URL, nil
}

func GetLatestJavaReleaseClientURL(ctx context.Context) (string, error) {
	launcherManifest, err := FetchLauncherManifest(ctx)
	if err != nil {
		return "", err
	}

	versionManifest, err := launcherManifest.GetLatestReleaseManifest(ctx)
	if err != nil {
		return "", err
	}

	return versionManifest.ClientURL()
}

func Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func GetString(ctx context.Context, url string) (string, error) {
	res, err := Get(ctx, url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// https://piston-data.mojang.com/v1/objects/fd19469fed4a4b4c15b2d5133985f0e3e7816a8a/client.jar
var urlRegexp = regexp.MustCompile(`.*/objects/([0-9a-f]+)/client[.]jar`)

// LatestJavaReleaseClientZip returns a *zip.ReadCloser representing
// the latest Java client JAR.
func LatestJavaReleaseClientZip(ctx context.Context) (*zip.ReadCloser, error) {
	clientURL, err := GetLatestJavaReleaseClientURL(ctx)
	if err != nil {
		return nil, err
	}

	match := urlRegexp.FindStringSubmatch(clientURL)
	if match == nil {
		return nil, fmt.Errorf(`unrecognized URL shape (want ".../objects/$SHASUM/client.jar"): %q`, clientURL)
	}

	sha := match[1]

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	filename := filepath.Join(cacheDir, "bedrockprune", "java-"+sha+".jar")

	exists, err := FileExists(filename)
	if err != nil {
		return nil, err
	}

	if exists {
		gotSHA, err := shaSumFile(filename)
		if err != nil {
			return nil, fmt.Errorf("error reading SHA from existing file %q: %w", filename, err)
		}
		if gotSHA == sha {
			return zip.OpenReader(filename)
		}

		if err := os.Remove(filename); err != nil {
			return nil, fmt.Errorf("error removing file %q with wrong sha %q (want %q): %w", filename, gotSHA, sha, err)
		}
	} else {
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, err
		}
	}

	out, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	resp, err := Get(ctx, clientURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status downloading %q: %s", clientURL, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}

	gotSHA, err := shaSumFile(filename)
	if err != nil {
		return nil, err
	}

	if sha != gotSHA {
		return nil, fmt.Errorf("expected file %q from URL %q to have shasum %q; got %q", filename, clientURL, sha, gotSHA)
	}

	return zip.OpenReader(filename)
}

func shaSumFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	gotSha := fmt.Sprintf("%x", h.Sum(nil))
	return gotSha, nil
}

// FileExists checks if a file exists (and it is not a directory).
func FileExists(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err == nil {
		return !info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func GetLatestBedrockReleaseURL(ctx context.Context) (string, error) {
	dlURL := BEDROCK_DOWNLOAD_PAGE_URL
	html, err := GetString(ctx, dlURL)
	if err != nil {
		return "", err
	}
	doc := soup.HTMLParse(html)

	downloadButtonsDiv := doc.Find("div", "class", "dwbuttonslist")
	if downloadButtonsDiv.Error != nil {
		return "", fmt.Errorf("error parsing HTML from %q: %w", dlURL, downloadButtonsDiv.Error)
	}
	divs := downloadButtonsDiv.Children()

	var href string
	for _, div := range divs {
		if div.NodeValue != "div" {
			continue
		}
		if !strings.Contains(div.HTML(), ": Release)") {
			continue
		}

		button := div.Find("a", "class", "button")
		if button.Error != nil {
			return "", fmt.Errorf("error parsing HTML from %q: %w", dlURL, button.Error)
		}

		href = button.Attrs()["href"]
	}

	if href == "" {
		return "", fmt.Errorf(`unable to find ": Release)" in dwbuttonslist div from %q`, dlURL)
	}

	base, _ := url.Parse(BEDROCK_DOWNLOAD_PAGE_URL)
	relativeURL, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	assetPageURL := base.ResolveReference(relativeURL)
	html, err = GetString(ctx, assetPageURL.String())
	if err != nil {
		return "", err
	}

	// time_wrap.innerHTML = "<a href='https://mcpedl.org/uploads_files/15-02-2024/minecraft-1-20-62.apk'
	assetRegexp := regexp.MustCompile(`<a href='(https://mcpedl.org/uploads_files/[^']+)'[^>]*>Download</a>`)

	match := assetRegexp.FindStringSubmatch(html)
	if match == nil {
		return "", fmt.Errorf("unable to find %q in HTML from %s", assetRegexp, assetPageURL)
	}

	return match[1], nil
}

// LatestBedrockReleaseClientZip returns a *zip.ReadCloser representing
// the latest Bedrock APK.
func LatestBedrockReleaseClientZip(ctx context.Context) (*zip.ReadCloser, error) {
	clientURL, err := GetLatestBedrockReleaseURL(ctx)
	if err != nil {
		return nil, err
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	filename := filepath.Join(cacheDir, "bedrockprune", "bedrock-"+path.Base(clientURL))

	exists, err := FileExists(filename)
	if err != nil {
		return nil, err
	}

	if exists {
		return zip.OpenReader(filename)
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}

	out, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	resp, err := Get(ctx, clientURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status downloading %q: %s", clientURL, resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}

	return zip.OpenReader(filename)
}

func LatestCachedZip(prefix, suffix string) (*zip.ReadCloser, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	myCacheDir := filepath.Join(cacheDir, "bedrockprune")

	files, err := os.ReadDir(myCacheDir)
	if err != nil {
		return nil, err
	}

	var newestTime time.Time
	var newestName string

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasPrefix(file.Name(), prefix) {
			continue
		}
		if !strings.HasSuffix(file.Name(), suffix) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			return nil, err
		}
		modTime := info.ModTime()
		if modTime.After(newestTime) {
			newestTime = modTime
			newestName = file.Name()
		}
	}

	if newestName == "" {
		return nil, fs.ErrNotExist
	}

	return zip.OpenReader(filepath.Join(myCacheDir, newestName))
}
