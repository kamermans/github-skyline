package skyline

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

type StatsCollection []Stats

func (sc StatsCollection) Max() int {
	max := 0
	for _, s := range sc {
		if s.Count > max {
			max = s.Count
		}
	}

	return max
}

type Contributions struct {
	Username           string         `json:"username"`
	TotalContributions int            `json:"total_contributions"`
	FirstDate          string         `json:"first_date"`
	LastDate           string         `json:"last_date"`
	ByDate             map[string]int `json:"by_date"`
}

// TrimStartYear trims the contributions to the first year with at least one contribution
func (c *Contributions) TrimStartYear() bool {
	firstContributionYear := 0
	for date, numContribs := range c.ByDate {

		if numContribs == 0 {
			continue
		}

		year, err := time.Parse("2006-01-02", date)
		if err != nil {
			panic(err)
		}

		if firstContributionYear == 0 || year.Year() < firstContributionYear {
			firstContributionYear = year.Year()
		}
	}

	fmt.Printf("First contribution year: %d\n", firstContributionYear)

	if firstContributionYear == 0 {
		return false
	}

	for date := range c.ByDate {
		year, err := time.Parse("2006-01-02", date)
		if err != nil {
			panic(err)
		}

		if year.Year() < firstContributionYear {
			delete(c.ByDate, date)
		}
	}

	firstDate := fmt.Sprintf("%d-01-01", firstContributionYear)
	if firstDate != c.FirstDate {
		c.FirstDate = firstDate
		return true
	}

	return false
}

func (c *Contributions) YearRangeText() string {
	startYear := c.FirstDate[:4]
	endYear := c.LastDate[:4]

	if startYear == endYear {
		return startYear
	}

	return fmt.Sprintf("%s-%s", startYear, endYear)
}

func (c *Contributions) PerDay() StatsCollection {
	dayKeys := make([]string, 0, len(c.ByDate))
	for key := range c.ByDate {
		dayKeys = append(dayKeys, key)
	}

	// Sort the days
	sort.Strings(dayKeys)

	days := make(StatsCollection, 0, len(c.ByDate))
	for _, date := range dayKeys {
		days = append(days, Stats{
			Date:  date,
			Count: c.ByDate[date],
		})
	}

	return days
}

func (c *Contributions) PerWeek() StatsCollection {
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

	weekStats := make(StatsCollection, 0, len(weeks))
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
		Username: gcf.username,
		ByDate:   make(map[string]int),
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
