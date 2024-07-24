package main

// Use pflag instead of flag
import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
)

var (
	username     string
	token        string
	saveContribs bool
	contribsFile string
)

func init() {
	flag.StringVarP(&username, "username", "u", os.Getenv("GITHUB_USERNAME"), "GitHub username")
	flag.StringVarP(&token, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.BoolVarP(&saveContribs, "save", "s", false, "Save contributions to a file")
	flag.StringVarP(&contribsFile, "file", "f", "contributions.json", "File to save/load contributions")
	flag.Parse()

	if username == "" || token == "" {
		flag.PrintDefaults()
		panic("username and token are required")
	}
}

func main() {

	var err error
	var contribs *Contributions

	if contribsFile != "" && !saveContribs {
		contribs, err = NewContributionsFromFile(contribsFile)
		if err != nil {
			panic(err)
		}
	} else {
		fetcher := NewGitHubContributionsFetcher(username, token)
		contribs, err = fetcher.FetchContributions(2010, 2024)
		if err != nil {
			panic(err)
		}

		if saveContribs {
			err = contribs.SaveToFile(contribsFile)
			if err != nil {
				panic(err)
			}
		}
	}

	fmt.Printf("Total contributions: %d between %v and %v\n", contribs.TotalContributions, contribs.FirstDate, contribs.LastDate)

	sg := NewSkylineGenerator(*contribs, [2]int{16, 4}, 10, 3, 2, 2)
	skyline := sg.Generate()
	skyline.ToOpenSCAD()
}
