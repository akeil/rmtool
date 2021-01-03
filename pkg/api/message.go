package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/akeil/rmtool"
)

// Event is the type for event types used in notification messages.
type Event int

const (
	DocAdded Event = iota
	DocDeleted
)

// Message contains the data from a notification message,
// slightly simplified.
type Message struct {
	MessageID   string
	PublishTime time.Time
	SourceDesc  string
	SourceID    string
	Event       Event
	ItemID      string
	Parent      string
	Type        rmtool.NotebookType
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
	AuthUserID       string              `json:"auth0UserID"`
	Bookmarked       boolStr             `json:"bookmarked"`
	Event            Event               `json:"event"`
	ID               string              `json:"id"`
	Parent           string              `json:"parent"`
	SourceDeviceDesc string              `json:"sourceDeviceDesc"`
	SourceDeviceID   string              `json:"sourceDeviceID"`
	Type             rmtool.NotebookType `json:"type"`
	Version          intStr              `json:"version"`
	VisibleName      string              `json:"vissibleName"`
}

// UnmarshalJSON unmarshals an Event from a JSON string value.
func (e *Event) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	var et Event
	switch s {
	case "DocAdded":
		et = DocAdded
	case "DocDeleted":
		et = DocDeleted
	default:
		return fmt.Errorf("invalid event type %q", s)
	}

	*e = et
	return nil
}

// MarshalJSON marshals an Event to a JSON string value.
func (e Event) MarshalJSON() ([]byte, error) {
	s := e.String()

	if s == "UNKNOWN" {
		return nil, fmt.Errorf("invalid notebook type %v", e)
	}

	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

	return buf.Bytes(), nil
}

func (e Event) String() string {
	switch e {
	case DocAdded:
		return "DocAdded"
	case DocDeleted:
		return "DocDeleted"
	default:
		return "UNKNOWN"
	}
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
	if err != nil {
		return err
	}

	*is = intStr(v)
	return nil
}
