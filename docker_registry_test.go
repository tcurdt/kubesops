package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestBuildDockerConfigJSON(t *testing.T) {
	values := map[string]string{
		"docker-server":   "https://ghcr.io",
		"docker-username": "testuser",
		"docker-password": "testpass",
		"docker-email":    "test@example.com",
	}

	result, err := BuildDockerConfigJSON(values)
	if err != nil {
		t.Fatalf("BuildDockerConfigJSON failed: %v", err)
	}

	// parse the JSON to verify structure
	var config DockerConfig
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// verify server exists
	auth, ok := config.Auths["https://ghcr.io"]
	if !ok {
		t.Fatal("expected server https://ghcr.io not found in auths")
	}

	// verify credentials
	if auth.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", auth.Username)
	}
	if auth.Password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", auth.Password)
	}
	if auth.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", auth.Email)
	}

	// verify auth field
	expectedAuth := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	if auth.Auth != expectedAuth {
		t.Errorf("expected auth '%s', got '%s'", expectedAuth, auth.Auth)
	}
}

func TestBuildDockerConfigJSON_NoEmail(t *testing.T) {
	values := map[string]string{
		"docker-server":   "https://ghcr.io",
		"docker-username": "testuser",
		"docker-password": "testpass",
	}

	result, err := BuildDockerConfigJSON(values)
	if err != nil {
		t.Fatalf("BuildDockerConfigJSON failed: %v", err)
	}

	var config DockerConfig
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	auth := config.Auths["https://ghcr.io"]
	if auth.Email != "" {
		t.Errorf("expected empty email, got '%s'", auth.Email)
	}
}

func TestBuildDockerConfigJSON_MissingRequired(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		errMsg string
	}{
		{
			name: "missing server",
			values: map[string]string{
				"docker-username": "testuser",
				"docker-password": "testpass",
			},
			errMsg: "docker-server is required",
		},
		{
			name: "missing username",
			values: map[string]string{
				"docker-server":   "https://ghcr.io",
				"docker-password": "testpass",
			},
			errMsg: "docker-username is required",
		},
		{
			name: "missing password",
			values: map[string]string{
				"docker-server":   "https://ghcr.io",
				"docker-username": "testuser",
			},
			errMsg: "docker-password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildDockerConfigJSON(tt.values)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseDockerConfigJSON(t *testing.T) {
	// create a valid docker config JSON
	auth := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	config := DockerConfig{
		Auths: map[string]DockerAuth{
			"https://ghcr.io": {
				Username: "testuser",
				Password: "testpass",
				Email:    "test@example.com",
				Auth:     auth,
			},
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	result, err := ParseDockerConfigJSON(string(jsonData))
	if err != nil {
		t.Fatalf("ParseDockerConfigJSON failed: %v", err)
	}

	// verify all keys are present
	expected := map[string]string{
		"docker-server":   "https://ghcr.io",
		"docker-username": "testuser",
		"docker-password": "testpass",
		"docker-email":    "test@example.com",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}

func TestParseDockerConfigJSON_NoEmail(t *testing.T) {
	auth := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	config := DockerConfig{
		Auths: map[string]DockerAuth{
			"https://ghcr.io": {
				Username: "testuser",
				Password: "testpass",
				Auth:     auth,
			},
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	result, err := ParseDockerConfigJSON(string(jsonData))
	if err != nil {
		t.Fatalf("ParseDockerConfigJSON failed: %v", err)
	}

	// email should not be present if it was empty
	if _, ok := result["docker-email"]; ok {
		t.Error("docker-email should not be present when empty")
	}
}

func TestParseDockerConfigJSON_Invalid(t *testing.T) {
	_, err := ParseDockerConfigJSON("invalid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	_, err = ParseDockerConfigJSON("{\"auths\": {}}")
	if err == nil {
		t.Error("expected error for empty auths")
	}
}

func TestMergeSecretSources_DockerRegistry(t *testing.T) {
	spec := &SecretSpec{
		Type: "kubernetes.io/dockerconfigjson",
		Operations: []Operation{
			{"docker-server": "https://ghcr.io"},
			{"docker-username": "testuser"},
			{"docker-password": "testpass"},
			{"docker-email": "test@example.com"},
		},
	}

	result, err := MergeSecretSources(spec)
	if err != nil {
		t.Fatalf("MergeSecretSources failed: %v", err)
	}

	// should have only .dockerconfigjson key
	if len(result) != 1 {
		t.Fatalf("expected 1 key, got %d", len(result))
	}

	dockerConfigJSON, ok := result[".dockerconfigjson"]
	if !ok {
		t.Fatal("expected .dockerconfigjson key")
	}

	// verify it's valid JSON
	var config DockerConfig
	if err := json.Unmarshal([]byte(dockerConfigJSON), &config); err != nil {
		t.Fatalf("failed to unmarshal docker config: %v", err)
	}

	// verify contents
	auth, ok := config.Auths["https://ghcr.io"]
	if !ok {
		t.Fatal("expected server https://ghcr.io not found")
	}

	if auth.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", auth.Username)
	}
}

func TestReverseAndWriteSecretSources_DockerRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := tmpDir + "/.env.docker"

	spec := &SecretSpec{
		Type: "kubernetes.io/dockerconfigjson",
		Operations: []Operation{
			{"envfile": envFile},
		},
	}

	// create a docker config JSON (as would come from Kubernetes)
	auth := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	config := DockerConfig{
		Auths: map[string]DockerAuth{
			"https://ghcr.io": {
				Username: "testuser",
				Password: "testpass",
				Email:    "test@example.com",
				Auth:     auth,
			},
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	values := map[string]string{
		".dockerconfigjson": string(jsonData),
	}

	// reverse and write
	err = ReverseAndWriteSecretSources(spec, values)
	if err != nil {
		t.Fatalf("ReverseAndWriteSecretSources failed: %v", err)
	}

	// read the envfile to verify docker-* keys were written
	result, err := ParseDotenvFile(envFile)
	if err != nil {
		t.Fatalf("failed to read envfile: %v", err)
	}

	// verify all docker keys are present
	expected := map[string]string{
		"docker-server":   "https://ghcr.io",
		"docker-username": "testuser",
		"docker-password": "testpass",
		"docker-email":    "test@example.com",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}
