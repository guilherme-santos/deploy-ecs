package cobra

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	update "github.com/inconshreveable/go-update"
	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
)

func NewSelfUpdateCommand(cmd *Command) {
	cobraCmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update this tool for the lastest version",
	}

	cobraCmd.Run = func(cobraCmd *cobra.Command, args []string) {
		token := cmd.Config.GitHub.Token
		operationalSystem := runtime.GOOS

		fmt.Println("Your version is:", cmd.Version)
		fmt.Printf("Looking for the latest version available to '%s'...\n", operationalSystem)
		url, version, err := getLatestVersion(cmd.Config.GitHub.Token, operationalSystem)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		newFile, err := downloadLatestVersion(token, url)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		updatedFile := new(bytes.Buffer)
		nfReader := io.TeeReader(newFile, updatedFile)

		execpath, _ := osext.Executable()
		currentFile, _ := ioutil.ReadFile(execpath)

		checksumNewFile := calculateChecksum(nfReader)
		checksumCurrentFile := calculateChecksum(bytes.NewReader(currentFile))

		if bytes.EqualFold(checksumNewFile, checksumCurrentFile) {
			fmt.Println("You already have the latest version!")
			os.Exit(0)
		}

		fmt.Printf("New version was found %s...\n", version)

		err = update.Apply(updatedFile, update.Options{})
		if err != nil {
			if rerr := update.RollbackError(err); rerr != nil {
				fmt.Println("Failed to rollback from bad update:", rerr)
				os.Exit(1)
			}
		}

		fmt.Println("Your version was updated!")
	}

	cmd.AddCommand(cobraCmd)
}

func getLatestVersion(token, operationalSystem string) (string, string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/guilherme-santos/deploy-ecs/releases/latest", nil)
	if !strings.EqualFold("", token) {
		req.Header.Add("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("Cannot get latest release from Github API: %s", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		errMsg := "It was not possible to access our Github repository.\n"
		if strings.EqualFold("", token) {
			errMsg += "No GitHub token was informed, try: deploy-ecs config github set token <your-token>"
		} else {
			errMsg += "Verify if informed token can access the respository: https://github.com/guilherme-santos/deploy-ecs"
		}

		return "", "", errors.New(errMsg)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("We had some problem comunicating with GitHub API, see their response:\n%s", body)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"assets"`
	}

	json.Unmarshal(body, &release)

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, operationalSystem+".tar.gz") {
			return asset.URL, release.TagName, nil
		}
	}

	return "", "", errors.New("No version was found to your operational system")
}

func downloadLatestVersion(token, url string) (io.Reader, error) {
	req, _ := http.NewRequest("GET", url, nil)
	if !strings.EqualFold("", token) {
		req.Header.Add("Authorization", "token "+token)
	}
	req.Header.Add("Accept", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cannot download latest release from Github API: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("We had some problem downloading latest release, see GitHub response:\n%s", body)
	}

	archive, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot uncompress latest release: %s", err)
	}
	defer archive.Close()

	tarReader := tar.NewReader(archive)
	tarReader.Next() // Ignore header
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("Downloaded file is corrupted!")
		}

		return nil, fmt.Errorf("Downloaded file is corrupted: %s", err)
	}

	content := new(bytes.Buffer)
	io.Copy(content, tarReader)

	return content, nil
}

func calculateChecksum(content io.Reader) []byte {
	h := sha256.New()

	_, err := io.Copy(h, content)
	if err != nil {
		return nil
	}

	return h.Sum(nil)
}
