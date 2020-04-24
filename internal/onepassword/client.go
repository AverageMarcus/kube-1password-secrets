package onepassword

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

// Client is the 1Password client
type Client struct {
	Domain    string
	Email     string
	Password  string
	SecretKey string
	Session   string
}

// Secret contains the credentials from a 1Password secret
type Secret struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	SecretText string `json:"secretText"`
}

// New authenticates with the provided values and returns a new 1Password client
func New(domain string, email string, password string, secretKey string) (*Client, error) {
	client := &Client{
		Domain:    domain,
		Email:     email,
		Password:  password,
		SecretKey: secretKey,
	}
	if err := client.authenticate(); err != nil {
		return nil, err
	}
	return client, nil
}

func (op *Client) authenticate() error {
	cmd := exec.Command("op", "signin", op.Domain, op.Email, op.SecretKey, "--output=raw")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("Cannot attach to stdin: %s", err)
	}
	go func() {
		defer stdin.Close()
		if _, err := io.WriteString(stdin, fmt.Sprintf("%s\n", op.Password)); err != nil {
			log.Println("[Error]", err)
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Cannot signin: %s\n%s", err, output)
	}
	op.Session = strings.Trim(string(output), "\n")
	return nil
}

func (op Client) runCmd(args ...string) ([]byte, error) {
	args = append(args, fmt.Sprintf("--session=%s", op.Session))
	cmd := exec.Command("op", args...)
	res, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("error calling 1Password: %s\n%s", err, res)
	}
	return res, err
}

// GetSecret returns the values from the secret stored in 1Password with a UUID matching the secretID
func (op *Client) GetSecret(vault, secretID string) (*Secret, error) {
	res, err := op.runCmd("get", "item", secretID, fmt.Sprintf("--vault=%s", vault))
	if err != nil {
		return nil, err
	}

	item := response{}
	if err := json.Unmarshal(res, &item); err != nil {
		return nil, err
	}

	secret := &Secret{
		ID:         item.UUID,
		Title:      item.Overview.Title,
		Username:   "",
		Password:   "",
		SecretText: "",
	}

	if len(item.Details.Fields) > 0 {
		for _, field := range item.Details.Fields {
			switch field.Name {
			case "username":
				secret.Username = field.Value
			case "password":
				secret.Password = field.Value
			}
		}
	}

	if item.Details.Password != nil && *item.Details.Password != "" {
		secret.Password = *item.Details.Password
	}

	if item.Details.Notes != nil {
		secret.SecretText = *item.Details.Notes
	}

	return secret, nil
}
