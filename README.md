# MusicCLI
MusicCLI is a small command line tool to interface with the [HiFi API](https://github.com/binimum/hifi-api) to provide an alternative to music streaming accessible offline.

## How to use MusicCLI
MusicCLI supports a set of commands detailed below alongside file input in the form of space delimited commands.

### How to run MusicCLI
Ensure you have installed [Go](https://go.dev/dl/), then run the following commands to add the `musicCLI` command to terminal:
```
go install github.com/sdc0/musicCLI@latest
```

If you prefer to build from source instead run the following and add the resulting `musicCLI` executable to the Go bin folder:
```
git clone https://github.com/sdc0/musicCLI.git
cd musicCLI
go build
```

After doing one of the two, ensure the Go bin folder is in Path variables and reboot terminal to ensure functionality.

---

### `help` [not currently functional]
Provides support for command purposes and formatting

#### Params
- Optionally, add command name following help for details specific to the command

---

### `echo`
Test command that echoes any passed arguments, functionally useless

#### Params
- None, only words to be repeated

---

### `quit`
Exits the program

#### Params
- None

---

### `dir`
Sets the working directory for downloads

#### Params
- Path to the working directory surrounded by double quotes (i.e. `dir "path/to/dir"`)

---

### `artist`
Grabs details about an artist

#### Params
- Name of the artist surrounded by double quotes or with all spaces removed (i.e. `artist "Artist Name"` or `artist ArtistName`)
  - [temporary inconvenience] As of the current state, exact matches are required of all string arguments
- Optionally, use save flag `-s` to store output to saved artist variable

---

### `album`
Grabs details about an album

#### Params
- One of the following to provide identifying information for the album:
  - Album ID sourced from the HiFi API (uncommonly used)
  - Name of the album surrounded by double quotes or with all spaces removed (i.e.  `album "Album Name"` or `album AlbumName`)
    - Assumes a saved artist exists and will cause an error if no artist is saved or album doesn't belong to artist
    - [temporary inconvenience] As of the current state, exact matches are required of all string arguments
  - The word `artist`, which goes into a menu-based selection to choose a specific album
    - Either an artist ID sourced from the HiFi API must be provided as an additional argument or a saved artist must exist
    - Doesn't support any flags, flag-associated commands must be called separately
- Optionally, use one of the following flags:
  - The save flag `-s`, which stores the output to the saved album variable
  - The download flag `-d`, which downloads entire album following fetch
    - [temporary inconvenience] As of the currrent state, if the fetch fails and an album or song is the previously successful fetch, that item will be downloaded instead

---

### `track`
Grabs details about a track

#### Params
- One of the following to provide identifying information for the track:
  - Track ID sourced from the HiFi API (uncommonly used)
  - Name of the track surrounded by double quotes or with all spaces removed (i.e.  `track "Track Name"` or `track TrackName`)
    - Assumes a saved album exists and will cause an error if no album is saved or track doesn't belong to album
    - [temporary inconvenience] As of the current state, exact matches are required of all string arguments
  - The word `album`, which goes into a menu-based selection to choose a specific track
    - Either an album ID sourced from the HiFi API must be provided as an additional argument or a saved album must exist
    - Doesn't support any flags, flag-associated commands must be called separately
- Optionally, use one of the following flags:
  - The save flag `-s`, which stores the output to the saved track variable
  - The download flag `-d`, which downloads track following fetch
    - [temporary inconvenience] As of the currrent state, if the fetch fails and an album or song is the previously successful fetch, that item will be downloaded instead

---

###  `save`
Saves the previous fetch to corresponding variable for future usage

#### Params
- None

#### Subcommands
- `print` (i.e. `save print`)
  - Prints the previous successful fetch's results to the console, taking no params
  - Further subcommands:
    - `artist` (i.e. `save print artist`)
      - Prints the contents of the saved artist variable to the console, taking no params
    - `album` (i.e. `save print album`)
      - Prints the contents of the saved album variable to the console, taking no params
    - `track` (i.e. `save print track`)
      - Prints the contents of the saved track variable to the console, taking no params
    - `all` (i.e. `save print all`)
      - Prints the contents of all saved variables to the console in the order of artist, album, then track, taking no params
     
---

### `download`
Downloads the currently saved track, assuming it is from the currently saved album

#### Params
- None

#### Subcommands
- `album` (i.e. `download album`)
  - Downloads the entire currently saved album, taking no params

___

### `file`
Executes the commands sourced from a file sequentially, automating the entry of commands for batch inputs

#### Params
- Path of the file to be read in double quotes (i.e. `file "path/to/file"`) that is in the following format of newline separated standard format commands:
```
dir "./temp"
artist "Example Artist" -s
album "Example Album 1" -s
track "Example Track 1" -d
track "Example Track 2" -d
album "Example Album 2" -d
```
The above would download the tracks "Example Track 1" and "Example Track 2" from "Example Album 1" by "Example Artist" as well as downloading all tracks from "Example Album 2" by "Example Artist".

---

## Troubleshooting Connection Issues
MusicCLI relies on an HiFi API URL specified in the const `baseURL` towards the beginning of the `main.go` file. Network issues can be solved by swapping that `baseURL` with any of the confirmed working ones sourced from one of the following:
- [https://tidal-uptime.jiffy-puffs-1j.workers.dev](https://tidal-uptime.jiffy-puffs-1j.workers.dev)
- [https://tidal-uptime.props-76styles.workers.dev](https://tidal-uptime.props-76styles.workers.dev)

MusicCLI utilizes very few and only essential endpoints of the HiFi API and should be functional with a switch to any of the currently functioning API instances.
