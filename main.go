package main

// Use pflag instead of flag
import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kamermans/github-skyline/pkg/skyline"
	flag "github.com/spf13/pflag"
)

const (
	version = "1.0.0"
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
	font              string
	openscadPath      string
	showVersion       bool
	showVersionRaw    bool

	aspectRatioInts [2]int
	outputFileType  skyline.OutputType
)

func init() {
	flag.StringVarP(&username, "username", "u", os.Getenv("GITHUB_USERNAME"), "GitHub username")
	flag.StringVarP(&token, "token", "t", os.Getenv("GITHUB_TOKEN"), "GitHub token")
	flag.BoolVarP(&saveContribs, "save", "s", false, "Save contributions to a file")
	flag.StringVarP(&contribsFile, "contributions", "f", "contributions.json", "File to save/load contributions")
	flag.StringVarP(&outputFile, "output", "o", "skyline.scad", "Output file (.scad and .stl are supported, but stl requires 'openscad')")
	flag.IntVarP(&startYear, "start", "b", 0, "Start year")
	flag.IntVarP(&endYear, "end", "e", 0, "End year")
	flag.StringVarP(&aspectRatio, "aspect-ratio", "a", "16:4", "Aspect ratio of the skyline")
	flag.Float64VarP(&baseAngle, "base-angle", "A", 22.5, "Slope of the base walls in degrees")
	flag.Float64VarP(&baseHeight, "base-height", "h", 5.0, "Height of the base (mm)")
	flag.Float64VarP(&baseMargin, "base-margin", "g", 1.0, "Distance from the buildings to the base walls (mm)")
	flag.Float64VarP(&maxBuildingHeight, "max-building-height", "m", 20.0, "Max building height (mm)")
	flag.Float64VarP(&buildingWidth, "building-width", "w", 2.0, "Building width (mm)")
	flag.Float64VarP(&buildingLength, "building-length", "l", 2.0, "Building length (mm)")
	flag.StringVarP(&interval, "interval", "i", "week", "Interval to use for contributions (day, week)")
	flag.StringVarP(&font, "font", "F", "Liberation Sans:style=Bold", "Font to use for text")
	flag.StringVarP(&openscadPath, "openscad", "O", "openscad", "Path to the OpenSCAD executable")
	flag.BoolVarP(&showVersion, "version", "V", false, "Show version")
	flag.BoolVar(&showVersionRaw, "version-raw", false, "Show version (raw)")
	flag.Parse()

	if showVersion {
		fmt.Printf("github-skyline %s\n", version)
		os.Exit(0)
	}

	if showVersionRaw {
		fmt.Printf("%s\n", version)
		os.Exit(0)
	}

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

	if outputFile == "" && !saveContribs {
		panic("output file is required unless you are using --save")
	}

	parts := strings.Split(path.Base(outputFile), ".")
	if len(parts) < 2 {
		panic("output file must have an extension")
	}

	outputFileType = skyline.OutputType(parts[len(parts)-1])
	if outputFileType != skyline.OutputTypeSCAD && outputFileType != skyline.OutputTypeSTL {
		panic("output file must be .scad or .stl")
	}
}

func main() {

	var err error
	var contribs *skyline.Contributions

	if contribsFile != "" && !saveContribs {
		contribs, err = skyline.NewContributionsFromFile(contribsFile)
		if err != nil {
			panic(err)
		}
	} else {
		fetcher := skyline.NewGitHubContributionsFetcher(username, token)
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
	sg := skyline.NewSkylineGenerator(*contribs, aspectRatioInts, maxBuildingHeight, buildingWidth, buildingLength, font)
	sl := sg.Generate(interval)
	sl.BaseAngle = baseAngle
	sl.BaseHeight = baseHeight
	sl.BaseMargin = baseMargin

	if outputFileType == skyline.OutputTypeSCAD {
		dur, err := sl.ToOpenSCAD(outputFile)
		if err != nil {
			panic(err)
		}

		fmt.Printf("OpenSCAD file %s generated in %v\n", outputFile, dur)

	} else if outputFileType == skyline.OutputTypeSTL {
		fmt.Printf("Generating STL ...\n")

		dur, err := sl.ToSTL(outputFile, openscadPath)
		if err != nil {
			panic(err)
		}

		fmt.Printf("STL file written to %s in %v\n", outputFile, dur)
	}
}
