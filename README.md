# github-skyline

This program generates GitHub Skyline CAD files in OpenSCAD and STL format.

A GitHub Skyline is a 3D representation of a user's GitHub contributions,
where each building in the skyline represents one day or week of contributions.

GitHub briefly had this feature in 2021, and I wanted an updated version of it
so I wrote this code to allow people to create their own from their GitHub
account.

# Retrieving your GitHub contributions

## Authentication
In order to authenticate with GitHub to retrieve your contributions, you must
[generate a Classic Personal Access Token](https://github.com/settings/tokens).
The token only needs one scope: `read:user`.

In order to keep from sending too many requests to GitHub, I recommend that you
save your contributions first, and then use that saved file to play around with
the other settings.

To save your contributions, use `--save -f contributions.json`:
```
$ github-skyline \
    --username someuser \
    --token ghp_dusFo29zivY4jteBv3G7Vu2Fb8fkqr3SrdeY \
    --save \
    -f contributions.json \
    --start 2011 \
    --end 2024
```

> You can also use the environment variables `GITHUB_USERNAME` and `GITHUB_TOKEN`
> to specify the credentials.

The above command pulls the entire contribution history between 2011 and 2024 (inclusive) and saves it to `contributions.json`.

# Generating an OpenSCAD file
To generate an OpenSCAD file from your contribution history, you can use the
`contributions.json` file as input so you don't have to make more requests to GitHub:

```
$ github-skyline -f contributions.json -o skyline.scad

Total contributions: 11721 between 2011-01-02 and 2024-07-25
Generating OpenSCAD ...
Skyline details:
  Buildings: 709 (36 x 20 matrix)
  Dimensions: 72.0mm x 40.0mm
OpenSCAD file written to skyline.scad
```

The output OpenSCAD file is parametric, so you can tweak most things like the
base dimensions, building width, length and height, for example:

```
// Base Parameters
baseMargin = 1.000000;
baseAngle = 22.500000;
baseHeight = 5.000000;
baseWidth = 316.000000 + (2 * baseMargin);
baseLength = 40.000000 + (2 * baseMargin);
baseOffset = baseHeight * tan(baseAngle);
baseColor = "cyan";

// Building Parameters
buildingWidth = 4.000000;
buildingLength = 4.000000;
maxBuildingHeight = 60.000000;
buildingColor = "red";
```

# Generating an STL file
In order to generate an STL file, you msut have a recent version of [OpenSCAD](https://openscad.org/downloads.html)
installed and accessible from your `PATH`.

```
$ github-skyline -f contributions.json -o skyline.stl

Total contributions: 11721 between 2011-01-02 and 2024-07-25
Generating OpenSCAD ...
Skyline details:
  Buildings: 709 (36 x 20 matrix)
  Dimensions: 72.0mm x 40.0mm
Generating STL ...
STL file written to skyline.stl in 23.530318653s
```

> The STL file is generated in millimeters.

# Skyline options
For an up-to-date list of options, use `github-skyline --help`:
```
  -a, --aspect-ratio string         Aspect ratio of the skyline (default "16:9")
  -A, --base-angle float            Slope of the base walls in degrees (default 22.5)
  -h, --base-height float           Height of the base (mm) (default 5)
  -g, --base-margin float           Distance from the buildings to the base walls (mm) (default 1)
  -l, --building-length float       Building length (mm) (default 2)
  -w, --building-width float        Building width (mm) (default 2)
  -f, --contributions string        File to save/load contributions (default "contributions.json")
  -e, --end int                     End year
  -i, --interval string             Interval to use for contributions (day, week) (default "week")
  -m, --max-building-height float   Max building height (mm) (default 20)
  -o, --output string               Output file (.scad and .stl are supported, but stl requires 'openscad') (default "skyline.scad")
  -s, --save                        Save contributions to a file
  -b, --start int                   Start year
  -t, --token string                GitHub token
  -u, --username string             GitHub username
  ```
