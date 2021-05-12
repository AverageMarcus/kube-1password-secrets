# kube-1password-secrets

Sync secrets from a 1Password vault into Kubernetes secrets.

> **Note:** This should not be considered production grade. It is built on top of the 1Password CLI client which could stop working without warning due to changes made by 1Password.

## Features

* Sync data from items stored in 1Password to Secret resources within Kubernetes
* Rename fields when storing the data in the Kubernetes secret

## Install

1. Create an environment variable with your 1Password credentials:

    ```sh
    cp ./manifests/example.env ./manifests/.env
    ```

1. Deploy to Kubernetes

    ```sh
    make release
    ```

## Usage

1Password secrets are configured using annotation on Secret resources in Kubernetes.

The only required value is the ID of the secret in 1Password. You can get this by looking at the URL when viewing the secret in 1Password, e.g.

> my.1password.com/vaults/123456789qwertyuiop/allitems/**123456example7890**

Example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-secret
  annotations:
    kube-1password: 123456example7890           # [Required] This is the ID of the item within 1Password
    kube-1password/vault: Kubernetes            # The name of the Vault
    kube-1password/username-key: "user"         # The key the username should be saved as in the Secret resource (default: `username`)
    kube-1password/password-key: "pass"         # The key the password should be saved as in the Secret resource (default: `password`)
    kube-1password/secret-text-key: "note"      # The key the secret text should be saved as in the Secret resource (default: `secretText`)
    kube-1password/secret-text-parse: "true"    # Parse the secret texts as individual secret values in format `key=value` (default: ``)
type: Opaque
```

kube-1password-secrets currently supports *Login*, *Secure Note* and *Password* item types in 1Password. Only the **username**, **password** and **notes** fields are retrieved.

## Building from source

With Docker:

```sh
make docker-build
```

Standalone:

```sh
make build
```

## Resources

* [1Password CLI client](https://app-updates.agilebits.com/product_history/CLI)

## Contributing

If you find a bug or have an idea for a new feature please raise an issue to discuss it.

Pull requests are welcomed but please try and follow similar code style as the rest of the project and ensure all tests and code checkers are passing.

Thank you 💛

## License

See [LICENSE](LICENSE)
