package onepassword

import (
	"time"
)

type field struct {
	Name  string `json:"label"`
	Value string `json:"value"`
}

type response struct {
	UUID    string    `json:"id"`
	Updated time.Time `json:"created_at"`
	Fields  []field   `json:"fields"`
	Title   string    `json:"title"`
}
