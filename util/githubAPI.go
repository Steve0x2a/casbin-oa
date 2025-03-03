// Copyright 2021 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
	"github.com/casbin/casbin-oa/proxy"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

func GetClient() *github.Client {
	accessToken := beego.AppConfig.String("githubAccessToken")
	proxy.InitHttpClient()
	if len(accessToken) == 0 {
		return github.NewClient(proxy.ProxyHttpClient)
	} else {
		return github.NewClient(GetHttpClient())
	}
}

func GetHttpClient() *http.Client {
	accessToken := beego.AppConfig.String("githubAccessToken")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	c := context.WithValue(context.Background(), oauth2.HTTPClient, proxy.ProxyHttpClient)
	return oauth2.NewClient(c, ts)
}

func SetIssueLabel(owner string, repo string, number int, label string) bool {
	client := GetClient()
	issueService := client.Issues
	_, response, err := issueService.AddLabelsToIssue(context.Background(), owner, repo, number, []string{label})

	if err != nil {
		return false
	}

	if response.StatusCode == 200 {
		return true
	}
	return false
}

func SetIssueAssignee(owner string, repo string, number int, assignee string) bool {
	client := GetClient()
	issueService := client.Issues
	_, response, err := issueService.AddAssignees(context.Background(), owner, repo, number, []string{assignee})

	if err != nil {
		return false
	}

	if response.StatusCode == 200 {
		return true
	}
	return false
}

func AddIssueToProjectCard(cardId int64, issueId int64) bool {
	client := GetClient()
	cardOption := github.ProjectCardOptions{ContentType: "Issue", ContentID: issueId}
	projects := client.Projects
	_, response, err := projects.CreateProjectCard(context.Background(), cardId, &cardOption)
	if err != nil {
		return false
	}

	if response.StatusCode == 200 {
		return true
	}
	return false
}

func Comment(commentStr string, org string, repo string, number int) bool {
	client := GetClient()
	issues := client.Issues

	comment := github.IssueComment{Body: &commentStr}
	_, response, _ := issues.CreateComment(context.Background(), org, repo, number, &comment)
	return response.StatusCode == 201
}

func AtPeople(people []string, org string, repo string, number int) bool {
	var commentStr string
	for i := range people {
		commentStr = fmt.Sprintf("%s @%s", commentStr, people[i])
	}

	return Comment(commentStr, org, repo, number)
}

func GetIssueLabel(title string, content string) string {
	title = strings.ToLower(title)
	content = strings.ToLower(content)

	bugWords := []string{"bug", "wrong", "error", "broken", "failed", "disable"}
	for i := range bugWords {
		if strings.Contains(title, bugWords[i]) {
			return "bug"
		}
	}

	enhancementWords := []string{"make", "implement", "support", "update", "add", "allow", "enable", "design", "use", "extract"}
	for i := range enhancementWords {
		if strings.Contains(title, enhancementWords[i]) {
			return "enhancement"
		}
	}

	questionWords := []string{"?", "what", "how", "why"}
	for i := range questionWords {
		if strings.Contains(title, questionWords[i]) || strings.Contains(content, questionWords[i]) {
			return "question"
		}
	}

	return "question"
}

func GetProjectColumns(projectId int64) []*github.ProjectColumn {
	client := GetClient()
	projects := client.Projects
	columns, _, err := projects.ListProjectColumns(context.Background(), projectId, nil)
	if err != nil {
		panic(err)
	}

	return columns
}

func GetUserByUsername(githubUsername string) *github.User {
	client := GetClient()
	users := client.Users

	user, response, err := users.Get(context.Background(), githubUsername)
	if err != nil {
		panic(err)
	}

	if response.StatusCode == 404 {
		return nil
	}

	return user
}

func RequestReviewers(owner string, repo string, number int, reviewerNames []string) bool {
	client := GetClient()
	pullRequests := client.PullRequests

	reviewers := github.ReviewersRequest{Reviewers: reviewerNames}
	_, response, err := pullRequests.RequestReviewers(context.Background(), owner, repo, number, reviewers)
	if err != nil {
		panic(err)
	}

	return response.StatusCode == 201
}

func GetOrgMembers(org string) map[string]bool {
	client := GetClient()

	curPage := 1
	membersMap := make(map[string]bool)
	options := github.ListMembersOptions{ListOptions: github.ListOptions{Page: curPage, PerPage: 100}}
	lists, _, err := client.Organizations.ListMembers(context.Background(), org, &options)

	for {
		if err != nil {
			panic(err)
		}
		if len(lists) != 0 {
			for i := range lists {
				membersMap[lists[i].GetLogin()] = true
			}
			curPage++
			options = github.ListMembersOptions{ListOptions: github.ListOptions{Page: curPage, PerPage: 100}}
			lists, _, err = client.Organizations.ListMembers(context.Background(), org, &options)
		} else {
			break
		}
	}

	return membersMap
}
