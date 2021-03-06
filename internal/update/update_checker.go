// update package provides a way to check for github release update
// it will parse the tag of the repository in question and compare it with the current version
// in the database, if there's update available, it will store it inside the badger database,
// and the agent will update the relevant component in question
package update

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/go-github/github"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"go.amplifyedge.org/booty-v2/internal/logging"
)

const (
	defaultTimeout = 5 * time.Second
)

// RepositoryURL is the github repository url we will monitor
type RepositoryURL string

// Version is the version information for the releases.
type Version string

func (v Version) String() string {
	return string(v)
}

type Checker struct {
	logger     logging.Logger
	repos      map[RepositoryURL]Version          // repository urls
	updateFunc func(RepositoryURL, Version) error // call this function on new release
}

func NewChecker(logger logging.Logger, repos map[RepositoryURL]Version, updateFunc func(r RepositoryURL, v Version) error) *Checker {
	return &Checker{
		repos:      repos,
		logger:     logger,
		updateFunc: updateFunc,
	}
}

func (c *Checker) getRepoInfos() (chan *repoInfo, error) {
	rchan := make(chan *repoInfo, len(c.repos))
	for r, v := range c.repos {
		rinfo, err := parseGithubUrl(r, v)
		if err != nil {
			c.logger.Errorf("error parsing github url for %s: %v", r, err)
			return nil, err
		}
		rchan <- rinfo
	}
	close(rchan)
	return rchan, nil
}

func (c *Checker) CheckNewReleases() error {
	ghc := github.NewClient(nil)
	rinfos, err := c.getRepoInfos()
	if err != nil {
		return err
	}
	for k := range rinfos {
		if err = c.fetchLatest(k, ghc); err != nil {
			return err
		}
	}
	return nil
}

func (c *Checker) fetchLatest(r *repoInfo, ghc *github.Client) error {
	c.logger.Infof("checking update for: %s", r.repoUrl)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var v, latest string
	release, _, err := ghc.Repositories.GetLatestRelease(ctx, r.repoUser, r.repoName)
	if err != nil {
		// scrape it if it doesn't work
		latest, err = FallbackScrape(r.repoUrl)
		if err != nil {
			return err
		}
		v = versionNumber(latest)
	}
	if release != nil {
		v = versionNumber(*release.TagName)
	}
	if isTagNewer(r.curVer.toSemver(), parseReleaseTag(v)) {
		c.logger.Infof("latest version is newer: %s than current: %s, updating...", Version(v), r.curVer)
		// assign it to the repos
		c.repos[r.repoUrl] = Version(v)
		// do the update function
		if err = c.updateFunc(r.repoUrl, Version(v)); err != nil {
			return err
		}
	}
	return nil
}

func GetLatestVersion(repoUrl RepositoryURL) (string, error) {
	ghc := github.NewClient(nil)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	info, err := parseGithubUrl(repoUrl, "")
	if err != nil {
		return "", err
	}
	var v string
	release, _, err := ghc.Repositories.GetLatestRelease(ctx, info.repoUser, info.repoName)
	if err != nil {
		var latest string
		latest, err = FallbackScrape(repoUrl)
		if err != nil {
			return "", err
		}
		v = versionNumber(latest)
	}
	if release != nil {
		v = *release.TagName
	}
	return versionNumber(v), nil
}

func versionNumber(v string) string {
	v = filepath.Base(v)
	if v[0] == 'v' || v[0] == 'V' {
		return v[1:]
	}
	return v
}

func FallbackScrape(repoUrl RepositoryURL) (string, error) {
	// scrape it if it doesn't work
	htc := http.Client{Timeout: defaultTimeout}
	releaseUrl := string(repoUrl) + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, releaseUrl, nil)
	if err != nil {
		return "", err
	}
	resp, err := htc.Do(req)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	latestTag := doc.Find(".d-none.d-md-block.mt-2.list-style-none").Children().First().Find("a").First().Text()
	if latestTag == "" {
		return "", fmt.Errorf("releases not found")
	}
	return strings.TrimSpace(latestTag), nil
}
