package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// represents the structure of .dockerconfigjson
type DockerConfig struct {
	Auths map[string]DockerAuth `json:"auths"`
}

// represents authentication for a single registry
type DockerAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth"`
}

// builds a .dockerconfigjson from docker-* keys
// expects keys: docker-server, docker-username, docker-password, docker-email (optional)
func BuildDockerConfigJSON(values map[string]string) (string, error) {
	server, ok := values["docker-server"]
	if !ok || server == "" {
		return "", fmt.Errorf("docker-server is required for docker-registry secrets")
	}

	username, ok := values["docker-username"]
	if !ok || username == "" {
		return "", fmt.Errorf("docker-username is required for docker-registry secrets")
	}

	password, ok := values["docker-password"]
	if !ok || password == "" {
		return "", fmt.Errorf("docker-password is required for docker-registry secrets")
	}

	email := values["docker-email"] // Optional

	// build auth string (base64 of username:password)
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	config := DockerConfig{
		Auths: map[string]DockerAuth{
			server: {
				Username: username,
				Password: password,
				Email:    email,
				Auth:     auth,
			},
		},
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal docker config: %w", err)
	}

	return string(jsonData), nil
}

// parses .dockerconfigjson and extracts docker-* keys
func ParseDockerConfigJSON(jsonData string) (map[string]string, error) {
	var config DockerConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal docker config: %w", err)
	}

	if len(config.Auths) == 0 {
		return nil, fmt.Errorf("no auths found in docker config")
	}

	// extract the first (and typically only) registry
	var server string
	var auth DockerAuth
	for s, a := range config.Auths {
		server = s
		auth = a
		break
	}

	result := map[string]string{
		"docker-server":   server,
		"docker-username": auth.Username,
		"docker-password": auth.Password,
	}

	if auth.Email != "" {
		result["docker-email"] = auth.Email
	}

	return result, nil
}
