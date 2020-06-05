package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"github.com/lrita/log"
	"golang.org/x/oauth2"
)

type stage int

const (
	ready stage = iota
	unkown
)

func main() {
	var (
		stage        stage
		buff         bytes.Buffer
		repositories []*github.StarredRepository
		opt          = &github.ActivityListStarredOptions{}
		fullname2m   = make(map[string]*github.Repository)
		lang2m       = make(map[string]bool)
	)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("X_FAVOR_GIT_OAUTH")})
	cli := github.NewClient(oauth2.NewClient(context.Background(), ts))
	opt.PerPage = 50
	for {
		repo, resp, err := cli.Activity.ListStarred(context.Background(), "", opt)
		if err != nil {
			log.Fatal(err)
		}
		repositories = append(repositories, repo...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	for _, repo := range repositories {
		r := repo.GetRepository()
		fullname2m[r.GetFullName()] = r
		lang2m[r.GetLanguage()] = true
	}

	d, err := ioutil.ReadFile("README.md")
	if err != nil {
		log.Fatalf("ioutil.ReadFile(README.md) failed: %v", err)
	}

	for _, line := range strings.Split(string(d), "\n") {
		if strings.HasPrefix(line, "# 未分类") {
			stage = unkown
		}
		if stage == ready {
			if strings.HasPrefix(line, "* [") {
				i := strings.IndexByte(line, ']')
				if i < 0 {
					log.Fatalf("bad line %v", line)
				}
				fullname := line[3:i]
				delete(fullname2m, fullname)
			}
			buff.WriteString(line)
			buff.WriteByte('\n')
		}
	}

	if len(fullname2m) != 0 {
		buff.WriteString("# 未分类\n")
		lang2ms := make(map[string][]*github.Repository)
		for _, repo := range fullname2m {
			lang := repo.GetLanguage()
			lang2ms[lang] = append(lang2ms[lang], repo)
		}
		langarray := make([]string, 0, len(lang2ms))
		for lang := range lang2ms {
			langarray = append(langarray, lang)
		}
		sort.Strings(langarray)
		for _, lang := range langarray {
			repoarray := lang2ms[lang]
			sort.Slice(repoarray, func(i, j int) bool {
				return repoarray[i].GetFullName() < repoarray[j].GetFullName()
			})
			fmt.Fprintf(&buff, "\n## %s\n", lang)
			for _, repo := range repoarray {
				fmt.Fprintf(&buff, "* [%s](%s) %s\n",
					repo.GetFullName(), repo.GetHTMLURL(), repo.GetDescription())
			}
		}
	}

	if err := ioutil.WriteFile("README.md", buff.Bytes(), 0644); err != nil {
		log.Fatalf("ioutil.WriteFile(README.md) failed: %v", err)
	}
}
