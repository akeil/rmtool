package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/api"
	"akeil.net/akeil/rm/pkg/fs"
)

func main() {
	rm.SetLogLevel("debug")
	var err error

	/*
		client, err := setup()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/
	/*
		err = register(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	/*
		err = notifications(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	//dataDir := "/tmp/remarkable"
	//repo := api.NewRepository(client, dataDir)
	srcDir := "/tmp/xochitl"
	repo := fs.NewRepository(srcDir)

	/*
		err = repository(repo)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	err = upload(repo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func register(client *api.Client) error {
	if !client.Registered() {
		if len(os.Args) != 2 {
			return fmt.Errorf("invalid number of arguments")
		}
		code := os.Args[1]
		token, err := client.Register(code)
		if err != nil {
			return err
		}
		saveToken(token)
	}

	return nil
}

func setup() (*api.Client, error) {
	token, err := readToken()
	if err != nil {
		return nil, err
	}
	client := api.NewClient(api.StorageDiscoveryURL, api.NotificationsDiscoveryURL, api.AuthURL, token)

	return client, nil
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
		fmt.Printf("failed to save token: %v\n", err)
	}
	defer f.Close()

	f.Write([]byte(token))
}

func notifications(c *api.Client) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	n, err := c.NewNotifications()
	if err != nil {
		return err
	}

	n.OnMessage(func(msg api.Message) {
		fmt.Printf("Message received: %v\n", msg)
	})

	err = n.Connect()
	if err != nil {
		return err
	}
	fmt.Println("Notifications connected...")
	defer n.Disconnect()

	<-interrupt
	return nil
}

func repository(repo rm.Repository) error {
	items, err := repo.List()
	if err != nil {
		return err
	}

	for _, i := range items {
		fmt.Printf("%v - %v\n", i.ID(), i.Name())
	}

	item := items[2]

	doc, err := rm.ReadDocument(repo, item)
	if err != nil {
		return err
	}
	fmt.Println(doc)

	fmt.Println("Pages:")
	for _, pageId := range doc.Pages() {
		pg, err := doc.Page(pageId)
		if err != nil {
			return err
		}
		fmt.Printf("Page %d - %v\n", pg.Number(), pg.Template())

		// Drawing
		d, err := doc.Drawing(pageId)
		if err != nil {
			return err
		}
		fmt.Printf("Drawing version=%v\n", d.Version)
	}

	item.SetPinned(true)
	err = repo.Update(item)
	if err != nil {
		return err
	}

	return nil
}

func upload(repo rm.Repository) error {

	d := rm.NewDocument("my document", rm.Notebook)
	d.SetPinned(true)

	err := repo.Upload(d)
	if err != nil {
		return err
	}

	return nil
}
