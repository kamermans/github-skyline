package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/hasura/go-graphql-client"
)

const (
	githubAPIURL = "https://api.github.com/graphql"
)

type Stats struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Contributions struct {
	TotalContributions int            `json:"total_contributions"`
	FirstDate          string         `json:"first_date"`
	LastDate           string         `json:"last_date"`
	ByDate             map[string]int `json:"by_date"`
}

func (c *Contributions) PerDay() []Stats {
	dayKeys := make([]string, 0, len(c.ByDate))
	for key := range c.ByDate {
		dayKeys = append(dayKeys, key)
	}

	// Sort the days
	sort.Strings(dayKeys)

	days := make([]Stats, 0, len(c.ByDate))
	for _, date := range dayKeys {
		days = append(days, Stats{
			Date:  date,
			Count: c.ByDate[date],
		})
	}

	return days
}

func (c *Contributions) PerWeek() []Stats {
	weeks := make(map[string]int)

	for date, count := range c.ByDate {
		// Compute week of the year as an integer
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			panic(err)
		}

		year, week := t.ISOWeek()
		key := fmt.Sprintf("%d-%02d", year, week)
		weeks[key] += count
	}

	weekKeys := make([]string, 0, len(weeks))
	for key := range weeks {
		weekKeys = append(weekKeys, key)
	}

	// Sort the weeks
	sort.Strings(weekKeys)

	weekStats := make([]Stats, 0, len(weeks))
	for _, week := range weekKeys {
		weekStats = append(weekStats, Stats{
			Date:  week,
			Count: weeks[week],
		})
	}

	return weekStats
}

func (c *Contributions) SaveToFile(file string) error {
	fh, err := os.Create(file)
	if err != nil {
		return err
	}

	defer fh.Close()

	return json.NewEncoder(fh).Encode(c)
}

func NewContributionsFromFile(file string) (*Contributions, error) {
	fh, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer fh.Close()

	contribs := &Contributions{}
	err = json.NewDecoder(fh).Decode(contribs)
	if err != nil {
		return nil, err
	}

	return contribs, nil
}

type headerRoundTripper struct {
	headers      map[string]string
	roundTripper http.RoundTripper
}

func (hrt *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range hrt.headers {
		req.Header.Set(key, value)
	}
	return hrt.roundTripper.RoundTrip(req)
}

func newClientWithHeaders(headers map[string]string) *http.Client {
	return &http.Client{
		Transport: &headerRoundTripper{
			headers:      headers,
			roundTripper: http.DefaultTransport,
		},
	}
}

func NewGraphQLClient(token string) *graphql.Client {
	httpClient := newClientWithHeaders(map[string]string{
		"Authorization": "Bearer " + token,
		"User-Agent":    "github-skyline",
	})

	return graphql.NewClient(githubAPIURL, httpClient).WithDebug(true)
}

type GitHubContributionsFetcher struct {
	client   *graphql.Client
	username string
}

func NewGitHubContributionsFetcher(username string, token string) *GitHubContributionsFetcher {
	return &GitHubContributionsFetcher{
		client:   NewGraphQLClient(token),
		username: username,
	}
}

type DateTime struct{ time.Time }

func (gcf *GitHubContributionsFetcher) FetchContributions(startYear, endYear int) (*Contributions, error) {

	contrib := &Contributions{
		ByDate: make(map[string]int),
	}

	var firstDate time.Time
	var lastDate time.Time

	thisYear := startYear
	for thisYear <= endYear {
		var query struct {
			User struct {
				ContributionsCollection struct {
					ContributionCalendar struct {
						TotalContributions graphql.Int
						Weeks              []struct {
							ContributionDays []struct {
								ContributionCount graphql.Int
								Date              graphql.String
							}
						}
					}
				} `graphql:"contributionsCollection(from: $start)"`
			} `graphql:"user(login: $username)"`
		}

		start := time.Date(thisYear, 1, 1, 0, 0, 0, 0, time.UTC)

		fmt.Printf("Fetching contributions from %v...", thisYear)

		var variables = map[string]any{
			"username": graphql.String(gcf.username),
			"start":    DateTime{start},
		}

		err := gcf.client.Query(context.Background(), &query, variables)
		if err != nil {
			return nil, err
		}

		fmt.Printf(" found %d\n", query.User.ContributionsCollection.ContributionCalendar.TotalContributions)

		for _, week := range query.User.ContributionsCollection.ContributionCalendar.Weeks {
			for _, day := range week.ContributionDays {
				date, err := time.Parse("2006-01-02", string(day.Date))
				if err != nil {
					return nil, err
				}

				// Skip contributions from the future
				if date.After(time.Now()) {
					continue
				}

				if firstDate.IsZero() || date.Before(firstDate) {
					firstDate = date
				}

				if lastDate.IsZero() || date.After(lastDate) {
					lastDate = date
				}

				contrib.ByDate[string(day.Date)] = int(day.ContributionCount)
			}
		}

		thisYear++
	}

	for _, count := range contrib.ByDate {
		contrib.TotalContributions += count
	}

	contrib.FirstDate = firstDate.Format("2006-01-02")
	contrib.LastDate = lastDate.Format("2006-01-02")

	return contrib, nil
}
