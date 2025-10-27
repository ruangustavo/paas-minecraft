package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type InfraredService struct {
	configPath    string
	dockerService *DockerService
}

type ProxyConfig struct {
	Domains   []string `yaml:"domains"`
	Addresses []string `yaml:"addresses"`
}

func NewInfraredService(configPath string, dockerService *DockerService) *InfraredService {
	return &InfraredService{
		configPath:    configPath,
		dockerService: dockerService,
	}
}

func (is *InfraredService) CreateProxyConfig(serverName string, subdomain string) error {
	config := ProxyConfig{
		Domains:   []string{subdomain},
		Addresses: []string{fmt.Sprintf("%s:25565", serverName)},
	}

	configFile := filepath.Join(is.configPath, fmt.Sprintf("%s.yaml", serverName))

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal proxy config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write proxy config: %w", err)
	}

	if err := is.dockerService.Restart("infrared-proxy"); err != nil {
		return fmt.Errorf("failed to reload infrared container: %w", err)
	}

	return nil
}

func (is *InfraredService) DeleteProxyConfig(serverName string) error {
	configFile := filepath.Join(is.configPath, fmt.Sprintf("%s.yaml", serverName))

	if err := os.Remove(configFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete proxy config: %w", err)
	}

	return nil
}
