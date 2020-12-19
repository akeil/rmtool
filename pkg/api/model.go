package api

import (
	"errors"
)

type Item struct {
	ID                string
	Version           int
	Message           string
	Success           bool
	BlobURLGet        string
	BlobURLGetExpires string // datetime, 2018-01-24T21:02:59.624624Z = RFC3339Nano
	BlobURLPut        string
	BlobURLPutExpires string // datetime, 2018-01-24T21:02:59.624624Z
	ModifiedClient    string // datetime, 2018-01-24T21:02:59.624624Z
	Type              string // DocumentType or CollectionType
	VisibleName       string // key: VissibleName w/ typo
	CurrentPage       int
	Bookmarked        bool /// "pinned"
	Parent            string
}

func Err(i Item) error {
	if i.Success {
		return nil
	}
	return errors.New(i.Message)
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
