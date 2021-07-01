package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/akeil/rmtool"
	"github.com/akeil/rmtool/pkg/api"
)

const (
	checkmark = "\u2713"
	crossmark = "\u2717"
	ellipsis  = "\u2026"
)

func main() {
	app := kingpin.New("rmtool", "reMarkable Tool")
	app.HelpFlag.Short('h')

	var (
		verbose = app.Flag("verbose", "Print debug messages").Short('v').Bool()
	)

	ls := app.Command("ls", "List notebooks").Default()
	var (
		pinned = ls.Flag("pinned", "Show only pinned items").Short('p').Bool()
		format = ls.Flag("format", "Output format").Short('f').Default("tree").String()
		match  = ls.Arg("match", "Name must match this").String()
	)

	get := app.Command("get", "Download one or more notebooks in PDF format")
	var (
		matchGet = get.Arg("match", "Name must match this").String()
		outDir   = get.Flag("output", "Output directory").Short('o').Default(".").String()
		mkDirs   = get.Flag("dirs", "Create subdirectories from tablet's folders").Short('d').Bool()
	)

	put := app.Command("put", "Upload PDF documents to reMarkable")
	var (
		paths = put.Arg("paths", "Source and destination paths").Strings()
		// TODO: --pin to immediately pin the item
	)

	pin := app.Command("pin", "Add or remove a bookmark")
	var (
		matchPin = pin.Arg("match", "Which documents or folders to pin").String()
		unpin    = pin.Flag("negate", "Remove a bookmark").Short('n').Bool()
	)

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *verbose {
		rmtool.SetLogLevel("debug")
	} else {
		rmtool.SetLogLevel("warning")
	}

	settings, err := loadSettings()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	switch command {
	case "ls":
		err = doLs(settings, *format, *match, *pinned)
	case "get":
		err = doGet(settings, *matchGet, *outDir, *mkDirs)
	case "put":
		err = doPut(settings, *paths)
	case "pin":
		err = doPin(settings, *matchPin, !*unpin)
	default:
		err = fmt.Errorf("unknown command: %q", command)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

type settings struct {
	dataDir  string
	cacheDir string
}

func loadSettings() (settings, error) {
	var s settings
	// TODO: from env vars
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		// TODO linux only
		home, err := os.UserHomeDir()
		if err != nil {
			return s, err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	s.dataDir = filepath.Join(dataHome, "rmtool")

	cacheHome, err := os.UserCacheDir()
	if err != nil {
		return s, err
	}
	s.cacheDir = filepath.Join(cacheHome, "rmtool")

	return s, nil
}

func setupRepo(s settings) (rmtool.Repository, error) {
	client, err := setupClient(s)
	if err != nil {
		return nil, err
	}

	repo := api.NewRepository(client, s.cacheDir)
	return repo, nil
}

func setupClient(s settings) (*api.Client, error) {
	var token string
	token, err := readToken(s)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	if !client.IsRegistered() {
		err = register(s, client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func promptUserForRegCode() (string, error) {
	fmt.Print("Enter one-time code (go to https://my.remarkable.com/device/connect/desktop): ")
	var code string

	// Taking input from user
	_, err := fmt.Scanln(&code)
	if err != nil {
		return "", err
	}

	if len(code) != 8 {
		fmt.Printf("Code has the wrong length, it should be 8, but got %d '%s'\n", len(code), code)
		return promptUserForRegCode()
	}

	return code, nil
}

func register(s settings, client *api.Client) error {

	code, err := promptUserForRegCode()
	if err != nil {
		return err
	}
	token, err := client.Register(code)
	if err != nil {
		return err
	}

	saveToken(s, token)

	return nil
}

func readToken(s settings) (string, error) {
	tokenfile := filepath.Join(s.dataDir, "device-token")
	f, err := os.Open(tokenfile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	d, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(d), err
}

func saveToken(s settings, token string) {
	// ignoring error b/c we have error handling for the case of unable to write the token
	_ = os.MkdirAll(s.dataDir, 0755)
	tokenfile := filepath.Join(s.dataDir, "device-token")
	f, err := os.Create(tokenfile)
	if err != nil {
		fmt.Printf("Failed to save token to %q: %v\n", tokenfile, err)
	}
	defer f.Close()

	f.Write([]byte(token))
}
