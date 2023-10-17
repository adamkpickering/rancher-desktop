/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package config handles all the config-related parts of rdctl
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	p "github.com/rancher-sandbox/rancher-desktop/src/go/rdctl/pkg/paths"
	"github.com/spf13/cobra"
)

// The CLIConfig struct is used to store the json data read from the config file.
type CLIConfig struct {
	User     string
	Password string
	Port     int
}

// ConnectionInfo stores the parameters needed to connect to an HTTP server
type ConnectionInfo struct {
	User     string
	Password string
	Host     string
	Port     string
}

var (
	connectionSettings ConnectionInfo

	configPath string
	// DefaultConfigPath - used to differentiate not being able to find a user-specified config file from the default
	DefaultConfigPath string
)

var ErrMainProcessNotRunning = errors.New("main process not running")

// DefineGlobalFlags sets up the global flags, available for all sub-commands
func DefineGlobalFlags(rootCmd *cobra.Command) {
	var configDir string
	var err error
	if runtime.GOOS == "linux" && isWSLDistro() {
		if configDir, err = wslifyConfigDir(); err != nil {
			log.Fatalf("Can't get WSL config-dir: %v", err)
		}
		configDir = filepath.Join(configDir, "rancher-desktop")
	} else {
		paths, err := p.GetPaths()
		if err != nil {
			log.Fatalf("failed to get paths: %s", err)
		}
		configDir = paths.AppHome
	}
	DefaultConfigPath = filepath.Join(configDir, "rd-engine.json")
	rootCmd.PersistentFlags().StringVar(&configPath, "config-path", "", fmt.Sprintf("config file (default %s)", DefaultConfigPath))
	rootCmd.PersistentFlags().StringVar(&connectionSettings.User, "user", "", "overrides the user setting in the config file")
	rootCmd.PersistentFlags().StringVar(&connectionSettings.Host, "host", "", "default is 127.0.0.1; most useful for WSL")
	rootCmd.PersistentFlags().StringVar(&connectionSettings.Port, "port", "", "overrides the port setting in the config file")
	rootCmd.PersistentFlags().StringVar(&connectionSettings.Password, "password", "", "overrides the password setting in the config file")
}

func PrototypeGetConnectionInfo() (*ConnectionInfo, error) {
	// Create default *ConnectionInfo
	connectionInfo := &ConnectionInfo{
		Host: "127.0.0.1",
	}

	// overlay config file
	configFilePresent := true
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	content, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		configFilePresent = false
	} else if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", configPath, err)
	} else {
		var fileSettings CLIConfig
		if err = json.Unmarshal(content, &fileSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config file %q: %w", err)
		}
		if fileSettings.Port != 0 {
			connectionInfo.Port = strconv.Itoa(fileSettings.Port)
		}
		if fileSettings.User != "" {
			connectionInfo.User = fileSettings.User
		}
		if fileSettings.Password != "" {
			connectionInfo.Password = fileSettings.Password
		}
	}

	// Overwrite connectionInfo values with any user-specified values
	if connectionSettings.Host != "" {
		connectionInfo.Host = connectionSettings.Host
	}
	if connectionSettings.Port != "" {
		connectionInfo.Port = connectionSettings.Port
	}
	if connectionSettings.User != "" {
		connectionInfo.User = connectionSettings.User
	}
	if connectionSettings.Password != "" {
		connectionInfo.Password = connectionSettings.Password
	}

	if sufficientConnectionInfo(connectionInfo) {
		return connectionInfo, nil
	}
	if configPath == DefaultConfigPath && !configFilePresent {
		return nil, ErrMainProcessNotRunning
	}
}

func sufficientConnectionInfo(connectionInfo *ConnectionInfo) bool {
	return connectionSettings.Port == "" || connectionSettings.User == "" || connectionSettings.Password == ""
}

// func readConfigFromFile() (*ConnectionInfo, error) {
// 	if configPath == "" {
// 		configPath = DefaultConfigPath
// 	}
// 	content, err := os.ReadFile(configPath)
// 	if err != nil {
// 		if configPath == DefaultConfigPath && errors.Is(err, os.ErrNotExist) {
// 			return nil, ErrMainProcessNotRunning
// 		}
// 		return nil, fmt.Errorf("failed to read config file %q: %w", configPath, err)
// 	}
// 	var fileSettings CLIConfig
// 	if err = json.Unmarshal(content, &fileSettings); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal config file %q: %w", err)
// 	}
// }

// GetConnectionInfo returns the connection info if it has it, and an error message explaining why
// it isn't available if it doesn't have it.
// So if the user runs an `rdctl` command after a factory reset, there is no config file (in the default location),
// but it might not be necessary. So only use the error message for the missing file if it is actually needed.
func GetConnectionInfo() (*ConnectionInfo, error) {
	isImmediateError, err := finishConnectionSettings()
	if err != nil && (isImmediateError || insufficientConnectionInfo()) {
		return nil, err
	}
	return &connectionSettings, nil
}

func finishConnectionSettings() (bool, error) {
	if configPath == "" {
		configPath = DefaultConfigPath
	}
	if connectionSettings.Host == "" {
		connectionSettings.Host = "127.0.0.1"
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		// If the default config file isn't available, it might not have been created yet,
		// so don't complain if we don't need it.
		// But if the user specified their own --config-path and it's not readable, complain immediately.
		return configPath != DefaultConfigPath, err
	}

	var settings CLIConfig
	if err = json.Unmarshal(content, &settings); err != nil {
		return configPath != DefaultConfigPath, fmt.Errorf("error in config file %q: %w", configPath, err)
	}

	if connectionSettings.User == "" {
		connectionSettings.User = settings.User
	}
	if connectionSettings.Password == "" {
		connectionSettings.Password = settings.Password
	}
	if connectionSettings.Port == "" {
		connectionSettings.Port = strconv.Itoa(settings.Port)
	}
	return false, nil
}

func insufficientConnectionInfo() bool {
	return connectionSettings.Port == "" || connectionSettings.User == "" || connectionSettings.Password == ""
}

// determines if we are running in a wsl linux distro
// by checking for availability of wslpath and see if it's a symlink
func isWSLDistro() bool {
	fi, err := os.Lstat("/bin/wslpath")
	if os.IsNotExist(err) {
		return false
	}
	return fi.Mode()&os.ModeSymlink == os.ModeSymlink
}

func getAppDataPath() (string, error) {
	var outBuf bytes.Buffer
	// changes the codepage to 65001 which is UTF-8
	subCommand := `chcp 65001 >nul & echo %APPDATA%`
	cmd := exec.Command("cmd.exe", "/c", subCommand)
	cmd.Stdout = &outBuf
	// We are intentionally not using CombinedOutput and
	// excluding the stderr since it could contain some
	// warnings when rdctl is triggered from a non WSL mounted directory
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimRight(outBuf.String(), "\r\n"), nil
}

func wslifyConfigDir() (string, error) {
	path, err := getAppDataPath()
	if err != nil {
		return "", err
	}
	var outBuf bytes.Buffer
	cmd := exec.Command("/bin/wslpath", path)
	cmd.Stdout = &outBuf
	if err = cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimRight(outBuf.String(), "\r\n"), err
}
