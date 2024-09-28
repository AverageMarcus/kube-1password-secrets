package onepassword

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Client is the 1Password client
type Client struct{}

// Secret contains the credentials from a 1Password secret
type Secret struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	SecretText string `json:"secretText"`
}

// New authenticates with the provided values and returns a new 1Password client
func New() (*Client, error) {
	client := &Client{}
	if err := client.authenticate(); err != nil {
		return nil, err
	}
	return client, nil
}

func (op *Client) authenticate() error {
	cmd := exec.Command("op", "user", "get", "--me")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Cannot verify auth: %s\n%s", err, output)
	}
	return nil
}

func (op Client) runCmd(args ...string) ([]byte, error) {
	args = append(args, "--format=json")
	cmd := exec.Command("op", args...)
	res, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error calling 1Password: %s\n%s", err, res)
	}
	return res, err
}

// GetSecret returns the values from the secret stored in 1Password with a UUID matching the secretID
func (op *Client) GetSecret(vault, secretID string) (*Secret, error) {
	res, err := op.runCmd("item", "get", secretID, "--reveal", fmt.Sprintf("--vault=%s", vault))
	if err != nil {
		return nil, err
	}
	item := response{}
	if err := json.Unmarshal(res, &item); err != nil {
		return nil, err
	}

	secret := &Secret{
		ID:         item.UUID,
		Title:      item.Title,
		Username:   "",
		Password:   "",
		SecretText: "",
	}

	if len(item.Fields) > 0 {
		for _, field := range item.Fields {
			switch field.Name {
			case "username":
				secret.Username = field.Value
			case "password":
				secret.Password = field.Value
			case "notesPlain":
				secret.SecretText = field.Value
			}
		}
	}

	return secret, nil
}
