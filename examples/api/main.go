package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

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

		err = list(client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	*/
	err = notifications(client)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func register(client *api.Client) error {
	var err error
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

	err = client.Discover()
	if err != nil {
		return err
	}

	// fetch a (new) user token. This must be done once per session
	err = client.RefreshToken()
	if err != nil {
		return err
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

	err := c.Discover()
	if err != nil {
		return err
	}

	// fetch a (new) user token. This must be done once per session
	err = c.RefreshToken()
	if err != nil {
		return err
	}

	n := c.Notifications()

	n.OnMessage(func(msg string) {
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
