package api

import (
	"bytes"
	"encoding/json"
	"errors"
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
