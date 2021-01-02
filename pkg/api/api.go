package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/akeil/rm"
	"github.com/akeil/rm/internal/errors"
	"github.com/akeil/rm/internal/logging"
)

// Default URLs
const (
	AuthURL                   = "https://my.remarkable.com"
	StorageDiscoveryURL       = "https://service-manager-production-dot-remarkable-production.appspot.com/service/json/1/document-storage?environment=production&group=auth0%7C5a68dc51cb30df3877a1d7c4&apiVer=2"
	NotificationsDiscoveryURL = "https://service-manager-production-dot-remarkable-production.appspot.com/service/json/1/notifications?environment=production&group=auth0%7C5a68dc51cb30df3877a1d7c4&apiVer=1"
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
	// notifications
	epNotifications = "/notifications/ws/json/1"
)

// Client represents the ReST API for the reMarkable cloud service.
type Client struct {
	discoverStorageURL string
	discoverNotifURL   string
	authBase           string
	storageBase        string
	deviceToken        string
	userToken          string
	tokenExpires       time.Time
	client             *http.Client
}

// NewClient sets up an API client with the given base URLs.
func NewClient(discoveryStorage, discoverNotif, authBase, deviceToken string) *Client {
	return &Client{
		discoverStorageURL: discoveryStorage,
		discoverNotifURL:   discoverNotif,
		authBase:           authBase,
		deviceToken:        deviceToken,
		client:             &http.Client{},
	}
}

// NewNotifications sets up a client for the notifications service.
//
// This method will retrieve the hostname for the notification service from
// the discovery URL.
// If necessary, this method will also fetch a fresh authentication token for
// the notification service.
func (c *Client) NewNotifications() (*Notifications, error) {
	host, err := c.discoverHost(c.discoverNotifURL)
	if err != nil {
		return nil, err
	}

	url := "wss://" + host + epNotifications

	if c.userToken == "" {
		err = c.refreshToken()
		if err != nil {
			return nil, err
		}
	}

	return newNotifications(url, c.userToken), nil
}

// Storage --------------------------------------------------------------------

// List retrieves the full list of items (notebooks and folders) from the
// service.
func (c *Client) List() ([]Item, error) {
	return c.doList("", false)
}

// Fetch retrieves a single item from the service
// and writes the item's blob data to the given writer.
//
// The caller is responsible for closing the writer.
func (c *Client) Fetch(id string, w io.Writer) (Item, error) {
	item, err := c.fetchItem(id)
	if err != nil {
		return item, err
	}

	if item.Type == rm.CollectionType {
		return item, fmt.Errorf("can only fetch document type items")
	}

	err = c.fetchBlob(item.BlobURLGet, w)

	return item, err
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

	logging.Debug("List request returned %d items\n", len(items))

	return items, nil
}

// FetchItem downloads metadata for a single item.
func (c *Client) fetchItem(id string) (Item, error) {
	var item Item
	// uses List endpoint, but adds params 'doc' and 'withBlob'
	items, err := c.doList(id, true)
	if err != nil {
		return item, err
	}

	if len(items) == 0 {
		return item, errors.NewNotFound("no item with id %q", id)
	} else if len(items) != 1 {
		return item, fmt.Errorf("got unexpected number of items (%v)", len(items))
	}
	item = items[0]

	// A successful response can still include errors
	err = item.Err()
	if err != nil {
		return item, err
	}

	return item, nil
}

// FetchBlob downloads the zipped content from the BlobURL
// and writes it to the given writer.
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

	err = errors.ExpectOK(res, "blob request failed")
	if err != nil {
		return err
	}

	defer res.Body.Close()
	_, err = io.Copy(w, res.Body)
	if err != nil {
		return err
	}

	return nil
}

// CreateFolder creates a new folder under the given parent folder.
// The parentID can be empty (root folder) or refer to another folder.
func (c *Client) CreateFolder(parentID, name string) error {
	// Check if the parent is an existing folder
	err := c.checkParent(parentID)
	if err != nil {
		return err
	}

	item := Item{
		ID:          uuid.New().String(),
		Type:        rm.CollectionType,
		Parent:      parentID,
		VisibleName: name,
	}

	return c.update(item)
}

// Delete a document or folder referred to by the given ID.
func (c *Client) Delete(id string) error {
	item, err := c.fetchItem(id)
	if err != nil {
		return err
	}

	// TODO: if CollectionType, check if empty
	if item.Type == rm.CollectionType {
		err = c.checkEmpty(item.ID)
		if err != nil {
			return err
		}
	}

	wrap := make([]uploadItem, 1)
	wrap[0] = item.toUpload()
	result := make([]Item, 0)
	c.storageRequest("PUT", epDelete, wrap, result)

	if len(result) != 1 {
		return fmt.Errorf("got unexpected number of items (%v)", len(result))
	}
	i := result[0]

	// A successful response can still include errors
	return i.Err()
}

// Move transfers the documents with the given id to a destination folder.
// The parentID can be empty (root folder) or refer to another folder.
func (c *Client) Move(id, parentID string) error {
	item, err := c.fetchItem(id)
	if err != nil {
		return err
	}

	// Early exit if there is no actual change
	if item.Parent == parentID {
		return nil
	}

	// Check if the parent is an existing folder
	err = c.checkParent(parentID)
	if err != nil {
		return err
	}

	item.Parent = parentID
	return c.update(item)
}

// Bookmark adds or removes a bookmark for the given item.
func (c *Client) Bookmark(id string, mark bool) error {
	item, err := c.fetchItem(id)
	if err != nil {
		return err
	}

	// Early exit if there is no actual change
	if item.Bookmarked == mark {
		return nil
	}

	item.Bookmarked = mark
	return c.update(item)
}

// Rename changes the name for an item.
func (c *Client) Rename(id, name string) error {
	item, err := c.fetchItem(id)
	if err != nil {
		return err
	}

	// Early exit if there is no actual change
	if item.VisibleName == name {
		return nil
	}

	item.VisibleName = name
	return c.update(item)
}

// Upload adds a document to the given parent folder.
// The parentID can be empty (root folder) or refer to another folder.
func (c *Client) Upload(name, id, parentID string, src io.Reader) error {
	if id == "" {
		return fmt.Errorf("id must not be empty")
	}
	// We need to check the parent folder, server will not check
	err := c.checkParent(parentID)
	if err != nil {
		return err
	}

	// Create an "upload request" which will give us the upload URL
	u := uploadItem{
		ID:      id,
		Version: 1,
	}

	wrap := make([]uploadItem, 1)
	wrap[0] = u
	result := make([]Item, 0)

	logging.Debug("create upload request for item with ID %q", id)
	err = c.storageRequest("PUT", epUpload, wrap, &result)
	if err != nil {
		return err
	}

	if len(result) != 1 {
		return fmt.Errorf("unexpected number of result documents (%d)", len(result))
	}

	i := result[0]
	err = i.Err()
	if err != nil {
		return err
	}

	// Use the Put URL to upload the zipped content.
	// The content will not be visible until we have set its metadata (below).
	err = c.putBlob(i.BlobURLPut, src)
	if err != nil {
		return err
	}

	// Set the metadata for the new item
	meta := Item{
		ID:          u.ID,
		Version:     0, // update() will increment te version; we need version 1, not 2
		Type:        rm.DocumentType,
		Parent:      parentID,
		VisibleName: name,
	}
	return c.update(meta)
}

// checkParent checks if a given id can be used as a parent,
// i.e. it exists and it is a folder.
func (c *Client) checkParent(parentID string) error {
	if parentID == "" {
		return nil
	}

	p, err := c.fetchItem(parentID)
	if err != nil {
		return err
	}

	if p.Type != rm.CollectionType {
		return fmt.Errorf("parent %q is not a collection", parentID)
	}

	return nil
}

// checkEmpty is used for a collection type to determine whether it has any
// content. Returns an error if the collection is non-empty
func (c *Client) checkEmpty(id string) error {
	items, err := c.List()
	if err != nil {
		return err
	}

	for _, item := range items {
		if item.Parent == id {
			return fmt.Errorf("collection is not empty")
		}
	}

	return nil
}

func (c *Client) putBlob(url string, src io.Reader) error {
	if url == "" {
		return fmt.Errorf("upload URL is empty")
	}

	req, err := http.NewRequest("PUT", url, src)
	if err != nil {
		return fmt.Errorf("blob upload failed with %v", err)
	}

	logging.Debug("Upload blob...")
	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("blob upload failed with %v", err)
	}

	return errors.ExpectOK(res, "blob upload failed")
}

// Update updates the metadata for an item.
func (c *Client) update(i Item) error {
	u := i.toUpload()
	u.Version++
	u.ModifiedClient = now()

	result := make([]Item, 0)
	wrap := make([]uploadItem, 1)
	wrap[0] = u

	err := c.storageRequest("PUT", epUpdate, wrap, &result)
	if err != nil {
		return err
	}

	if len(result) != 1 {
		return fmt.Errorf("unexpected response (empty list/more than one item)")
	}
	return result[0].Err()
}

func (c *Client) storageRequest(method, endpoint string, payload, dst interface{}) error {
	logging.Debug("API %v %v\n", method, endpoint)
	if c.storageBase == "" {
		err := c.discover()
		if err != nil {
			return err
		}
	}

	expired := false
	if !c.tokenExpires.IsZero() {
		// We must expect the expiration time to be unknown
		// and still be in an "OK" state.
		// If we would consider the token "expired", this would cause
		// constant refreshToken calls
		expired = c.tokenExpires.Before(time.Now())
	}
	if c.userToken == "" || expired {
		err := c.refreshToken()
		if err != nil {
			return err
		}
	}

	req, err := newRequest(method, c.storageBase, endpoint, c.userToken, payload)
	if err != nil {
		return fmt.Errorf("could not prepare API request: %v", err)
	}

	// log the request body
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		if err == nil {
			logging.Debug("Request body: %v", string(data))
			req.Body = ioutil.NopCloser(bytes.NewBuffer(data))
		}
	}

	res, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %v", err)
	}
	defer res.Body.Close()
	// must read body to end
	// https://golang.org/pkg/net/http/#Client.Do
	resData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	logging.Debug("API request %v %v returned status %v\n", req.Method, req.URL, res.StatusCode)
	logging.Debug("Response body: %v", string(resData))

	err = errors.ExpectOK(res, "storage request failed")
	if err != nil {
		return err
	}

	if dst != nil {
		dec := json.NewDecoder(bytes.NewBuffer(resData))
		err = dec.Decode(dst)
		if err != nil {
			return fmt.Errorf("failed to read API response: %v", err)
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
	deviceID := uuid.New().String()
	reg := &Registration{
		Code:        code,
		Description: "desktop-windows",
		DeviceID:    deviceID,
	}

	token, err := c.requestToken(epRegister, "", reg)
	if err != nil {
		return "", err
	}

	c.deviceToken = token
	c.userToken = ""

	return token, nil
}

// IsRegistered tells if this client thinks it is registered.
// This merely looks if a device token is present; that token might still be invalid.
func (c *Client) IsRegistered() bool {
	return c.deviceToken != ""
}

// Authenticate requests a user token from the remarkable API.
// This requires that the device is registered and the we have a valid
// "device token".
//
// The user token is stored internally and also returned to the caller.
func (c *Client) refreshToken() error {
	c.userToken = ""
	c.tokenExpires = time.Time{}

	if c.deviceToken == "" {
		return fmt.Errorf("device not registered/missing device token")
	}

	token, err := c.requestToken(epRefresh, c.deviceToken, nil)
	if err != nil {
		return err
	}

	t, parseErr := parseTokenExpiration(token)
	if parseErr == nil {
		c.tokenExpires = t
		logging.Debug("Token will expire at %v\n", t)
	} else {
		logging.Debug("Error parsing expiration time from JWT: %v\n", parseErr)
		// we still consider the token as "valid" and carry on
	}

	c.userToken = token
	return nil
}

func (c *Client) requestToken(endpoint, token string, payload interface{}) (string, error) {
	logging.Debug("Request new token from %q\n", endpoint)

	req, err := newRequest("POST", c.authBase, endpoint, token, payload)
	if err != nil {
		return "", err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	err = errors.ExpectOK(res, "token request failed")
	if err != nil {
		var msg string
		d, xerr := ioutil.ReadAll(res.Body)
		if xerr == nil {
			msg = string(d)
			msg = strings.TrimSpace(msg)
		}
		return "", errors.Wrap(err, msg)
	}

	// The token is returned as a plain string
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Discover is used internally to determine the endpoints that should be used
// for Storage. It will retrieve the storage base URL from the respective
// endpoint ONLY if the url has not been discovered yet.
//
// The call is unauthenticated and can be made before authenticaion.
func (c *Client) discover() error {
	s, err := c.discoverHost(c.discoverStorageURL)
	if err != nil {
		return err
	}

	c.storageBase = "https://" + s

	return nil
}

func (c *Client) discoverHost(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	err = errors.ExpectOK(res, "service discovery failed")
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	dis := &Discovery{}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(dis)
	if err != nil {
		return "", err
	}

	if dis.Status != "OK" {
		return "", fmt.Errorf(dis.Status)
	}

	if dis.Host == "" {
		return "", fmt.Errorf("service discovery returned empty host name")
	}

	return dis.Host, nil
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
	// Not sure if this is necessary, won't hurt either
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "rmtools")

	return req, nil
}
