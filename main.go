package main

// Use pflag instead of flag
import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
)

var (
	username          string
	token             string
	saveContribs      bool
	contribsFile      string
	outputFile        string
	startYear         int
	endYear           int
	aspectRatio       string
	baseAngle         float64
	baseHeight        float64
	baseMargin        float64
	maxBuildingHeight float64
	buildingWidth     float64
	buildingLength    float64
	interval          string

	aspectRatioInts [2]int
	outputFileType  string
)

func init() {
	flag.StringVarP(&username, "username", "u", os.Getenv("GITHUB_USERNAME"), "GitHub username")
	flag.StringVarP(&token, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.BoolVarP(&saveContribs, "save", "s", false, "Save contributions to a file")
	flag.StringVarP(&contribsFile, "contributions", "f", "contributions.json", "File to save/load contributions")
	flag.StringVarP(&outputFile, "output", "o", "skyline.scad", "Output file (.scad and .stl are supported, but stl requires 'openscad')")
	flag.IntVarP(&startYear, "start", "b", 0, "Start year")
	flag.IntVarP(&endYear, "end", "e", 0, "End year")
	flag.StringVarP(&aspectRatio, "aspect-ratio", "a", "16:9", "Aspect ratio of the skyline")
	flag.Float64VarP(&baseAngle, "base-angle", "A", 22.5, "Slope of the base walls in degrees")
	flag.Float64VarP(&baseHeight, "base-height", "h", 5.0, "Height of the base (mm)")
	flag.Float64VarP(&baseMargin, "base-margin", "g", 1.0, "Distance from the buildings to the base walls (mm)")
	flag.Float64VarP(&maxBuildingHeight, "max-building-height", "m", 20.0, "Max building height (mm)")
	flag.Float64VarP(&buildingWidth, "building-width", "w", 2.0, "Building width (mm)")
	flag.Float64VarP(&buildingLength, "building-length", "l", 2.0, "Building length (mm)")
	flag.StringVarP(&interval, "interval", "i", "week", "Interval to use for contributions (day, week)")
	flag.Parse()

	if contribsFile == "" && !saveContribs && (username == "" || token == "") {
		flag.PrintDefaults()
		panic("username and token are required")
	}

	_, err := fmt.Sscanf(aspectRatio, "%d:%d", &aspectRatioInts[0], &aspectRatioInts[1])
	if err != nil {
		panic(fmt.Errorf("invalid aspect ratio: %s; %w", aspectRatio, err))
	}

	if interval != "day" && interval != "week" {
		panic(fmt.Errorf("invalid interval: %s; must be day or week", interval))
	}

	if outputFile == "" {
		panic("output file is required")
	}

	parts := strings.Split(path.Base(outputFile), ".")
	if len(parts) < 2 {
		panic("output file must have an extension")
	}

	outputFileType = parts[len(parts)-1]
	if outputFileType != "scad" && outputFileType != "stl" {
		panic("output file must be .scad or .stl")
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
		contribs, err = fetcher.FetchContributions(startYear, endYear)
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

	fmt.Printf("Generating OpenSCAD ...\n")
	sg := NewSkylineGenerator(*contribs, aspectRatioInts, maxBuildingHeight, buildingWidth, buildingLength)
	skyline := sg.Generate(interval)
	skyline.BaseAngle = baseAngle
	skyline.BaseHeight = baseHeight
	skyline.BaseMargin = baseMargin
	scad, err := skyline.ToOpenSCAD()
	if err != nil {
		panic(err)
	}

	if outputFileType == "scad" {
		err = os.WriteFile(outputFile, scad, 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("OpenSCAD file written to %s\n", outputFile)

	} else if outputFileType == "stl" {
		fmt.Printf("Generating STL ...\n")
		start := time.Now()

		tmpFile, err := os.CreateTemp("", "skyline*.scad")
		if err != nil {
			panic(err)
		}

		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write(scad)
		if err != nil {
			panic(err)
		}

		cmd := exec.Command("openscad", "-o", outputFile, tmpFile.Name())
		err = cmd.Run()
		if err != nil {
			panic(err)
		}

		dur := time.Since(start)
		fmt.Printf("STL file written to %s in %v\n", outputFile, dur)
	}
}
