package onepassword

import (
	"time"
)

type field struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type details struct {
	Fields   []field `json:"fields"`
	Notes    *string `json:"notesPlain;omitempty"`
	Password *string `json:"password;omitempty"`
}

type overview struct {
	Title string `json:"title"`
}

type response struct {
	UUID     string    `json:"uuid"`
	Updated  time.Time `json:"createdAt"`
	Details  details   `json:"details"`
	Overview overview  `json:"overview"`
}
