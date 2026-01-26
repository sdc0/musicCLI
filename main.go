package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	"github.com/go-flac/go-flac/v2"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

type artist struct {
	ID                         int
	Name                       string
	ArtistTypes                []any
	URL                        string
	Picture                    string
	SelectedAlbumCoverFallback string
	Popularity                 int
	ArtistRoles                []struct {
		CategoryID int
		Category   string
	}
	Mixes       map[string]any
	Handle      string
	UserId      int
	Spotlighted bool
}

type condensed_artist struct {
	ID      int
	Name    string
	Handle  any
	Type    string
	Picture string
}

type album struct {
	ID                     int
	Title                  string
	Duration               int
	StreamReady            bool
	PayToStream            bool
	AdSupportedStreamReady bool
	DJReady                bool
	StemReady              bool
	StreamStartDate        string
	AllowStreaming         bool
	PremiumStreamingOnly   bool
	NumberOfTracks         int
	NumberOfVideos         int
	NumberOfVolumes        int
	ReleaseDate            string
	Copyright              string
	Type                   string
	Version                any
	URL                    string
	Cover                  string
	VibrantColor           any
	VideoCover             any
	Explicit               bool
	UPC                    string
	Popularity             int
	AudioQuality           string
	AudioModes             []string
	MediaMetadata          map[string]any
	Upload                 bool
	Artist                 condensed_artist
	Artists                []condensed_artist
}

type condensed_album struct {
	ID           int
	Title        string
	Cover        string
	VibrantColor string
	VideoCover   any
}

type track struct {
	ID                     int
	Title                  string
	Duration               int
	ReplayGain             float64
	Peak                   float64
	AllowStreaming         bool
	StreamReady            bool
	PayToStream            bool
	AdSupportedStreamReady bool
	DJReady                bool
	StemReady              bool
	StreamStartDate        string
	PremiumStreamingOnly   bool
	TrackNumber            int
	VolumeNumber           int
	Version                any
	Popularity             int
	Copyright              string
	BPM                    int
	Key                    string
	KeyScale               any
	URL                    string
	ISRC                   string
	Editable               bool
	Explicit               bool
	AudioQuality           string
	AudioModes             []string
	MediaMetadata          map[string]any
	Upload                 bool
	AccessType             string
	Spotlighted            bool
	Artist                 condensed_artist
	Artists                []condensed_artist
	Album                  condensed_album
	Mixes                  map[string]string
}

type query[T any] struct {
	Limit              int
	Offset             int
	TotalNumberOfItems int
	Items              []T
}

type artist_call struct {
	Version string
	Data    struct {
		Artists   query[artist]
		Albums    query[any]
		Playlists query[any]
		Tracks    query[any]
		Videos    query[any]
		Genres    []any
		TopHits   []any
	}
}

type artist_albums_call struct {
	Version string
	Albums  struct {
		Items []album
	}
}

type album_call struct {
	Version string
	Data    struct {
		album
		Items []struct {
			Item track
			Type string
		}
	}
}

type track_call struct {
	Version string
	Data    struct {
		TrackID            int
		AssetPresentation  string
		AudioMode          string
		AudioQuality       string
		ManifestMimeType   string
		ManifestHash       string
		Manifest           string
		AlbumReplayGain    float64
		AlbumPeakAmplitude float64
		TrackReplayGain    float64
		TrackPeakAmplitude float64
		BitDepth           int
		SampleRate         int
	}
}

type decoded_track_call struct {
	MimeType       string
	Codecs         string
	EncryptionType string
	URLs           []string
}

type cover_call struct {
	Version string
	Covers  []struct {
		ID      int
		Name    string
		HighRes string `json:"1280"`
		MedRes  string `json:"640"`
		LowRes  string `json:"80"`
	}
}

type prevDataType interface {
	DisplayName() string
	IDNum() int
	SetPrev()
}

func (a artist) DisplayName() string { return a.Name }
func (a artist) IDNum() int          { return a.ID }
func (a artist) SetPrev()            { saved_artist = &a }

func (a album) DisplayName() string { return a.Title }
func (a album) IDNum() int          { return a.ID }
func (a album) SetPrev()            { saved_album = &a }

func (t track) DisplayName() string { return t.Title }
func (t track) IDNum() int          { return t.ID }
func (t track) SetPrev()            { saved_track = &t }

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func getBody[T any](url string, data *T) error {
	resp, err := http.Get(url)
	check(err)

	body, err := io.ReadAll(resp.Body)
	check(err)

	err = json.Unmarshal(body, data)
	check(err)

	return nil
}

func getArtistsFromName(name string) []artist {
	var data artist_call
	err := getBody(fmt.Sprintf("http://triton.squid.wtf/search?a=%s", strings.ReplaceAll(name, " ", "")), &data)
	check(err)

	return data.Data.Artists.Items
}

func getAlbumsFromArtist(id int) []album {
	var data artist_albums_call
	err := getBody(fmt.Sprintf("https://triton.squid.wtf/artist?f=%d&skip_tracks=True", id), &data)
	check(err)

	return data.Albums.Items
}

func getAlbumFromID(id int) album {
	var data album_call
	err := getBody(fmt.Sprintf("http://triton.squid.wtf/album?id=%d", id), &data)
	check(err)

	return data.Data.album
}

func getTracksFromAlbum(id int) ([]track, int) {
	var data album_call
	err := getBody(fmt.Sprintf("http://triton.squid.wtf/album?id=%d", id), &data)
	check(err)

	var tracks []track
	for _, t := range data.Data.Items {
		if t.Type == "track" {
			tracks = append(tracks, t.Item)
		}
	}

	return tracks, data.Data.NumberOfTracks
}

func getTrackData(id int) []byte {
	var data track_call
	err := getBody(fmt.Sprintf("https://triton.squid.wtf/track/?id=%d", id), &data)
	check(err)

	decoded, err := base64.StdEncoding.DecodeString(data.Data.Manifest)
	check(err)

	var track decoded_track_call
	err = json.Unmarshal(decoded, &track)
	check(err)

	resp, err := http.Get(track.URLs[0])
	check(err)
	body, err := io.ReadAll(resp.Body)
	check(err)

	return body
}

func ConvertFlacToMp3(flacPath string, mp3Path string) {
	_ = os.Remove(mp3Path)
	_ = os.MkdirAll(filepath.Dir(mp3Path), 0755)

	prev := log.Writer()
	log.SetOutput(io.Discard) // disable Go logging to prevent ffmpeg-go from logging

	err := ffmpeg.Input(flacPath).Output(mp3Path, ffmpeg.KwArgs{
		"c:v":      "copy",
		"loglevel": "quiet",
	}).OverWriteOutput().ErrorToStdOut().Run()

	log.SetOutput(prev) // re-enable Go logging
	check(err)

	err = os.Remove(flacPath)
	check(err)
}

func getCoverArt(id int) []byte {
	resp, err := http.Get(fmt.Sprintf("https://triton.squid.wtf/cover?id=%d", id))
	check(err)

	body, err := io.ReadAll(resp.Body)
	check(err)

	var data cover_call
	err = json.Unmarshal(body, &data)
	check(err)

	resp, err = http.Get(data.Covers[0].MedRes)
	check(err)

	body, err = io.ReadAll(resp.Body)
	check(err)

	return body
}

func writeData(data []byte, t track, a album, path string) {
	err := os.Remove(path)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	check(err)

	defer f.Close()

	_, err = f.Write(data)
	check(err)

	f.Sync()

	// metadata fixes
	imgData := getCoverArt(t.ID)

	file, err := flac.ParseFile(path)
	picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Front cover", imgData, "image/jpeg")
	picturemeta := picture.Marshal()
	file.Meta = append(file.Meta, &picturemeta)

	cmts := flacvorbis.New()
	cmts.Add(flacvorbis.FIELD_TITLE, t.Title)
	cmts.Add(flacvorbis.FIELD_ARTIST, t.Artist.Name)
	cmts.Add(flacvorbis.FIELD_ALBUM, t.Album.Title)
	cmts.Add(flacvorbis.FIELD_DATE, a.ReleaseDate)
	cmts.Add(flacvorbis.FIELD_TRACKNUMBER, strconv.Itoa(t.TrackNumber))
	cmtsmeta := cmts.Marshal()
	file.Meta = append(file.Meta, &cmtsmeta)

	file.Save(path)
}

func getArgs(reader *bufio.Reader) []string {
	c, _ := reader.ReadString('\n')
	r := csv.NewReader(strings.NewReader(c))
	r.Comma = ' '

	args, err := r.Read()
	check(err)

	return args
}

// commhand shared globals (used for saving between commands)
var saved_artist *artist = nil
var saved_album *album = nil
var saved_track *track = nil
var saved_artist_list []artist = nil
var saved_album_list []album = nil
var saved_track_list []track = nil
var prev any
var dir string

var reader = bufio.NewReader(os.Stdin)
var readRestore *term.State

// command tree struct
type command struct {
	Pattern       string
	Run           func(args []string)
	Children      map[string]*command
	NonTaggedArgs bool
}

// start of all commhand functions (take user input)
func help(args []string) {
	if len(args) == 1 {
		fmt.Print("help for all commands\r\n")
		return
	}

	switch args[1] {
	case "artist":
		fmt.Print("help for artist commands\r\n")
		return
	case "album":
		fmt.Print("help for album commands\r\n")
		return
	case "track":
		fmt.Print("help for track command\r\n")
		return
	default:
		fmt.Print("command has no help, retry\r\n")
	}
}

func printPrev[T prevDataType](data *T, datatype string) {
	if data == nil {
		fmt.Printf("Previous %s is not initialized, retry\r\n", datatype)
		return
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", (*data).DisplayName(), (*data).IDNum())
}

func artistComm(args []string) {
	if len(args) < 2 {
		fmt.Printf("Provide an artist name when searching\r\n")
		return
	}

	prev = getArtistsFromName(args[1])[0]

	t, ok := prev.(prevDataType)
	if !ok {
		fmt.Printf("Artist \"%s\" couldn't be found\r\n", args[1])
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func albumComm(args []string) {
	var id int
	var err error

	if len(args) >= 2 {
		id, err = strconv.Atoi(args[1])
		check(err)
	} else if saved_artist != nil {
		id = saved_artist.ID
	} else {
		fmt.Printf("Provide an artist id or saved artist when searching\r\n")
		return
	}

	albums := getAlbumsFromArtist(id)
	for i, a := range albums {
		fmt.Printf("[%3d] Name: %s (ID: %d)\r\n", i, a.Title, a.ID)
	}

	// prompt and get index
	term.Restore(int(os.Stdin.Fd()), readRestore)

	var index int = -1
	for index == -1 {
		fmt.Print("Enter an index: ")
		c, err := reader.ReadString('\n')
		check(err)

		if i, err := strconv.Atoi(strings.ReplaceAll(c, "\n", "")); err == nil {
			index = i
		}
	}

	readRestore, err = term.MakeRaw(int(os.Stdin.Fd()))
	check(err)

	prev = albums[index]

	t, ok := prev.(prevDataType)
	if !ok {
		fmt.Printf("Album \"%s\" couldn't be found\r\n", args[1])
		return
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func trackComm(args []string) {
	var id int
	var err error

	if len(args) >= 2 {
		id, err = strconv.Atoi(args[1])
		check(err)
	} else if saved_album != nil {
		id = saved_album.ID
	} else {
		fmt.Printf("Provide an album id or saved album when searching\r\n")
		return
	}

	tracks, trackNum := getTracksFromAlbum(id)
	for i, a := range tracks {
		fmt.Printf("[%3d/%3d] Name: %s (ID: %d)\r\n", i+1, trackNum, a.Title, a.ID)
	}

	// prompt and get index
	term.Restore(int(os.Stdin.Fd()), readRestore)

	var index int = -1
	for index == -1 {
		fmt.Print("Enter an index: ")
		c, err := reader.ReadString('\n')
		check(err)

		if i, err := strconv.Atoi(strings.ReplaceAll(c, "\n", "")); err == nil {
			index = i - 1
		}
	}

	readRestore, err = term.MakeRaw(int(os.Stdin.Fd()))
	check(err)

	prev = tracks[index]

	t, ok := prev.(prevDataType)
	if !ok {
		fmt.Printf("Track \"%s\" couldn't be found\r\n", args[1])
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func commhand(args []string, commands map[string]*command, currentArg int) (bool, bool) {
	c, ok := commands[args[currentArg]]
	if !ok {
		return false, false
	}

	//check next level for a match
	if currentArg < len(args)-1 {
		// only run if EITHER (1+2) OR (3)
		// (1) current command doesn't take non-tagged args (covered by below 1st condition)
		// (2) the next arg is not tagged (covered by below 2nd condition)
		// (3) no subcommand exists for next arg (covered by next conditional)
		nextSuccessful, nextAccepted := commhand(args, c.Children, currentArg+1)
		tagged := !c.NonTaggedArgs && args[currentArg+1][0] != '-'
		if nextSuccessful {
			return true, true
		} else if tagged && !nextAccepted {
			fmt.Printf("subcommand \"%s\" not recognized\r\n", args[currentArg+1])
			return false, true
		} else if tagged {
			return false, true
		}
	}

	c.Run(args)

	return true, true
}

var commands map[string]*command = map[string]*command{
	"echo": {
		Pattern: "echo [a-zA-Z0-9 '\"]*",
		Run: func(args []string) {
			for i, arg := range args {
				if i != len(args)-1 {
					fmt.Printf("%s, ", arg)
				} else {
					fmt.Printf("%s\r\n", arg)
				}
			}
		},
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"quit": {
		Pattern:       "quit",
		Run:           func(args []string) { panic("Exiting") },
		NonTaggedArgs: false,
		Children:      map[string]*command{},
	},
	"help": {
		Pattern:       "help [a-zA-Z0-9 ]*",
		Run:           help,
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"save": {
		Pattern: "save",
		Run: func(args []string) {
			t, ok := prev.(prevDataType)
			if !ok {
				fmt.Print("prev is of unknown type\r\n")
				return
			}

			t.SetPrev()
		},
		NonTaggedArgs: false,
		Children: map[string]*command{
			"print": {
				Pattern: "save print",
				Run: func(args []string) {
					t, ok := prev.(prevDataType)
					if !ok {
						fmt.Print("Previous output is of unknown type, retry\r\n")
						return
					}

					fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
				},
				NonTaggedArgs: false,
				Children: map[string]*command{
					"artist": {
						Pattern: "save print artist",
						Run: func(args []string) {
							printPrev(saved_artist, "artist")
						},
						NonTaggedArgs: false,
						Children:      map[string]*command{},
					},
					"album": {
						Pattern: "save print album",
						Run: func(args []string) {
							printPrev(saved_album, "album")
						},
						NonTaggedArgs: false,
						Children:      map[string]*command{},
					},
					"track": {
						Pattern: "save print track",
						Run: func(args []string) {
							printPrev(saved_track, "track")
						},
						NonTaggedArgs: false,
						Children:      map[string]*command{},
					},
					"all": {
						Pattern: "save print all",
						Run: func(args []string) {
							printPrev(saved_artist, "artist")
							printPrev(saved_album, "album")
							printPrev(saved_track, "track")
						},
						NonTaggedArgs: false,
						Children:      map[string]*command{},
					},
				},
			},
			"list": {
				Pattern: "save list",
			},
		},
	},
	"artist": {
		Pattern:       "artist \"[a-zA-Z0-9 ]*\"",
		Run:           artistComm,
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"album": {
		Pattern:       "album [0-9]*",
		Run:           albumComm,
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"track": {
		Pattern:       "album [0-9]*",
		Run:           trackComm,
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"dir": {
		Pattern: "dir",
		Run: func(args []string) {
			if len(args) < 2 {
				fmt.Printf("Saved Directory: \"%s\"\r\n", dir)
				return
			}

			path, err := filepath.Abs(args[1])
			check(err)
			dir = path
		},
		NonTaggedArgs: true,
		Children:      map[string]*command{},
	},
	"download": {
		Pattern: "download",
		Run: func(args []string) {
			if saved_track == nil {
				fmt.Printf("No track is saved\r\n")
				return
			}

			a := getAlbumFromID(saved_track.Album.ID)

			data := getTrackData(saved_track.ID)
			writeData(data, *saved_track, a, fmt.Sprintf("%s/temp.flac", dir))
			ConvertFlacToMp3(fmt.Sprintf("%s/temp.flac", dir), fmt.Sprintf("%s/%s.mp3", dir, strings.ReplaceAll(saved_track.Title, " ", "_")))
		},
		NonTaggedArgs: false,
		Children: map[string]*command{
			"album": {
				Pattern: "download album",
				Run: func(args []string) {
					if saved_album == nil {
						fmt.Printf("No album is saved\r\n")
						return
					}

					tracks, _ := getTracksFromAlbum(saved_album.ID)
					for _, t := range tracks {
						data := getTrackData(t.ID)
						writeData(data, t, *saved_album, fmt.Sprintf("%s/temp.flac", dir))
						ConvertFlacToMp3(fmt.Sprintf("%s/temp.flac", dir), fmt.Sprintf("%s/%s/%s.mp3", dir, strings.ReplaceAll(saved_album.Title, " ", "_"), strings.ReplaceAll(t.Title, " ", "_")))
					}
				},
				NonTaggedArgs: false,
				Children:      map[string]*command{},
			},
		},
	},
}

func main() {
	cmd := &cli.Command{
		Name:  "console",
		Usage: "Start console prompt for interfacing with music grabber",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			running := true

			fmt.Println("Enter a new command (\"help\" for syntax) and press enter:")
			var err error

			for running {
				func() {
					fmt.Print(" >> ")
					args := getArgs(reader)

					readRestore, err = term.MakeRaw(int(os.Stdin.Fd()))
					check(err)

					defer func() {
						term.Restore(int(os.Stdin.Fd()), readRestore)
						if r := recover(); r != nil {
							os.Exit(1)
						}
					}()

					commhand(args, commands, 0)
				}()
			}

			return nil
		},
		/*Commands: []*cli.Command{
			{
				Name:  "artist",
				Usage: "Get information related to an artist",
				Commands: []*cli.Command{
					{
						Name:  "info",
						Usage: "Get information about an artist",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							artists := getArtistsFromName("The New Pornographers")
							for _, a := range artists {
								fmt.Printf("Name: %s (ID: %d)\r\n", a.Name, a.ID)
							}

							return nil
						},
					},
					{
						Name:  "albums",
						Usage: "Gets album titles and IDs (with optional track inclusion) from an artist",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							albums := getAlbumsFromArtist(3575035)
							for _, i := range albums {
								fmt.Printf("Album: %s (ID: %d)\r\n", i.Title, i.ID)
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "album",
				Usage: "Grabs data about albums",
				Commands: []*cli.Command{
					{
						Name:  "tracks",
						Usage: "Get tracks and their IDs from album",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							tracks, numTracks := getTracksFromAlbum(49793859)

							fmt.Printf("%s Tracks:\r\n", tracks[0].Album.Title)
							for _, t := range tracks {
								fmt.Printf("[%2d/%d] Title: %s (ID: %d)\r\n", t.TrackNumber, numTracks, t.Title, t.ID)
							}

							return nil
						},
					},
				},
			},
			{
				Name:  "track",
				Usage: "Get track data",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					path, err := os.UserHomeDir()
					check(err)
					flacPath := filepath.Join(path, "test_data.flac")
					mp3Path := filepath.Join(path, "test_data.mp3")

					data := getTrackData(49793860)
					writeData(data, flacPath)
					ConvertFlacToMp3(flacPath, mp3Path)

					return nil
				},
			},
		},*/
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
