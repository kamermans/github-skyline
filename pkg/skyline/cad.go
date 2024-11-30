package skyline

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"
	// _ "github.com/go-gl/mathgl/mgl64"
	// _ "github.com/ljanyst/ghostscad/primitive"
	// osc "github.com/ljanyst/ghostscad/sys"
)

const (
	defaultBaseMargin = 1.0
	defaultBaseHeight = 5.0
	defaultBaseAngle  = 22.5

	OutputTypeSCAD = OutputType("scad")
	OutputTypeSTL  = OutputType("stl")
)

type OutputType string

type SkylineGenerator struct {
	contributions  Contributions
	aspectRatio    float64
	maxHeight      float64
	buildingWidth  float64
	buildingLength float64
	font           string
}

type Building struct {
	*BoundingBox
	Col   int
	Row   int
	Count int
	Date  string
}

type BoundingBox struct {
	MinX   float64
	MinY   float64
	MaxX   float64
	MaxY   float64
	Length float64
	Width  float64
	Height float64
}

type Skyline struct {
	BuildingMatrix    [][]*Building
	Buildings         []Building
	BuildingWidth     float64
	BuildingLength    float64
	MaxBuildingHeight float64
	MaxContributions  int
	Bounds            BoundingBox
	BaseMargin        float64
	BaseHeight        float64
	BaseAngle         float64
	Font              string
	TextLeft          string
	TextRight         string
}

// NewSkylineGenerator creates a new SkylineGenerator
// Contributions: the contributions to use
// aspectRatio: the aspect ratio of the skyline, like [2]int[16, 9] for a 16:9 aspect ratio
// maxHeight: the maximum height of the buildings in mm
// buildingWidth: the width of the buildings in mm
// buildingLength: the length of the buildings in mm
func NewSkylineGenerator(Contributions Contributions, aspectRatio [2]int, maxHeight, buildingWidth, buildingLength float64, font string) *SkylineGenerator {
	sg := &SkylineGenerator{
		contributions:  Contributions,
		aspectRatio:    float64(aspectRatio[0]) / float64(aspectRatio[1]),
		maxHeight:      maxHeight,
		buildingWidth:  buildingWidth,
		buildingLength: buildingLength,
		font:           font,
	}

	return sg
}

func (sg *SkylineGenerator) Generate(interval string) *Skyline {
	matrix, contribs := sg.computeMatrix(interval)
	buildings := []Building{}
	for _, col := range matrix {
		for _, b := range col {
			if b != nil {
				buildings = append(buildings, *b)
			}
		}
	}

	skyline := &Skyline{
		BuildingMatrix:    matrix,
		Buildings:         buildings,
		BuildingWidth:     sg.buildingWidth,
		BuildingLength:    sg.buildingLength,
		MaxBuildingHeight: sg.maxHeight,
		MaxContributions:  contribs.Max(),
		Bounds: BoundingBox{
			MinX:   0,
			MinY:   0,
			MaxX:   float64(len(matrix)) * sg.buildingWidth,
			MaxY:   float64(len(matrix[0])) * sg.buildingLength,
			Length: float64(len(matrix[0])) * sg.buildingLength,
			Width:  float64(len(matrix)) * sg.buildingWidth, // TODO: reduce width
			Height: sg.maxHeight,
		},
		BaseMargin: defaultBaseMargin,
		BaseHeight: defaultBaseHeight,
		BaseAngle:  defaultBaseAngle,
		Font:       sg.font,
		TextLeft:   "@" + sg.contributions.Username,
		TextRight:  sg.contributions.YearRangeText(),
	}

	return skyline
}

func (sg *SkylineGenerator) computeMatrix(interval string) ([][]*Building, StatsCollection) {
	// Calculate the number of rows and columns based on the aspect ratio
	// of the skyline and the number of contributions
	var contribs StatsCollection

	switch interval {
	case "day":
		contribs = sg.contributions.PerDay()
	case "week":
		contribs = sg.contributions.PerWeek()
	default:
		panic(fmt.Errorf("invalid interval: %s; must be day or week", interval))
	}

	numBuildings := float64(len(contribs))

	cols := int(math.Ceil(math.Sqrt(numBuildings * sg.aspectRatio)))
	rows := int(math.Ceil(numBuildings / float64(cols)))

	// Remove any unused columns
	if cols*rows > int(numBuildings) {
		cols = int(math.Ceil(numBuildings / float64(rows)))
	}

	fmt.Printf("Skyline details:\n")
	fmt.Printf("  Buildings: %d (%v x %v matrix)\n", len(contribs), cols, rows)
	fmt.Printf("  Dimensions: %0.1fmm x %0.1fmm\n", float64(cols)*sg.buildingWidth, float64(rows)*sg.buildingLength)

	matrix := make([][]*Building, cols)
	for col := range matrix {
		matrix[col] = make([]*Building, rows)
	}

	maxContributions := 0
	for _, contrib := range contribs {
		if contrib.Count > maxContributions {
			maxContributions = contrib.Count
		}
	}

	// Populate the matrix with buildings
	i := 0
	for col := range matrix {
		for row := range matrix[col] {
			if i >= len(contribs) {
				break
			}

			contrib := contribs[i]
			building := &Building{
				BoundingBox: &BoundingBox{
					MinX:   float64(col) * sg.buildingWidth,
					MinY:   float64(row) * sg.buildingLength,
					MaxX:   float64(col+1) * sg.buildingWidth,
					MaxY:   float64(row+1) * sg.buildingLength,
					Length: sg.buildingLength,
					Width:  sg.buildingWidth,
					Height: float64(contrib.Count) / float64(maxContributions) * sg.maxHeight,
				},
				Col:   col,
				Row:   row,
				Count: contrib.Count,
				Date:  contrib.Date,
			}

			matrix[col][row] = building
			i++
		}
	}

	return matrix, contribs
}

var (
	baseModule = `module base() {
    bottomWidth = baseWidth + 2 * baseOffset;
    bottomLength = baseLength + 2 * baseOffset;

    points = [
        // Bottom
        [0, 0, 0],
        [bottomWidth, 0, 0],
        [bottomWidth, bottomLength, 0],
        [0, bottomLength, 0],
        // Top
        [baseOffset, baseOffset, baseHeight],
        [baseWidth+baseOffset, baseOffset, baseHeight],
        [baseWidth+baseOffset, baseLength+baseOffset, baseHeight],
        [baseOffset, baseLength+baseOffset, baseHeight],
    ];

    faces = [
        [0,1,2,3],  // Bottom
        [4,5,1,0],  // Front
        [7,6,5,4],  // Top
        [5,6,2,1],  // Right
        [6,7,3,2],  // Back
        [7,4,0,3],  // Left
    ];

    color(baseColor)
    polyhedron(points, faces);

	if (textEnable) {
		textOffset = baseOffset+baseMargin;
		textSize = baseHeight-baseMargin;

        rotate([90-baseAngle, 0 ,0])
        translate([textOffset, 1, 0])
        color("red")
        linear_extrude(textHeight)
        text(textLeft, size=textSize, halign="left", valign="baseline", font=textFont)
        ;

        rotate([90-baseAngle, 0 ,0])
        translate([bottomWidth-textOffset, 1, 0])
        color("red")
        linear_extrude(textHeight)
        text(textRight, size=textSize, halign="right", valign="baseline", font=textFont);
    }
}`

	buildingModule = `module building(row, col, contributions) {
    height = contributions / maxContributions * maxBuildingHeight;
    color(buildingColor)
        translate([
            (col * buildingWidth)+baseMargin+baseOffset,
            (row * buildingLength)+baseMargin+baseOffset, baseHeight
        ])
        cube([buildingWidth, buildingLength, height]);
}`
)

func (sl *Skyline) ToOpenSCAD(filename string) (time.Duration, error) {
	start := time.Now()
	out := &bytes.Buffer{}

	// Variables
	fmt.Fprintf(out, "// GitHub Skyline Generator\n")
	fmt.Fprintf(out, "// by Steve Kamerman\n")
	fmt.Fprintf(out, "// https://github.com/kamermans/github-skyline\n\n")

	fmt.Fprintf(out, "// Base Parameters\n")
	fmt.Fprintf(out, "baseMargin = %f;\n", sl.BaseMargin)
	fmt.Fprintf(out, "baseAngle = %f;\n", sl.BaseAngle)
	fmt.Fprintf(out, "baseHeight = %f;\n", sl.BaseHeight)
	fmt.Fprintf(out, "baseWidth = %f + (2 * baseMargin);\n", sl.Bounds.Width)
	fmt.Fprintf(out, "baseLength = %f + (2 * baseMargin);\n", sl.Bounds.Length)
	fmt.Fprintf(out, "baseOffset = baseHeight * tan(baseAngle);\n")
	fmt.Fprintf(out, `baseColor = "cyan";`+"\n")

	fmt.Fprintf(out, "\n// Base Text\n")
	fmt.Fprintf(out, "textEnable = true;\n")
	fmt.Fprintf(out, "textFont = %q;\n", sl.Font)
	fmt.Fprintf(out, "textLeft = %q;\n", sl.TextLeft)
	fmt.Fprintf(out, "textRight = %q;\n", sl.TextRight)
	fmt.Fprintf(out, `textColor = "red";`+"\n")
	fmt.Fprintf(out, "textHeight = 0.4;\n")

	fmt.Fprintf(out, "\n// Building Parameters\n")
	fmt.Fprintf(out, "buildingWidth = %f;\n", sl.BuildingWidth)
	fmt.Fprintf(out, "buildingLength = %f;\n", sl.BuildingLength)
	fmt.Fprintf(out, "maxBuildingHeight = %f;\n", sl.MaxBuildingHeight)
	fmt.Fprintf(out, `buildingColor = "red";`+"\n")

	fmt.Fprintf(out, "\n// GitHub Parameters\n")
	fmt.Fprintf(out, "maxContributions = %d;\n", sl.MaxContributions)

	fmt.Fprintln(out)

	fmt.Fprintf(out, "%v\n\n", baseModule)
	fmt.Fprintf(out, "%v\n\n", buildingModule)

	fmt.Fprintf(out, "union() {\n")
	fmt.Fprintf(out, "  base();\n")
	fmt.Fprintf(out, "  // building(row, col, contributions);\n")

	for _, b := range sl.Buildings {
		if b.Count == 0 {
			continue
		}

		fmt.Fprintf(out, "  building(%d, %d, %d); // %v\n",
			b.Row, b.Col, b.Count, b.Date)
	}

	fmt.Fprintf(out, "}\n") // end union

	err := os.WriteFile(filename, out.Bytes(), 0644)
	return time.Since(start), err
}

func (sl *Skyline) ToSTL(filename string, openscadPath string) (time.Duration, error) {
	start := time.Now()

	tmpFile, err := os.CreateTemp("", "skyline*.scad")
	if err != nil {
		return time.Since(start), err
	}

	defer os.Remove(tmpFile.Name())

	_, err = sl.ToOpenSCAD(tmpFile.Name())
	if err != nil {
		return time.Since(start), err
	}

	cmd := exec.Command(openscadPath, "-o", filename, tmpFile.Name())
	err = cmd.Run()
	if err != nil {
		return time.Since(start), err
	}

	return time.Since(start), nil
}
