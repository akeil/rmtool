package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"akeil.net/akeil/rm/pkg/api"
)

func main() {
	client, err := setup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = register(client)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	items, err := client.List()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, item := range items {
		fmt.Println(item)
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
	client := api.NewClient(api.DiscoveryURL, api.AuthURL, token)

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
