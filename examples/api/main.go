package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"akeil.net/akeil/rm/pkg/api"
)

func main() {
	err := register()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func register() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("invalid number of arguments")
	}

	token, err := readToken()
	if err != nil {
		return err
	}

	code := os.Args[1]
	client := api.NewClient(api.DiscoveryURL, api.AuthURL, token)

	if !client.Registered() {
		token, err = client.Register(code)
		if err != nil {
			return err
		}
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
