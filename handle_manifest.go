package main

import (
	"fmt"
	"sort"
)

// prints Kubernetes secret manifests for specified path
func handleManifest(path string) error {
	// load secrets from path
	secrets, err := LoadSecretsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}

	if len(secrets) == 0 {
		fmt.Printf("No secrets found in %s\n", path)
		return nil
	}

	// print manifest for each secret
	for i, secret := range secrets {
		if i > 0 {
			fmt.Println("---")
		}

		// convert to Kubernetes format
		k8sData, err := secret.ToKubernetesData()
		if err != nil {
			return fmt.Errorf("failed to convert %s/%s: %w", secret.Namespace, secret.Name, err)
		}

		// print YAML manifest
		fmt.Printf("apiVersion: v1\n")
		fmt.Printf("kind: Secret\n")
		fmt.Printf("metadata:\n")
		fmt.Printf("  name: %s\n", secret.Name)
		fmt.Printf("  namespace: %s\n", secret.Namespace)
		fmt.Printf("type: %s\n", secret.Type)
		fmt.Printf("stringData:\n")

		// sort keys for consistent output
		keys := make([]string, 0, len(k8sData))
		for k := range k8sData {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// print plain text values
		for _, key := range keys {
			value := k8sData[key]
			// quote values for YAML safety
			fmt.Printf("  %s: %q\n", key, value)
		}
	}

	return nil
}
