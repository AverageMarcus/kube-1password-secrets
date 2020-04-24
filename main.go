package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"git.cloud.cluster.fun/AverageMarcus/kube-1password-secrets/internal/onepassword"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	idAnnotation         = "kube-1password"
	vaultAnnotation      = "kube-1password/vault"
	usernameAnnotation   = "kube-1password/username-key"
	passwordAnnotation   = "kube-1password/password-key"
	secretTextAnnotation = "kube-1password/secret-text-key"
)

func main() {
	opClient, err := buildOpClient()
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	secretsClient := clientset.CoreV1().Secrets(apiv1.NamespaceAll)

	for {
		list, err := secretsClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, s := range list.Items {
			if passwordID, exists := s.ObjectMeta.Annotations[idAnnotation]; exists {
				keys := parseAnnotations(s.ObjectMeta.Annotations)

				vault := keys["vault"]

				item, err := opClient.GetSecret(vault, passwordID)
				if err != nil {
					fmt.Println("[ERROR] Could not get secret", err)
					continue
				}

				if item.Username != "" {
					var username []byte
					base64.StdEncoding.Encode(username, []byte(item.Username))
					s.Data[keys["username"]] = username
				}

				if item.Password != "" {
					var password []byte
					base64.StdEncoding.Encode(password, []byte(item.Password))
					s.Data[keys["password"]] = password
				}

				if item.SecretText != "" {
					var secretText []byte
					base64.StdEncoding.Encode(secretText, []byte(item.SecretText))
					s.Data[keys["secretText"]] = secretText
				}

				if _, err := secretsClient.Update(context.Background(), &s, metav1.UpdateOptions{}); err != nil {
					fmt.Println("[ERROR] Could not update secret value", err)
					continue
				}
			}
		}

		time.Sleep(5 * time.Minute)
	}
}

func buildOpClient() (*onepassword.Client, error) {
	domain, ok := os.LookupEnv("OP_DOMAIN")
	if !ok {
		return nil, fmt.Errorf("OP_DOMAIN not specified")
	}
	email, ok := os.LookupEnv("OP_EMAIL")
	if !ok {
		return nil, fmt.Errorf("OP_EMAIL not specified")
	}
	password, ok := os.LookupEnv("OP_PASSWORD")
	if !ok {
		return nil, fmt.Errorf("OP_PASSWORD not specified")
	}
	secretKey, ok := os.LookupEnv("OP_SECRET_KEY")
	if !ok {
		return nil, fmt.Errorf("OP_SECRET_KEY not specified")
	}

	return onepassword.New(domain, email, password, secretKey)
}

func parseAnnotations(annotations map[string]string) map[string]string {
	keys := map[string]string{
		"username":   "username",
		"password":   "password",
		"secretText": "secretText",
		"vault":      os.Getenv("OP_VAULT"),
	}

	for k, v := range annotations {
		switch k {
		case vaultAnnotation:
			keys["vault"] = v
		case usernameAnnotation:
			keys["username"] = v
		case passwordAnnotation:
			keys["password"] = v
		case secretTextAnnotation:
			keys["secretText"] = v
		}
	}

	return keys
}
