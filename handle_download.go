package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// handleDownload downloads secrets from Kubernetes and writes them to local files
func handleDownload(path string) error {
	// Check if path exists
	info, err := os.Stat(path)

	if err == nil && info.IsDir() {
		// directory exists - download all secrets in it
		return downloadDirectory(path)
	} else if err == nil && !info.IsDir() {
		// file exists - download just that one
		return downloadFile(path)
	} else if os.IsNotExist(err) {
		// path doesn't exist - infer from path structure
		if strings.HasSuffix(path, ".env") {
			// looks like a file path - download single secret
			return downloadFile(path)
		}
		// looks like a directory - download all in namespace
		return downloadDirectory(path)
	}

	return fmt.Errorf("failed to access path %s: %w", path, err)
}

// downloadFile downloads a single secret based on file path
func downloadFile(filePath string) error {
	// parse namespace and secret name from path
	// expected: secrets/<namespace>/<secretname>.env
	parts := strings.Split(filepath.Clean(filePath), string(filepath.Separator))
	if len(parts) < 3 {
		return fmt.Errorf("invalid secret file path: expected secrets/<namespace>/<secretname>.env, got %s", filePath)
	}

	namespace := parts[len(parts)-2]
	secretNameWithExt := parts[len(parts)-1]
	secretName := strings.TrimSuffix(secretNameWithExt, filepath.Ext(secretNameWithExt))

	fmt.Printf("Downloading secret %s/%s...\n", namespace, secretName)

	// read from Kubernetes
	k8sData, err := secretRead(namespace, secretName)
	if err != nil {
		return fmt.Errorf("failed to read secret %s/%s: %w", namespace, secretName, err)
	}

	// get secret to determine type
	secret, err := getSecretMetadata(namespace, secretName)
	if err != nil {
		return fmt.Errorf("failed to get secret metadata %s/%s: %w", namespace, secretName, err)
	}

	// convert from Kubernetes format back to file format
	fileData, err := FromKubernetesData(string(secret.Type), k8sData)
	if err != nil {
		return fmt.Errorf("failed to convert secret %s/%s: %w", namespace, secretName, err)
	}

	// write to file
	if err := WriteSecretFile(filePath, string(secret.Type), fileData); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	fmt.Printf("Successfully downloaded %s/%s to %s\n", namespace, secretName, filePath)
	return nil
}

// downloads all secrets from a directory/namespace
func downloadDirectory(dirPath string) error {
	// try to load existing secrets first
	secrets, err := LoadSecretsFromPath(dirPath)

	// if no local secrets exist, try to list from Kubernetes
	if err != nil || len(secrets) == 0 {
		// parse namespace from path (e.g., "secrets/infra" -> "infra")
		parts := strings.Split(filepath.Clean(dirPath), string(filepath.Separator))
		if len(parts) < 2 {
			return fmt.Errorf("invalid path: expected secrets/<namespace>, got %s", dirPath)
		}
		namespace := parts[len(parts)-1]

		// list all secrets in namespace from Kubernetes
		fmt.Printf("Listing secrets in namespace %s...\n", namespace)
		k8sSecrets, err := listSecretsInNamespace(namespace)
		if err != nil {
			return fmt.Errorf("failed to list secrets in namespace %s: %w", namespace, err)
		}

		if len(k8sSecrets) == 0 {
			fmt.Printf("No secrets found in namespace %s\n", namespace)
			return nil
		}

		fmt.Printf("Found %d secret(s) in namespace %s\n", len(k8sSecrets), namespace)

		// download each secret from Kubernetes
		var errors []error
		for _, k8sSecret := range k8sSecrets {
			secretName := k8sSecret.Name
			fmt.Printf("Downloading secret %s/%s...\n", namespace, secretName)

			// read from Kubernetes
			k8sData, err := secretRead(namespace, secretName)
			if err != nil {
				fmt.Printf("Warning: download failed for %s/%s: %v\n", namespace, secretName, err)
				errors = append(errors, fmt.Errorf("%s/%s: %w", namespace, secretName, err))
				continue
			}

			// convert from Kubernetes format back to file format
			fileData, err := FromKubernetesData(string(k8sSecret.Type), k8sData)
			if err != nil {
				fmt.Printf("Warning: conversion failed for %s/%s: %v\n", namespace, secretName, err)
				errors = append(errors, fmt.Errorf("%s/%s: %w", namespace, secretName, err))
				continue
			}

			// determine file path
			filePath := filepath.Join("secrets", namespace, secretName+".env")

			// write to file
			if err := WriteSecretFile(filePath, string(k8sSecret.Type), fileData); err != nil {
				fmt.Printf("Warning: write failed for %s/%s: %v\n", namespace, secretName, err)
				errors = append(errors, fmt.Errorf("%s/%s: %w", namespace, secretName, err))
				continue
			}

			fmt.Printf("Successfully downloaded %s/%s to %s\n", namespace, secretName, filePath)
		}

		if len(errors) > 0 {
			fmt.Printf("\nCompleted with %d error(s)\n", len(errors))
		} else {
			fmt.Printf("\nSuccessfully downloaded %d secret(s)\n", len(k8sSecrets))
		}

		return nil
	}

	// download based on existing local files
	var errors []error
	for _, secret := range secrets {
		fmt.Printf("Downloading secret %s/%s...\n", secret.Namespace, secret.Name)

		// read from Kubernetes
		k8sData, err := secretRead(secret.Namespace, secret.Name)
		if err != nil {
			fmt.Printf("Warning: download failed for %s/%s: %v\n", secret.Namespace, secret.Name, err)
			errors = append(errors, fmt.Errorf("%s/%s: %w", secret.Namespace, secret.Name, err))
			continue
		}

		// convert from Kubernetes format back to file format
		fileData, err := FromKubernetesData(secret.Type, k8sData)
		if err != nil {
			fmt.Printf("Warning: conversion failed for %s/%s: %v\n", secret.Namespace, secret.Name, err)
			errors = append(errors, fmt.Errorf("%s/%s: %w", secret.Namespace, secret.Name, err))
			continue
		}

		// determine file path
		filePath := filepath.Join("secrets", secret.Namespace, secret.Name+".env")

		// write to file
		if err := WriteSecretFile(filePath, secret.Type, fileData); err != nil {
			fmt.Printf("Warning: write failed for %s/%s: %v\n", secret.Namespace, secret.Name, err)
			errors = append(errors, fmt.Errorf("%s/%s: %w", secret.Namespace, secret.Name, err))
			continue
		}

		fmt.Printf("Successfully downloaded %s/%s to %s\n", secret.Namespace, secret.Name, filePath)
	}

	if len(errors) > 0 {
		fmt.Printf("\nCompleted with %d error(s)\n", len(errors))
	} else {
		fmt.Printf("\nSuccessfully downloaded %d secret(s)\n", len(secrets))
	}

	return nil
}

// retrieves secret metadata from Kubernetes (just type info)
func getSecretMetadata(namespace, secretName string) (*corev1.Secret, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	ctx := context.Background()
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error reading secret: %w", err)
	}

	return secret, nil
}

// lists all secrets in a Kubernetes namespace
func listSecretsInNamespace(namespace string) ([]corev1.Secret, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("error getting Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	ctx := context.Background()
	secretList, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing secrets: %w", err)
	}

	return secretList.Items, nil
}
