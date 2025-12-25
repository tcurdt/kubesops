package main

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// secretWrite writes a secret to Kubernetes
// creates the secret if it doesn't exist, updates it if it does
func secretWrite(namespace, secretName, secretType string, values map[string]string) error {
	config, err := getKubeConfig()
	if err != nil {
		return fmt.Errorf("error getting Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating Kubernetes client: %w", err)
	}

	ctx := context.Background()

	// convert map[string]string to map[string][]byte
	secretData := make(map[string][]byte)
	for key, value := range values {
		secretData[key] = []byte(value)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretType(secretType),
		Data: secretData,
	}

	// try to get existing secret
	_, err = clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})

	if err != nil {
		// secret doesn't exist, create it
		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating secret: %w", err)
		}
		fmt.Printf("Created secret %s in namespace %s\n", secretName, namespace)
	} else {
		// secret exists, update it
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("error updating secret: %w", err)
		}
		fmt.Printf("Updated secret %s in namespace %s\n", secretName, namespace)
	}

	return nil
}

// secretRead reads a secret from Kubernetes
func secretRead(namespace, secretName string) (map[string]string, error) {
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

	// convert map[string][]byte to map[string]string
	result := make(map[string]string)
	for key, value := range secret.Data {
		result[key] = string(value)
	}

	return result, nil
}

// getKubeConfig gets the Kubernetes configuration
// priority order:
//  1. In-cluster config
//  2. KUBECONFIG env var (file path)
//  3. KUBECONFIG_DATA env var (content)
//  4. ~/.kube/config (default)
func getKubeConfig() (*rest.Config, error) {
	// try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	var kubeconfigData []byte

	// check KUBECONFIG file path first
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath != "" {
		// try to read the kubeconfig file
		kubeconfigData, err = os.ReadFile(kubeconfigPath)
		if err != nil {
			// if KUBECONFIG is set but unreadable, fall back to KUBECONFIG_DATA
			kubeconfigContent := os.Getenv("KUBECONFIG_DATA")
			if kubeconfigContent != "" {
				kubeconfigData = []byte(kubeconfigContent)
			} else {
				return nil, fmt.Errorf("error loading config file %q: %w", kubeconfigPath, err)
			}
		}
	} else {
		// KUBECONFIG not set, check KUBECONFIG_DATA
		kubeconfigContent := os.Getenv("KUBECONFIG_DATA")
		if kubeconfigContent != "" {
			kubeconfigData = []byte(kubeconfigContent)
		} else {
			// fall back to default kubeconfig location
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("error getting user home directory: %w", err)
			}
			kubeconfigPath = homeDir + "/.kube/config"
			kubeconfigData, err = os.ReadFile(kubeconfigPath)
			if err != nil {
				return nil, fmt.Errorf("error loading config file %q: %w", kubeconfigPath, err)
			}
		}
	}

	config, err = clientcmd.RESTConfigFromKubeConfig(kubeconfigData)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %w", err)
	}

	return config, nil
}
