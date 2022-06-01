package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/user"
	"strings"

	"git.cloud.cluster.fun/AverageMarcus/kube-1password-secrets/internal/onepassword"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	fieldManagerName          = "kube-1password-secrets"
	idAnnotation              = "kube-1password"
	vaultAnnotation           = "kube-1password/vault"
	usernameAnnotation        = "kube-1password/username-key"
	passwordAnnotation        = "kube-1password/password-key"
	secretTextAnnotation      = "kube-1password/secret-text-key"
	secretTextParseAnnotation = "kube-1password/secret-text-parse"
)

var (
	opClient  *onepassword.Client
	clientset *kubernetes.Clientset
)

func main() {
	var err error
	opClient, err = buildOpClient()
	if err != nil {
		panic(err.Error())
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	stopper := make(chan struct{})
	defer close(stopper)
	factory := informers.NewSharedInformerFactory(clientset, 0)
	secretInformer := factory.Core().V1().Secrets()
	informer := secretInformer.Informer()
	defer runtime.HandleCrash()
	go factory.Start(stopper)
	if !cache.WaitForCacheSync(stopper, informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { processSecret(obj.(*apiv1.Secret)) },
		UpdateFunc: func(old interface{}, new interface{}) {
			managedFields := new.(*apiv1.Secret).GetManagedFields()
			if len(managedFields) == 0 || managedFields[len(managedFields)-1].Manager != fieldManagerName {
				processSecret(new.(*apiv1.Secret))
			}
		},
		DeleteFunc: func(interface{}) {},
	})

	lister := secretInformer.Lister().Secrets((v1.NamespaceAll))
	secrets, err := lister.List(labels.Everything())

	for _, s := range secrets {
		processSecret(s)
	}

	<-stopper
}

func processSecret(s *apiv1.Secret) {
	if passwordID, exists := s.ObjectMeta.Annotations[idAnnotation]; exists {
		log.Printf("[INFO] Syncing secret %s with 1Password secret %s\n", s.GetName(), passwordID)
		keys := parseAnnotations(s.ObjectMeta.Annotations)

		vault := keys["vault"]

		item, err := opClient.GetSecret(vault, passwordID)
		if err != nil {
			log.Println("[ERROR] Could not get secret", err)
			return
		}

		s.Data = make(map[string][]byte)

		if item.Username != "" {
			s.Data[keys["username"]] = []byte(parseNewlines(item.Username))
		}

		if item.Password != "" {
			s.Data[keys["password"]] = []byte(parseNewlines(item.Password))
		}

		if item.SecretText != "" {
			if s.ObjectMeta.Annotations[secretTextParseAnnotation] != "" {
				// Parse secret text as individual secrets
				lines := strings.Split(item.SecretText, "\n")
				for _, line := range lines {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						s.Data[parts[0]] = []byte(parseNewlines(parts[1]))
					}
				}
			} else {
				s.Data[keys["secretText"]] = []byte(parseNewlines(item.SecretText))
			}
		}

		_, err = clientset.CoreV1().Secrets(s.GetNamespace()).Update(context.Background(), s, metav1.UpdateOptions{FieldManager: fieldManagerName})
		if err != nil {
			log.Println("[ERROR] Could not update secret value", err)
		}
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

func parseNewlines(in string) string {
	return strings.ReplaceAll(in, "\\n", "\n")
}
