package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	CollectionType = "CollectionType"
	DocumentType   = "DocumentType"
)

type Item struct {
	ID                string
	Version           int
	Message           string
	Success           bool
	BlobURLGet        string
	BlobURLGetExpires DateTime
	BlobURLPut        string
	BlobURLPutExpires DateTime
	ModifiedClient    DateTime
	Type              string
	VisibleName       string `json:"VissibleName"`
	CurrentPage       int
	Bookmarked        bool
	Parent            string
}

func errorFrom(i Item) error {
	if i.Success {
		return nil
	}
	return errors.New(i.Message)
}

// reduced variant of `item` with only the updateable fields.
type uploadItem struct {
	ID             string
	Version        int
	ModifiedClient DateTime
	Type           string
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

type Registration struct {
	Code        string `json:"code"`
	Description string `json:"deviceDesc"`
	DeviceID    string `json:"deviceID"`
}

type Discovery struct {
	Status string
	Host   string
}

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

type Message struct {
	MessageID   string
	PublishTime time.Time
	Bookmarked  bool
	Event       string
	ItemID      string
	Parent      string
	SourceDesc  string
	SourceID    string
	Type        string
	Version     int
	VisibleName string
}

// used to unmarshal from JSON
type msgWrapper struct {
	Msg msg    `json:"message"`
	Sub string `json:"subscription"`
}

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
