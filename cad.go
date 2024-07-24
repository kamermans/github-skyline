package main

import (
	"fmt"
	"math"
	"os"
	// _ "github.com/go-gl/mathgl/mgl64"
	// _ "github.com/ljanyst/ghostscad/primitive"
	// osc "github.com/ljanyst/ghostscad/sys"
)

const (
	defaultBaseMargin = 5.0
	defaultBaseHeight = 5.0
)

type SkylineGenerator struct {
	contributions  Contributions
	aspectRatio    float64
	maxHeight      float64
	stepSize       float64
	buildingWidth  float64
	buildingLength float64
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
	Buildings  [][]*Building
	Bounds     BoundingBox
	BaseMargin float64
	BaseHeight float64
}

// NewSkylineGenerator creates a new SkylineGenerator
// Contributions: the contributions to use
// aspectRatio: the aspect ratio of the skyline, like [2]int[16, 9] for a 16:9 aspect ratio
// maxHeight: the maximum height of the buildings in mm
// stepSize: the step size of the skyline in mm
// buildingWidth: the width of the buildings in mm
// buildingLength: the length of the buildings in mm
func NewSkylineGenerator(Contributions Contributions, aspectRatio [2]int, maxHeight, stepSize, buildingWidth, buildingLength float64) *SkylineGenerator {
	sg := &SkylineGenerator{
		contributions:  Contributions,
		aspectRatio:    float64(aspectRatio[0]) / float64(aspectRatio[1]),
		maxHeight:      maxHeight,
		stepSize:       stepSize,
		buildingWidth:  buildingWidth,
		buildingLength: buildingLength,
	}

	return sg
}

func (sg *SkylineGenerator) Generate() *Skyline {
	matrix := sg.computeMatrix()
	skyline := &Skyline{
		Buildings: matrix,
		Bounds: BoundingBox{
			MinX:   0,
			MinY:   0,
			MaxX:   float64(len(matrix)) * sg.buildingWidth,
			MaxY:   float64(len(matrix[0])) * sg.buildingLength,
			Length: float64(len(matrix[0])) * sg.buildingLength,
			Width:  float64(len(matrix)) * sg.buildingWidth,
			Height: sg.maxHeight,
		},
		BaseMargin: defaultBaseMargin,
		BaseHeight: defaultBaseHeight,
	}

	// fmt.Printf("Matrix:\n")
	// encoder := json.NewEncoder(os.Stdout)
	// encoder.SetIndent("", "  ")
	// encoder.Encode(matrix)
	// fmt.Println()

	return skyline
}

func (sg *SkylineGenerator) computeMatrix() [][]*Building {
	// Calculate the number of rows and columns based on the aspect ratio
	// of the skyline and the number of contributions
	contribs := sg.contributions.PerWeek()
	numBuildings := float64(len(contribs))

	// The aspectRatio is usually a number above 1
	// We want to make sure that the skyline is wider than it is tall
	// So we calculate the square root of the aspect ratio
	// and use that as the number of columns
	fmt.Printf("cols math: math.Ceil(math.Sqrt(float64(%v) * %v))\n", numBuildings, sg.aspectRatio)
	cols := int(math.Ceil(math.Sqrt(numBuildings * sg.aspectRatio)))
	rows := int(math.Ceil(numBuildings / float64(cols)))

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
			i++
			// fmt.Printf("Contrib: %v, %v, %v: %v\n", col, row, contrib.Date, contrib.Count)

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

			// fmt.Printf("%v / %v * %v, ", float64(contrib.Count), float64(maxContributions), sg.maxHeight)
			// fmt.Printf("%v,", contrib.Count)

			matrix[col][row] = building
		}
	}

	return matrix
}

var (
	baseModule = `module base() {
    color(baseColor, 1.0)
        translate([baseWidth/2, baseLength/2, 0])
            linear_extrude(height = baseHeight, scale = [baseChamferScaleX, baseChamferScaleY])
                square([baseWidth, baseLength], center = true);
}`

	buildingModule = `module building(x, y, contributions) {
    height = contributions / maxContributions * maxBuildingHeight;
    color(buildingColor)
        translate([x+baseMargin, y+baseMargin, 5.000000])
            cube([buildingWidth, buildingLength, height]);
}
`
)

func (sl *Skyline) ToOpenSCAD() ([]byte, error) {
	// osc.Initialize()

	// base := primitive.NewCube(mgl64.Vec3{sl.Bounds.Length, sl.Bounds.Width, sl.BaseHeight})

	// osc.RenderOne(primitive.NewCube(mgl64.Vec3{sl.Bounds.Length, sl.Bounds.Width, sl.BaseHeight}))
	file, err := os.Create("skyline.scad")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil, err
	}

	defer file.Close()

	maxContributions := 0
	for _, col := range sl.Buildings {
		for _, b := range col {
			if b == nil {
				continue
			}

			if b.Count > maxContributions {
				maxContributions = b.Count
			}
		}
	}

	// Variables
	fmt.Fprintf(file, "// GitHub Skyline Generator\n")
	fmt.Fprintf(file, "// by Steve Kamerman\n")
	fmt.Fprintf(file, "// https://github.com/kamermans/github-skyline\n\n")

	fmt.Fprintf(file, "// Base Parameters\n")
	fmt.Fprintf(file, "baseMargin = %f;\n", sl.BaseMargin)
	fmt.Fprintf(file, "baseChamferScaleX = 0.9;\n")
	fmt.Fprintf(file, "baseChamferScaleY = 0.8;\n")
	fmt.Fprintf(file, "baseHeight = %f;\n", sl.BaseHeight)
	fmt.Fprintf(file, "baseWidth = %f + (2 * baseMargin);\n", sl.Bounds.Width)
	fmt.Fprintf(file, "baseLength = %f + (2 * baseMargin);\n", sl.Bounds.Length)
	fmt.Fprintf(file, `baseColor = "cyan";`+"\n")

	fmt.Fprintf(file, "\n// Building Parameters\n")
	fmt.Fprintf(file, "buildingWidth = %f;\n", sl.Buildings[0][0].Width)
	fmt.Fprintf(file, "buildingLength = %f;\n", sl.Buildings[0][0].Length)
	fmt.Fprintf(file, "maxBuildingHeight = %f;\n", sl.Bounds.Height)
	fmt.Fprintf(file, `buildingColor = "red";`+"\n")

	fmt.Fprintf(file, "\n// GitHub Parameters\n")
	fmt.Fprintf(file, "maxContributions = %d;\n", maxContributions)

	fmt.Fprintln(file)

	fmt.Fprintf(file, "%v\n\n", baseModule)
	fmt.Fprintf(file, "%v\n\n", buildingModule)

	fmt.Fprintf(file, "base();\n")

	for _, col := range sl.Buildings {

		// fmt.Fprintf(file, "// Column %d (%v)\n", col[0].Col, col[0].Date)

		for _, b := range col {
			if b == nil || b.Count == 0 {
				continue
			}

			// fmt.Fprintf(file, "// %d contributions on %s\n", b.Count, b.Date)

			fmt.Fprintf(file, "building(%f, %f, %d); // %v\n",
				b.MinX, b.MinY, b.Count, b.Date)
		}
	}

	return nil, nil
}
