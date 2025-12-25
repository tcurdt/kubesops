package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// represents a Kubernetes secret
type Secret struct {
	Namespace string            // Kubernetes namespace
	Name      string            // Secret name
	Type      string            // Kubernetes secret type
	Data      map[string]string // Key-value pairs
}

// loads a secret from a file
// handles SOPS decryption, type detection, and env var substitution
func LoadSecretFile(filePath string) (*Secret, error) {
	// extract namespace and secret name from path
	// expected format: secrets/<namespace>/<secretname>
	parts := strings.Split(filepath.Clean(filePath), string(filepath.Separator))
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid secret file path: expected secrets/<namespace>/<secretname>, got %s", filePath)
	}

	namespace := parts[len(parts)-2]
	secretNameWithExt := parts[len(parts)-1]
	secretName := strings.TrimSuffix(secretNameWithExt, filepath.Ext(secretNameWithExt))

	// read file content (with SOPS decryption if needed)
	content, err := readFileWithSOPS(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// parse content
	data, secretType, err := parseSecretContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// perform env var substitution
	data = substituteEnvVars(data)

	return &Secret{
		Namespace: namespace,
		Name:      secretName,
		Type:      secretType,
		Data:      data,
	}, nil
}

// readFileWithSOPS reads a file and decrypts it with SOPS if needed
func readFileWithSOPS(filePath string) (string, error) {
	// first, try to read the file to check if it's SOPS-encrypted
	rawContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// check if file is SOPS-encrypted by looking for SOPS metadata
	if isSOPSEncrypted(string(rawContent)) {
		// Use SOPS to decrypt
		cmd := exec.Command("sops", "--decrypt", filePath)

		// pass through SOPS_AGE_KEY environment variable if set
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("SOPS decryption failed: %w\nOutput: %s", err, string(output))
		}
		return string(output), nil
	}

	// not encrypted, return as-is
	return string(rawContent), nil
}

// isSOPSEncrypted checks if content is SOPS-encrypted
func isSOPSEncrypted(content string) bool {
	// check for SOPS metadata markers
	return strings.Contains(content, "sops_") ||
		strings.Contains(content, "sops:") ||
		strings.Contains(content, "ENC[AES256_GCM,")
}

// parseSecretContent parses the file content and extracts type and data
func parseSecretContent(content string) (map[string]string, string, error) {
	data := make(map[string]string)
	secretType := "Opaque" // Default type

	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// check first line for type comment
		if lineNum == 1 && strings.HasPrefix(line, "#") {
			if typeMatch := extractTypeFromComment(line); typeMatch != "" {
				secretType = typeMatch
				continue
			}
		}

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid line %d: %s (expected KEY=VALUE format)", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		data[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading content: %w", err)
	}

	// map type aliases to Kubernetes types
	secretType = mapSecretType(secretType)

	return data, secretType, nil
}

// extractTypeFromComment extracts secret type from shebang-style comment
// example: "# type=docker-registry" returns "docker-registry"
func extractTypeFromComment(line string) string {
	re := regexp.MustCompile(`^#\s*type\s*=\s*(.+)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// mapSecretType maps common type aliases to full Kubernetes type names
func mapSecretType(secretType string) string {
	switch secretType {
	case "docker-registry":
		return "kubernetes.io/dockerconfigjson"
	case "tls":
		return "kubernetes.io/tls"
	case "basic-auth":
		return "kubernetes.io/basic-auth"
	case "ssh-auth":
		return "kubernetes.io/ssh-auth"
	case "generic", "opaque", "":
		return "Opaque"
	default:
		// allow full kubernetes type names or custom types
		return secretType
	}
}

// substituteEnvVars performs environment variable substitution on values
// supports ${VAR} and $VAR syntax
func substituteEnvVars(data map[string]string) map[string]string {
	result := make(map[string]string)

	for key, value := range data {
		result[key] = os.Expand(value, os.Getenv)
	}

	return result
}

// loads all secrets from a path (file or directory)
func LoadSecretsFromPath(path string) ([]*Secret, error) {
	// Default to "secrets" if no path provided
	if path == "" {
		path = "secrets"
	}

	// check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path %s does not exist: %w", path, err)
	}

	var files []string

	if info.IsDir() {
		// walk directory and find all .env files
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".env") {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
		}
	} else {
		// single file
		files = append(files, path)
	}

	// load all secret files
	var secrets []*Secret
	for _, file := range files {
		secret, err := LoadSecretFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// writes a secret to a file
// only writes static values (no env var references)
func WriteSecretFile(filePath string, secretType string, data map[string]string) error {
	// create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// write type comment if not generic/opaque
	if secretType != "Opaque" && secretType != "generic" {
		// map back to alias if possible
		typeAlias := secretType
		switch secretType {
		case "kubernetes.io/dockerconfigjson":
			typeAlias = "docker-registry"
		case "kubernetes.io/tls":
			typeAlias = "tls"
		case "kubernetes.io/basic-auth":
			typeAlias = "basic-auth"
		case "kubernetes.io/ssh-auth":
			typeAlias = "ssh-auth"
		}
		if _, err := fmt.Fprintf(writer, "# type=%s\n", typeAlias); err != nil {
			return fmt.Errorf("failed to write type comment: %w", err)
		}
	}

	// write key-value pairs (sorted for consistency)
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	// sort keys
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		value := data[key]
		// quote values that contain spaces or special characters
		if strings.ContainsAny(value, " \t\n\r\"'$") {
			value = fmt.Sprintf("%q", value)
		}
		if _, err := fmt.Fprintf(writer, "%s=%s\n", key, value); err != nil {
			return fmt.Errorf("failed to write key-value pair: %w", err)
		}
	}

	return nil
}

// converts Secret.Data to Kubernetes format
// for docker-registry, converts docker-* KVs to .dockerconfigjson
func (s *Secret) ToKubernetesData() (map[string]string, error) {
	if s.Type == "kubernetes.io/dockerconfigjson" {
		jsonData, err := BuildDockerConfigJSON(s.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to build docker config: %w", err)
		}
		return map[string]string{
			".dockerconfigjson": jsonData,
		}, nil
	}
	return s.Data, nil
}

// converts Kubernetes secret data to Secret.Data format
// for docker-registry, converts .dockerconfigjson to docker-* KVs
func FromKubernetesData(secretType string, k8sData map[string]string) (map[string]string, error) {
	if secretType == "kubernetes.io/dockerconfigjson" {
		if dockerConfigJSON, ok := k8sData[".dockerconfigjson"]; ok {
			return ParseDockerConfigJSON(dockerConfigJSON)
		}
		return nil, fmt.Errorf("docker-registry secret missing .dockerconfigjson key")
	}
	return k8sData, nil
}
