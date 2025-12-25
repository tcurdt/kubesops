package main

import (
	"fmt"
)

func FromOnsiteSecret(secret *Secret) (map[string]string, error) {
	return secret.Data, nil
}

func FromRemoteSecret(namespace, name, secretType string) (map[string]string, error) {
	k8sData, err := secretRead(namespace, name)
	if err != nil {
		return nil, err
	}

	remoteData, err := FromKubernetesData(secretType, k8sData)
	if err != nil {
		return nil, fmt.Errorf("conversion failed: %w", err)
	}

	return remoteData, nil
}

// upload is the unified function that handles both diff and upload operations
// force: if true, upload even if no changes detected
// doit: if true, actually perform the upload; if false, just show what would be done (dry-run)
// verbose: if true, show full values in diff output
func upload(path string, force, doit, verbose bool) error {
	secrets, err := LoadSecretsFromPath(path)
	if err != nil {
		return fmt.Errorf("failed to load secrets: %w", err)
	}

	if len(secrets) == 0 {
		fmt.Printf("no secrets found in %s\n", path)
		return nil
	}

	secretsChanged := 0
	keysChanged := 0
	var errors []error

	for _, secret := range secrets {
		secretName := secret.Namespace + "/" + secret.Name
		differences := 0

		onsiteMap, err := FromOnsiteSecret(secret)
		if err != nil {
			fmt.Printf("warning: failed to read onsite secret %s: %v\n", secretName, err)
			errors = append(errors, fmt.Errorf("%s: %w", secretName, err))
			continue
		}

		remoteMap, err := FromRemoteSecret(secret.Namespace, secret.Name, secret.Type)

		if err != nil {
			fmt.Printf("secret %s is missing\n", secretName)
			differences = 1 // treat missing secret as a change
		} else {
			differences = compareSecretsRemote(secretName, onsiteMap, remoteMap, verbose)
		}

		if differences > 0 {
			secretsChanged++
			keysChanged += differences
		}

		// upload if forced or if there are changes and doit is true
		if force || (differences > 0 && doit) {
			if doit {
				fmt.Printf("uploading secret %s...\n", secretName)

				k8sData, err := secret.ToKubernetesData()
				if err != nil {
					fmt.Printf("warning: conversion failed for %s: %v\n", secretName, err)
					errors = append(errors, fmt.Errorf("%s: %w", secretName, err))
					continue
				}

				if err := secretWrite(secret.Namespace, secret.Name, secret.Type, k8sData); err != nil {
					fmt.Printf("warning: upload failed for %s: %v\n", secretName, err)
					errors = append(errors, fmt.Errorf("%s: %w", secretName, err))
					continue
				}
			}
		}
	}

	fmt.Printf("\n")

	fmt.Printf("secrets changed: %d\n", secretsChanged)
	fmt.Printf("keys changed: %d\n", keysChanged)

	if len(errors) > 0 {
		fmt.Printf("completed with %d error(s)\n", len(errors))
	}

	return nil
}

func handleUpload(path string, force, doit, verbose bool) error {
	return upload(path, force, doit, verbose)
}
