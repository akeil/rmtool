package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const (
	AuthURL = "https://my.remarkable.com"
)

// API endpoints
const (
	// auth
	epRegister = "/token/json/2/device/new"
	epRefresh  = "/token/json/2/user/new"
	// storage
	epList         = "/document-storage/json/2/docs"
	epUpload       = "/document-storage/json/2/upload/request"
	epUpdateStatus = "/document-storage/json/2/upload/update-status"
	epDelete       = "/document-storage/json/2/delete"
)

type Client struct {
	// discoveryHost
	authBase string
	// storageHost
	deviceToken string
	userToken   string
	client      *http.Client
}

func NewClient(authBase, deviceToken string) *Client {
	return &Client{
		authBase:    authBase,
		deviceToken: deviceToken,
		client:      &http.Client{},
	}
}

func (c *Client) List() ([]Item, error) {
	// fetches a list of Item's
	return nil, nil
}

func (c *Client) Fetch(id string) (Item, error) {
	// use List endpoint, but add params:
	//
	// doc = <id>
	// withBlob=true

	// fetches a list with one item
	return Item{}, nil
}

func (c *Client) fetchBlob(url string) error {
	// fetches the "Blob" from a blob URL
	// this is a Zip archive with the same files that are present on the tablets file system.
	return nil
}

// CreateFolder creates a new folder under the given parent folder.
// The parentId must be empty (root folder) or refer to a CollectinType item.
func (c *Client) CreateFolder(parentId, name string) error {
	// TODO: generate the UUID here?
	item := Item{
		ID:          "TODO",
		Type:        "CollectionType",
		Parent:      parentId,
		VisibleName: name,
	}

	return c.update(item)
}

// Delete deletes document or folder referred to by the given ID.
func (c *Client) Delete(id string) error {
	// send a list of Items with documents to be deleted
	// assumption: requires the Version fields to match
	// Item{ID: id, Version: version}
	return nil
}

// Move transfers the documents with the given id to a destination folder.
// The dstId must be empty (root folder) or refer to a CollectinType item.
func (c *Client) Move(id, dstId string) error {
	// use epUpdateStatus
	// create an item with ID and Parent set
	// should also set new Modified and Version fields -> requires to fetch the item first?
	item, err := c.Fetch(id)
	if err != nil {
		return err
	}

	// TODO: copy item before changing it?
	item.Parent = dstId

	return c.update(item)
}

// Rename, Bookmark/Unbookmark

// Upload adds a document to the given parent folder.
// The parentId must be empty (root folder) or refer to a CollectinType item.
func (c *Client) Upload(parentId string) error {
	// TODO: supply an io.Reader for the source?

	// create upload Item, PUT to epUpload

	// response from upload req contains BlobPutURL

	// PUT the zip file

	// create metadata (upload Item with Modified=Now and Version +=1)
	return nil
}

// Update updates the metadata for an item
func (c *Client) update(i Item) error {
	// increment version, set modified = now
	i.Version += 1
	// PUT epUpdateStatus
	return nil
}

// Register registers a new device with the remarkable service.
// It sends a one-time code from my.remarkable.com/connect/desktop
// and retrieves a "device token" which can later be used to authenticate.
//
// Returns the device token.
func (c *Client) Register(code string) (string, error) {
	// Assumption: we do not have to remember our device ID.
	deviceId := uuid.New().String()
	reg := &Registration{
		Code:        code,
		Description: "desktop-windows",
		DeviceID:    deviceId,
	}

	token, err := c.requestToken(epRegister, "", reg)
	if err != nil {
		return "", err
	}

	c.deviceToken = token
	c.userToken = ""

	return token, nil
}

// Registered tells if this client thinks it is registered.
// This merely looks if a device token is present; that token might still be invalid.
func (c *Client) Registered() bool {
	return c.deviceToken != ""
}

// Authenticate requests a user token from the remarkable API.
// This requires that the device is registered and the we have a valid
// "device token".
func (c *Client) RefreshToken() error {
	c.userToken = ""
	if c.deviceToken == "" {
		return fmt.Errorf("device not registered/missing device token")
	}

	token, err := c.requestToken(epRefresh, c.deviceToken, nil)
	if err != nil {
		return err
	}
	c.userToken = token

	return nil
}

func (c *Client) requestToken(endpoint, token string, payload interface{}) (string, error) {
	req, err := newRequest("POST", c.authBase, endpoint, token, payload)
	if err != nil {
		return "", err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// attempt to read an error message from the response
		var msg string
		d, xerr := ioutil.ReadAll(res.Body)
		if xerr == nil {
			msg = string(d)
			msg = strings.TrimSpace(msg)
		}
		return "", fmt.Errorf("token request failed with status %d: %q", res.StatusCode, msg)
	}

	// The token is returned as a plain string
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// discover is used to determine the endpoints that should be used for Storage
// and Notifications.
// Call this once to initialize the client.
func (c *Client) discover() error {
	// GET
	// ?environment=production&group=<MAGIC>&apiVer=<VERSION>
	//
	// MAGIC: auth0|5a68dc51cb30df3877a1d7c4
	//
	// Version:
	// - Storage: 2
	// - Notifications: 1

	return nil
}

func newRequest(method, base, endpoint, token string, payload interface{}) (*http.Request, error) {
	url, err := resolve(base, endpoint)
	if err != nil {
		return nil, err
	}

	// If we have payload, encode it to JSON
	var body io.ReadWriter
	if payload != nil {
		body = &bytes.Buffer{}
		enc := json.NewEncoder(body)
		err = enc.Encode(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set required headers
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	//req.Header.Set("Accept", "application/json")  // necessary?

	return req, nil
}

func resolve(base, endpoint string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	e, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	return b.ResolveReference(e).String(), nil
}
