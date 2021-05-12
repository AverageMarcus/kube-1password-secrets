package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"
	"time"

	"git.cloud.cluster.fun/AverageMarcus/kube-1password-secrets/internal/onepassword"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	idAnnotation              = "kube-1password"
	vaultAnnotation           = "kube-1password/vault"
	usernameAnnotation        = "kube-1password/username-key"
	passwordAnnotation        = "kube-1password/password-key"
	secretTextAnnotation      = "kube-1password/secret-text-key"
	secretTextParseAnnotation = "kube-1password/secret-text-parse"
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

	for {
		log.Println("[DEBUG] Syncing secrets")
		list, err := clientset.CoreV1().Secrets(apiv1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err)
		}
		for _, s := range list.Items {
			if passwordID, exists := s.ObjectMeta.Annotations[idAnnotation]; exists {
				log.Printf("[INFO] Syncing secret %s with 1Password secret %s\n", s.GetName(), passwordID)
				keys := parseAnnotations(s.ObjectMeta.Annotations)

				vault := keys["vault"]

				item, err := opClient.GetSecret(vault, passwordID)
				if err != nil {
					log.Println("[ERROR] Could not get secret", err)
					continue
				}

				s.Data = make(map[string][]byte)

				if item.Username != "" {
					s.Data[keys["username"]] = []byte(item.Username)
				}

				if item.Password != "" {
					s.Data[keys["password"]] = []byte(item.Password)
				}

				if item.SecretText != "" {
					if s.ObjectMeta.Annotations[secretTextParseAnnotation] != "" {
						// Parse secret text as individual secrets
						lines := strings.Split(item.SecretText, "\n")
						for _, line := range lines {
							parts := strings.Split(line, "=")
							if len(parts) == 2 {
								s.Data[parts[0]] = []byte(parts[1])
							}
						}
					} else {
						s.Data[keys["secretText"]] = []byte(item.SecretText)
					}
				}

				if _, err := clientset.CoreV1().Secrets(s.GetNamespace()).Update(context.Background(), &s, metav1.UpdateOptions{}); err != nil {
					log.Println("[ERROR] Could not update secret value", err)
					continue
				}
			}
		}

		time.Sleep(5 * time.Minute)
	}
}

func buildOpClient() (*onepassword.Client, error) {
	usr, _ := user.Current()
	err := os.Chmod(usr.HomeDir+"/.op", 0700)
	if err != nil {
		panic(err.Error())
	}

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
