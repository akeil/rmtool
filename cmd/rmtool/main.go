package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/api"
)

func main() {
	rm.SetLogLevel("warning")

	app := kingpin.New("rmtool", "reMarkable Tool")
	app.HelpFlag.Short('h')

	app.Command("ls", "List notebooks").Default()

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	var err error
	switch command {
	case "ls":
		err = doLs()
	default:
		err = fmt.Errorf("unknown command: %q", command)
	}

	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func doLs() error {
	repo, err := setupRepo()
	if err != nil {
		return err
	}

	items, err := repo.List()
	if err != nil {
		return err
	}

	root := rm.BuildTree(items)
	// TODO: filter
	filters := make([]rm.NodeFilter, 0)
	root = root.Filtered(filters...)

	root.Sort(rm.DefaultSort)

	fmt.Println("# reMarkable Notebooks:")
	showTree(root, 0)

	return nil
}

func showTree(n *rm.Node, level int) {
	for i := 0; i < level; i++ {
		fmt.Print("  ")
	}

	if n.IsLeaf() {
		fmt.Print("- ")
	} else {
		fmt.Print("+ ")
	}

	fmt.Printf(n.Name())
	if n.Pinned() {
		fmt.Print(" *")
	}

	fmt.Println()

	if !n.IsLeaf() {
		for _, c := range n.Children {
			showTree(c, level+1)
		}
	}
}

func setupRepo() (rm.Repository, error) {
	client, err := setupClient()
	if err != nil {
		return nil, err
	}

	repo := api.NewRepository(client, "/tmp/remarkable")
	return repo, nil
}

func setupClient() (*api.Client, error) {
	var token string
	token, err := readToken()
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	if !client.IsRegistered() {
		err = register(client)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func register(client *api.Client) error {
	fmt.Printf("Register rmtool with remarkable\n")
	// TODO: prompt user
	code := ""
	token, err := client.Register(code)
	if err != nil {
		return err
	}

	saveToken(token)

	return nil
}

func readToken() (string, error) {
	tokenfile := "./data/device-token"
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

func saveToken(token string) {
	tokenfile := "./data/device-token"
	f, err := os.Create(tokenfile)
	if err != nil {
		fmt.Printf("Failed to save token to %q: %v\n", tokenfile, err)
	}
	defer f.Close()

	f.Write([]byte(token))
}
