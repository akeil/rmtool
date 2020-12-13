package rmtool

import (
    "bytes"
    "encoding/json"
    "io/ioutil"
    "strconv"
    "time"
    "fmt"
)

// Timestampp is the datatype for a UNIX timestamp in string format.
type Timestamp struct {
    time.Time
}

// Metadata holds the metadata for a notebook.
type Metadata struct {
	Deleted          bool   `json:"deleted"`
	LastModified     Timestamp `json:"lastModified"`
	LastOpenedPage   uint   `json:"lastOpenedPage"`
	Metadatamodified bool   `json:"metadatamodified"`
	Modified         bool   `json:"modified"`
	Parent           string `json:"parent"`
	Pinned           bool   `json:"bool"`
	Synced           bool   `json:"synced"`
	Type             string `json:"type"`
	Version          uint   `json:"version"`
	VisibleName      string `json:"visibleName"`
}

// ReadMetadata reads a Metadata struct from the given JSON file.
//
// Note that you can also use `json.Unmarshal(data, m)`.
func ReadMetadata(path string) (Metadata, error) {
    var m Metadata
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return m, err
    }

    err = json.Unmarshal(data, &m)
    return m, err
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
    // expects a string lke this: 1607462787637
    // with the last for digits containing nanoseconds.
    var s string
    err := json.Unmarshal(b, &s)
    if err != nil {
        return err
    }

    n, err := strconv.Atoi(s)
    if err != nil {
        return err
    }

    secs := int64(n / 1_000)
    nanos := (int64(n) - (secs * 1_000)) * 1_000_000
    ts := Timestamp{time.Unix(secs, nanos)}

    *t = ts
    return nil
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
    nanos := t.UnixNano()
    millis := nanos / 1_000_000

    s := fmt.Sprintf("%d", millis)
    buf := bytes.NewBufferString(`"`)
	buf.WriteString(s)
	buf.WriteString(`"`)

    return buf.Bytes(), nil
}
