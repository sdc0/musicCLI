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

// const baseURL = "http://triton.squid.wtf" // not working for some reason
const baseURL = "https://hifi-one.spotisaver.net"

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

type condensed_track struct {
	ID          int
	Title       string
	Duration    int
	TrackNumber int
	URL         string
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
		log.Panic(err)
	}
}

func getBody[T any](url string, data *T) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return nil
}

func getArtistsFromName(name string) ([]artist, error) {
	var data artist_call
	err := getBody(fmt.Sprintf("%s/search?a=%s", baseURL, strings.ReplaceAll(name, " ", "")), &data)
	if err != nil {
		fmt.Printf("Failed to grab artist with err: %s\r\n", err)
		return []artist{}, err
	}

	return data.Data.Artists.Items, nil
}

func getAlbumsFromArtist(id int) ([]album, error) {
	var data artist_albums_call
	err := getBody(fmt.Sprintf("%s/artist?f=%d&skip_tracks=True", baseURL, id), &data)
	if err != nil {
		fmt.Print("Failed to grab albums\r\n")
		return []album{}, err
	}

	return data.Albums.Items, nil
}

func getAlbumFromID(id int) (album, error) {
	var data album_call
	err := getBody(fmt.Sprintf("%s/album?id=%d", baseURL, id), &data)
	if err != nil {
		fmt.Print("Failed to grab album\r\n")
		return album{}, err
	}

	return data.Data.album, nil
}

func getTracksFromAlbum(id int) ([]track, int, error) {
	var data album_call
	err := getBody(fmt.Sprintf("%s/album?id=%d", baseURL, id), &data)
	if err != nil {
		fmt.Print("Failed to grab tracks\r\n")
		return []track{}, -1, err
	}

	var tracks []track
	for _, t := range data.Data.Items {
		if t.Type == "track" {
			tracks = append(tracks, t.Item)
		}
	}

	return tracks, data.Data.NumberOfTracks, nil
}

func getTrackFromID(id int) (track, error) {
	var data struct {
		Version string
		Data    struct {
			track
		}
	}
	err := getBody(fmt.Sprintf("%s/info/?id=%d", baseURL, id), &data)
	if err != nil {
		return track{}, err
	}

	return data.Data.track, nil
}

func getTrackData(id int) ([]byte, error) {
	errorHandle := func(err error) bool {
		if err != nil {
			fmt.Print("Failed to grab track data\r\n")
			return true
		}
		return false
	}

	var data track_call
	err := getBody(fmt.Sprintf("%s/track/?id=%d", baseURL, id), &data)
	if errorHandle(err) {
		return []byte{}, err
	}

	decoded, err := base64.StdEncoding.DecodeString(data.Data.Manifest)
	if errorHandle(err) {
		return []byte{}, err
	}

	var track decoded_track_call
	err = json.Unmarshal(decoded, &track)
	if errorHandle(err) {
		return []byte{}, err
	}

	resp, err := http.Get(track.URLs[0])
	if errorHandle(err) {
		return []byte{}, err
	}
	body, err := io.ReadAll(resp.Body)
	if errorHandle(err) {
		return []byte{}, err
	}

	return body, nil
}

func ConvertFlacToMp3(flacPath string, mp3Path string) error {
	_ = os.Remove(mp3Path)
	_ = os.MkdirAll(filepath.Dir(mp3Path), 0755)

	prev := log.Writer()
	log.SetOutput(io.Discard) // disable Go logging to prevent ffmpeg-go from logging

	err := ffmpeg.Input(flacPath).Output(mp3Path, ffmpeg.KwArgs{
		"c:v":      "copy",
		"loglevel": "quiet",
	}).OverWriteOutput().ErrorToStdOut().Run()

	log.SetOutput(prev) // re-enable Go logging
	if err != nil {
		fmt.Print("Failed in flac to mp3 conversion\r\n")
		return err
	}

	_ = os.Remove(flacPath)

	return nil
}

func getCoverArt(id int) ([]byte, error) {
	errorHandle := func(err error) bool {
		if err != nil {
			fmt.Print("Failed to grab cover art\r\n")
			return true
		}
		return false
	}

	var data cover_call
	err := getBody(fmt.Sprintf("%s/cover?id=%d", baseURL, id), &data)
	if errorHandle(err) {
		return []byte{}, err
	}

	resp, err := http.Get(data.Covers[0].MedRes)
	if errorHandle(err) {
		return []byte{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if errorHandle(err) {
		return []byte{}, err
	}

	return body, nil
}

func writeData(data []byte, t track, a album, path string) error {
	err := os.Remove(path)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	if err != nil {
		fmt.Printf("Failed during creation of flac file: %s\r\n", path)
		return err
	}

	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		fmt.Print("Failed during writing to flac file\r\n")
		return err
	}

	f.Sync()

	// metadata fixes
	imgData, err := getCoverArt(t.ID)
	if err != nil {
		return err
	}

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
	return nil
}

func getArgs(reader *bufio.Reader) ([]string, error) {
	c, _ := reader.ReadString('\n')
	r := csv.NewReader(strings.NewReader(c))
	r.Comma = ' '

	args, err := r.Read()
	if err != nil {
		fmt.Print("Failed during arg parsing\r\n")
		return []string{}, err
	}

	return args, nil
}

// commhand shared globals (used for saving between commands)
var saved_artist *artist = nil
var saved_album *album = nil
var saved_track *track = nil
var saved_artist_list []artist = nil
var saved_album_list []album = nil
var saved_track_list []track = nil
var prev any
var dir string = "./temp"

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

	artists, err := getArtistsFromName(args[1])
	if err != nil {
		return
	}
	if len(artists) == 0 {
		fmt.Printf("No artists found with name \"%s\"\r\n", args[1])
		return
	}
	prev = artists[0]

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
		id, err = strconv.Atoi(args[1]) // album ID
		if err != nil {                 // do string parsing (assuming saved artist)
			if saved_artist == nil {
				return
			}

			artist_id := saved_artist.ID
			albums, err := getAlbumsFromArtist(artist_id)
			if err != nil {
				return
			}

			id = -1
			for _, a := range albums {
				if strings.EqualFold(a.Title, args[1]) {
					id = a.ID
					break
				}
			}

			if id == -1 {
				return
			}
		}
	} else {
		fmt.Printf("Provide an album id or title when searching\r\n")
		return
	}

	album, err := getAlbumFromID(id)
	if err != nil {
		return
	}

	prev = album
	t, ok := prev.(prevDataType)
	if !ok {
		fmt.Printf("Album \"%s\" couldn't be found\r\n", args[1])
		return
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func albumFromArtistComm(args []string) {
	var id int
	var err error

	if len(args) >= 3 {
		id, err = strconv.Atoi(args[2]) // album ID
		check(err)
	} else if saved_artist != nil {
		id = saved_artist.ID
	} else {
		fmt.Printf("Provide an artist ID or saved artist when searching\r\n")
		return
	}

	albums, err := getAlbumsFromArtist(id)
	if err != nil {
		fmt.Printf("failed to get artist with error: %s\r\n", err)
		return
	}
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
		fmt.Printf("Album \"%s\" couldn't be found\r\n", args[2])
		return
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func downloadAlbum(album *album) {
	if album == nil {
		fmt.Printf("No album passed\r\n")
		return
	}

	tracks, _, err := getTracksFromAlbum(album.ID)
	if err != nil {
		return
	}
	for _, t := range tracks {
		data, err := getTrackData(t.ID)
		if err != nil {
			return
		}
		writeData(data, t, *album, fmt.Sprintf("%s/temp.flac", dir))
		ConvertFlacToMp3(fmt.Sprintf("%s/temp.flac", dir), fmt.Sprintf("%s/%s/%s.mp3", dir, strings.ReplaceAll(album.Title, " ", "_"), strings.ReplaceAll(t.Title, " ", "_")))
	}
}

func trackComm(args []string) {
	var id int
	var err error

	if len(args) >= 2 {
		id, err = strconv.Atoi(args[1]) // get ID if that is passed
		if err != nil {                 // do string parsing (assuming saved album)
			if saved_album == nil {
				fmt.Printf("Saved album is not initialized \r\n")
				return
			}

			album_id := saved_album.ID
			tracks, _, err := getTracksFromAlbum(album_id)
			if err != nil {
				fmt.Printf("Failed to fetch albums \r\n")
				return
			}

			id = -1
			for _, a := range tracks {
				if strings.EqualFold(a.Title, args[1]) {
					id = a.ID
					break
				}
			}

			if id == -1 {
				fmt.Printf("Track not matched \r\n")
				return
			}
		}
	} else if saved_track != nil {
		id = saved_track.ID
	} else {
		fmt.Printf("Provide an track id or saved track when searching\r\n")
		return
	}

	track, err := getTrackFromID(id)
	if err != nil {
		fmt.Printf("Failed to fetch track data \r\n")
		return
	}

	prev = track
	t, ok := prev.(prevDataType)
	if !ok {
		fmt.Printf("Track \"%s\" couldn't be found\r\n", args[1])
		return
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func trackFromAlbumComm(args []string) {
	var id int
	var err error

	if len(args) >= 3 {
		id, err = strconv.Atoi(args[2])
		check(err)
	} else if saved_album != nil {
		id = saved_album.ID
	} else {
		fmt.Printf("Provide an album id or saved album when searching\r\n")
		return
	}

	tracks, trackNum, err := getTracksFromAlbum(id)
	if err != nil {
		return
	}
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
		fmt.Printf("Track \"%s\" couldn't be found\r\n", args[2])
	}

	fmt.Printf("Name: %s (ID: %d)\r\n", t.DisplayName(), t.IDNum())
}

func downloadTrack(track *track, album *album) {
	if track == nil {
		fmt.Printf("No track passed\r\n")
		return
	}
	if album == nil {
		fmt.Printf("No album passed\r\n")
		return
	}

	data, err := getTrackData(track.ID)
	if err != nil {
		return
	}
	err = writeData(data, *track, *album, fmt.Sprintf("%s/temp.flac", dir))
	if err != nil {
		return
	}
	err = ConvertFlacToMp3(fmt.Sprintf("%s/temp.flac", dir), fmt.Sprintf("%s/%s.mp3", dir, strings.ReplaceAll(track.Title, " ", "_")))
	if err != nil {
		return
	}
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

	for _, a := range args[currentArg+1:] {
		switch a {
		case "-s":
			prev.(prevDataType).SetPrev()

		case "-d":
			if album, ok := prev.(album); ok {
				downloadAlbum(&album)
			} else if track, ok := prev.(track); ok {
				a, err := getAlbumFromID(track.Album.ID)
				if err != nil {
					fmt.Printf("Album could not be parsed\r\n")
					break
				}

				downloadTrack(&track, &a)
			} else {
				fmt.Print("Not album or track, couldn't download\r\n")
			}

		default:
		}
	}

	return true, true
}

var fileComm = command{
	Pattern: "file \"[a-zA-Z0-9_]+\"",
	Run: func(args []string) {
		file := args[1]
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Error opening CSV file: %s\r\n", err)
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			r := csv.NewReader(strings.NewReader(scanner.Text()))
			r.Comma = ' '

			arr, err := r.Read()
			if err != nil {
				fmt.Print("Failed during file parsing\r\n")
				return
			}

			for i, s := range arr {
				arr[i] = strings.ReplaceAll(strings.ReplaceAll(s, "\"", ""), "'", "")
			}
			commhand(arr, commands, 0)
		}
	},
	NonTaggedArgs: true,
	Children:      map[string]*command{},
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
		Children: map[string]*command{
			"artist": {
				Pattern:       "album artist [0-9]*",
				Run:           albumFromArtistComm,
				NonTaggedArgs: true,
				Children:      map[string]*command{},
			},
		},
	},
	"track": {
		Pattern:       "track [0-9]*",
		Run:           trackComm,
		NonTaggedArgs: true,
		Children: map[string]*command{
			"album": {
				Pattern:       "track album [0-9]*",
				Run:           trackFromAlbumComm,
				NonTaggedArgs: true,
				Children:      map[string]*command{},
			},
		},
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
			a, err := getAlbumFromID(saved_track.Album.ID)
			if err != nil {
				fmt.Printf("No album passed\r\n")
				return
			}
			downloadTrack(saved_track, &a)
		},
		NonTaggedArgs: false,
		Children: map[string]*command{
			"album": {
				Pattern: "download album",
				Run: func(args []string) {
					downloadAlbum(saved_album)
				},
				NonTaggedArgs: false,
				Children:      map[string]*command{},
			},
		},
	},
}

func main() {
	cmd := &cli.Command{
		Name:  "musicCLI",
		Usage: "Start console prompt for interfacing with music grabber",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			running := true

			fmt.Println("Enter a new command (\"help\" for syntax) and press enter:")
			var err error
			var args []string

			for running {
				func() {
					fmt.Print(" >> ")
					args, err = getArgs(reader)
					if err != nil {
						return
					}

					readRestore, err = term.MakeRaw(int(os.Stdin.Fd()))
					check(err)

					defer func() {
						term.Restore(int(os.Stdin.Fd()), readRestore)
						if r := recover(); r != nil {
							os.Exit(1)
						}
					}()

					fullComm := make(map[string]*command)
					for k, v := range commands {
						fullComm[k] = v
					}
					fullComm["file"] = &fileComm

					commhand(args, fullComm, 0)
				}()
			}

			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
