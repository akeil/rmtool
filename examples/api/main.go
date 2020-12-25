package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/pkg/api"
)

func main() {
	client, err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	/*
		err = register(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/

	/*
		err = list(client)
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

	err = repository(client)
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

func list(c *api.Client) error {
	items, err := c.List()
	if err != nil {
		return err
	}

	for _, item := range items {
		fmt.Printf("%v - %v\n", item.ID, item.VisibleName)
	}

	if len(items) == 0 {
		fmt.Println("List is empty")
		return nil
	}

	id := items[0].ID
	id = "e147e6dc-bf10-45d8-be95-a0d58ff40dd4"
	item, err := c.Fetch(id)
	if err != nil {
		return err
	}

	fmt.Println(item)

	return nil
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

func repository(c *api.Client) error {
	repo := api.NewRepository(c)
	fmt.Println(repo)
	items, err := repo.List()
	if err != nil {
		return err
	}

	fmt.Println(items)
	for _, i := range items {
		fmt.Printf("%v - %v", i.ID(), i.Name())
	}

	// choose a random entry and download stuff
	item := items[2]
	r, err := repo.Reader(item.ID(), item.Version(), item.ID()+".content")
	if err != nil {
		return err
	}
	defer r.Close()

	var content rm.Content
	err = json.NewDecoder(r).Decode(&content)
	if err != nil {
		return err
	}
	fmt.Println(content)

	return nil
}
