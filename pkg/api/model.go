package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"akeil.net/akeil/rm"
)

// Item holds the data for a single metadata entry from the API.
// The Item struct is also used by the service to send the response.
type Item struct {
	// ID is the UUID for this item.
	ID string

	// Version is incremented with each update.
	Version int

	// Type describes the type of item (Notebook or Folder).
	Type rm.NotebookType

	// VisibleName is the display name for an item.
	VisibleName string `json:"VissibleName"`

	// CurrentPage is the last opened page from a notebook,
	CurrentPage int

	// Bookmarked tells if this item is "pinned".
	Bookmarked bool

	// Parent is the id of the parent item.
	// It can be the empty string if the item is contained in the root folder.
	// The special value "trash" is used for deleted items.
	Parent string

	// Success is set to false if this item is sent by the server as a response
	// to a request.
	Success bool

	// Message should contain the error message if Success is false.
	Message string

	// BlobURLGet is the download URL for the zipped content.
	BlobURLGet string

	// BlobURLGetExpires describes how long the download URL remains valid.
	BlobURLGetExpires DateTime

	// BlobURLPut is the upload URL for zipped content.
	BlobURLPut string

	// BlobURLGetExpires describes how long the upload URL remains valid.
	BlobURLPutExpires DateTime

	// ModifiedClient is the last modification date for this item.
	// It is set automatically when the Client is used to change items-
	ModifiedClient DateTime
}

// Err returns the error from an API response, if this item was received as a
// response to an API request and contains Success = false.
// Returns nil if there is no error.
func (i Item) Err() error {
	if i.Success {
		return nil
	}
	return errors.New(i.Message)
}

func (i Item) Validate() error {
	switch i.Type {
	case rm.DocumentType, rm.CollectionType:
		// ok
	default:
		return rm.NewValidationError("invalid type %v", i.Type)
	}

	if i.VisibleName == "" {
		return rm.NewValidationError("visible name must not be emtpty")
	}

	return nil
}

// reduced variant of `item` with only the updateable fields.
type uploadItem struct {
	ID             string
	Version        int
	ModifiedClient DateTime
	Type           rm.NotebookType
	VisibleName    string `json:"VissibleName"`
	CurrentPage    int
	Bookmarked     bool
	Parent         string
}

func (i Item) toUpload() uploadItem {
	return uploadItem{
		ID:             i.ID,
		Version:        i.Version,
		ModifiedClient: i.ModifiedClient,
		Type:           i.Type,
		VisibleName:    i.VisibleName,
		CurrentPage:    i.CurrentPage,
		Bookmarked:     i.Bookmarked,
		Parent:         i.Parent,
	}
}

// Registration is the data structure used to register a deivce.
type Registration struct {
	Code        string `json:"code"`
	Description string `json:"deviceDesc"`
	DeviceID    string `json:"deviceID"`
}

// Discovery is the response data from the discovery service.
type Discovery struct {
	Status string
	Host   string
}

// DateTime is the type used to serialize a Time instance to a date string
// and vice versa. Used when converting an Item to and from JSON.
type DateTime struct {
	time.Time
}

func now() DateTime {
	return DateTime{time.Now()}
}

func (d *DateTime) UnmarshalJSON(b []byte) error {
	// expects a string lke this: 1607462787637
	// with the last for digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	dt := DateTime{t}

	*d = dt
	return nil
}

func (d DateTime) MarshalJSON() ([]byte, error) {
	s := d.Format(time.RFC3339Nano)
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

// Message contains the data from a notification message,
// slightly simplified.
type Message struct {
	MessageID   string
	PublishTime time.Time
	SourceDesc  string
	SourceID    string
	Event       string
	ItemID      string
	Parent      string
	Type        string
	Bookmarked  bool
	Version     int
	VisibleName string
}

// msgWrapper used to unmarshal a notification mapper from JSON.
type msgWrapper struct {
	Msg msg    `json:"message"`
	Sub string `json:"subscription"`
}

// ToMessage creates a proper Message from a "raw" notification message.
func (w msgWrapper) toMessage() Message {
	return Message{
		MessageID:   w.Msg.ID,
		PublishTime: w.Msg.PublishTime.Time,
		Bookmarked:  bool(w.Msg.Attr.Bookmarked),
		Event:       w.Msg.Attr.Event,
		ItemID:      w.Msg.Attr.ID,
		Parent:      w.Msg.Attr.Parent,
		SourceDesc:  w.Msg.Attr.SourceDeviceDesc,
		SourceID:    w.Msg.Attr.SourceDeviceID,
		Type:        w.Msg.Attr.Type,
		Version:     int(w.Msg.Attr.Version),
		VisibleName: w.Msg.Attr.VisibleName,
	}
}

type msg struct {
	Attr        msgAttr  `json:"attributes"`
	ID          string   `json:"messageId"`
	PublishTime DateTime `json:"publishTime"`
}

type msgAttr struct {
	AuthUserID       string  `json:"auth0UserID"`
	Bookmarked       boolStr `json:bookmarked`
	Event            string  `json:"event"`
	ID               string  `json:"id"`
	Parent           string  `json:"parent"`
	SourceDeviceDesc string  `json:"sourceDeviceDesc"`
	SourceDeviceID   string  `json:"sourceDeviceID"`
	Type             string  `json:"type"`
	Version          intStr  `json:"version"`
	VisibleName      string  `json:"vissibleName"`
}

type boolStr bool

func (bs *boolStr) UnmarshalJSON(b []byte) error {
	// expects a string lke this: 1607462787637
	// with the last for digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var v bool
	switch s {
	case "true":
		v = true
	case "false":
		v = false
	default:
		return fmt.Errorf("invalid boolean value %v", s)
	}

	*bs = boolStr(v)
	return nil
}

type intStr int

func (is *intStr) UnmarshalJSON(b []byte) error {
	// expects a string lke this: 1607462787637
	// with the last for digits containing nanoseconds.
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	v, err := strconv.Atoi(s)

	*is = intStr(v)
	return nil
}
