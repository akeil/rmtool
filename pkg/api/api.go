package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
)

const (
	AuthURL      = "https://my.remarkable.com"
	DiscoveryURL = "https://service-manager-production-dot-remarkable-production.appspot.com/service/json/1/document-storage?environment=production&group=auth0%7C5a68dc51cb30df3877a1d7c4&apiVer=2"
)

// API endpoints
const (
	// auth
	epRegister = "/token/json/2/device/new"
	epRefresh  = "/token/json/2/user/new"
	// storage
	epList   = "/document-storage/json/2/docs"
	epUpload = "/document-storage/json/2/upload/request"
	epUpdate = "/document-storage/json/2/upload/update-status"
	epDelete = "/document-storage/json/2/delete"
)

type Client struct {
	discoveryURL string
	authBase     string
	storageBase  string
	deviceToken  string
	userToken    string
	client       *http.Client
}

func NewClient(discoveryURL, authBase, deviceToken string) *Client {
	return &Client{
		discoveryURL: discoveryURL,
		authBase:     authBase,
		deviceToken:  deviceToken,
		client:       &http.Client{},
	}
}

// Storage --------------------------------------------------------------------

func (c *Client) List() ([]Item, error) {
	items := make([]Item, 0)

	err := c.storageRequest("GET", epList, nil, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (c *Client) Fetch(id string) (Item, error) {
	item, err := c.fetchItem(id)
	if err != nil {
		return item, err
	}

	//var w bytes.Buffer
	// TODO temporary
	w, err := os.Create("./data/rm-api-blob.zip")
	if err != nil {
		return item, err
	}
	defer w.Close()
	c.fetchBlob(item.BlobURLGet, w)

	return item, nil
}

func (c *Client) doList(id string, blob bool) ([]Item, error) {
	ep, err := url.Parse(epList)
	if err != nil {
		return nil, err
	}

	// Add optional query parameters
	if blob || id != "" {
		q := url.Values{}
		q.Set("withBlob", "true")
		if id != "" {
			q.Set("doc", id)
		}
		qry, err := url.Parse("?" + q.Encode())
		if err != nil {
			return nil, err
		}
		ep = ep.ResolveReference(qry)
	}

	items := make([]Item, 0)
	err = c.storageRequest("GET", ep.String(), nil, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// FetchItem downloads metadata for a single item.
func (c *Client) fetchItem(id string) (Item, error) {
	// uses List endpoint, but adds params 'doc' and 'withBlob'
	items, err := c.doList(id, true)
	if err != nil {
		return Item{}, err
	}

	if len(items) != 1 {
		return Item{}, fmt.Errorf("got unexpected number of items (%v)", len(items))
	}
	item := items[0]

	// A successful response can still include errors
	err = errorFrom(item)
	if err != nil {
		return Item{}, err
	}

	return item, nil
}

func (c *Client) fetchBlob(url string, w io.Writer) error {
	// fetches the "Blob" from a blob URL
	// this is a Zip archive with the same files that are present on the tablets file system.
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("blob request failed with status %d", res.StatusCode)
	}

	defer res.Body.Close()
	_, err = io.Copy(w, res.Body)
	if err != nil {
		return err
	}

	return nil
}

// CreateFolder creates a new folder under the given parent folder.
// The parentId must be empty (root folder) or refer to a CollectinType item.
func (c *Client) CreateFolder(parentId, name string) error {
	item := Item{
		ID:          uuid.New().String(),
		Version:     0,
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
	item, err := c.fetchItem(id)
	if err != nil {
		return err
	}

	// Early exit if there is no actual change
	if item.Parent == dstId {
		return nil
	}

	// We need to check if the parent is an existing folder
	// (service will not check this)
	parent, err := c.fetchItem(dstId)
	if err != nil {
		return err
	}
	if parent.Type != CollectionType {
		return fmt.Errorf("Destination %q is not a collection", dstId)
	}

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
	i.Version += 1
	i.ModifiedClient = now()

	result := make([]Item, 0)
	wrap := make([]Item, 1)
	wrap[0] = i

	err := c.storageRequest("PUT", epUpdate, wrap, &result)
	if err != nil {
		return err
	}

	if len(result) == 0 {
		return fmt.Errorf("unexpected response (empty list)")
	}
	err = errorFrom(result[0])
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) storageRequest(method, endpoint string, payload, dst interface{}) error {
	req, err := newRequest(method, c.storageBase, endpoint, c.userToken, payload)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		// TODO: body can contain plain text error message
		return fmt.Errorf("storage request failed with status %d", res.StatusCode)
	}

	defer res.Body.Close()
	if dst != nil {
		dec := json.NewDecoder(res.Body)
		err = dec.Decode(dst)
		if err != nil {
			return err
		}
	}

	return nil
}

// Auth -----------------------------------------------------------------------

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

// Discover is used to determine the endpoints that should be used for Storage
// and Notifications.
// Call this once to initialize the client.
// The call is unauthenticated and can be made before authenticaion.
func (c *Client) Discover() error {
	req, err := http.NewRequest("GET", c.discoveryURL, nil)
	if err != nil {
		return err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("service discovery failed with status %d", res.StatusCode)
	}

	defer res.Body.Close()

	dis := &Discovery{}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(dis)
	if err != nil {
		return err
	}

	c.storageBase = "https://" + dis.Host

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
