// Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Repository: https://github.com/gojue/moling-minecraft

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gojue/moling-minecraft/services"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"time"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the configuration of the current service list",
	Long: `Show the configuration of the current service list. You can refer to the configuration file to modify the configuration.
`,
	RunE: ConfigCommandFunc,
}

var (
	initial bool
)

// ConfigCommandFunc executes the "config" command.
func ConfigCommandFunc(command *cobra.Command, args []string) error {
	var err error
	logger := initLogger(mlConfig.BasePath)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	multi := zerolog.MultiLevelWriter(consoleWriter, logger)
	logger = zerolog.New(multi).With().Timestamp().Logger()
	mlConfig.SetLogger(logger)
	logger.Info().Msg("Start to show config")
	ctx := context.WithValue(context.Background(), services.MoLingConfigKey, mlConfig)
	ctx = context.WithValue(ctx, services.MoLingLoggerKey, logger)

	// 当前配置文件检测
	hasConfig := false
	var nowConfig []byte
	nowConfigJson := make(map[string]interface{})
	configFilePath := filepath.Join(mlConfig.BasePath, mlConfig.ConfigFile)
	if nowConfig, err = os.ReadFile(configFilePath); err == nil {
		hasConfig = true
	}
	if hasConfig {
		err = json.Unmarshal(nowConfig, &nowConfigJson)
		if err != nil {
			return fmt.Errorf("Error unmarshaling JSON: %v, payload:%s\n", err, string(nowConfig))
		}
	}

	bf := bytes.Buffer{}
	bf.WriteString("\n{\n")

	// 写入GlobalConfig
	mlConfigJson, err := json.Marshal(mlConfig)
	if err != nil {
		return fmt.Errorf("Error marshaling GlobalConfig: %v\n", err)
	}
	bf.WriteString("\t\"MoLingConfig\":\n")
	bf.WriteString(fmt.Sprintf("\t%s,\n", mlConfigJson))
	first := true
	for srvName, nsv := range services.ServiceList() {
		// 获取服务对应的配置
		cfg, ok := nowConfigJson[string(srvName)].(map[string]interface{})

		srv, err := nsv(ctx)
		if err != nil {
			return err
		}
		// srv Loadconfig
		if ok {
			err = srv.LoadConfig(cfg)
			if err != nil {
				return fmt.Errorf("Error loading config for service %s: %v\n", srv.Name(), err)
			}
		} else {
			logger.Debug().Str("service", string(srv.Name())).Msg("Service not found in config, using default config")
		}
		// srv Init
		err = srv.Init()
		if err != nil {
			return fmt.Errorf("Error initializing service %s: %v\n", srv.Name(), err)
		}
		if !first {
			bf.WriteString(",\n")
		}
		bf.WriteString(fmt.Sprintf("\t\"%s\":\n", srv.Name()))
		bf.WriteString(fmt.Sprintf("\t%s\n", srv.Config()))
		first = false
	}
	bf.WriteString("}\n")
	// 解析原始 JSON 字符串
	var data interface{}
	err = json.Unmarshal(bf.Bytes(), &data)
	if err != nil {
		return fmt.Errorf("Error unmarshaling JSON: %v, payload:%s\n", err, bf.String())
	}

	// 格式化 JSON
	formattedJson, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("Error marshaling JSON: %v\n", err)
	}

	// 如果不存在配置文件
	if !hasConfig {
		logger.Info().Msgf("Configuration file %s does not exist. Creating a new one.", configFilePath)
		err = os.WriteFile(configFilePath, formattedJson, 0644)
		if err != nil {
			return fmt.Errorf("Error writing configuration file: %v\n", err)
		}
		logger.Info().Msgf("Configuration file %s created successfully.", configFilePath)
	}
	logger.Info().Str("config", configFilePath).Msg("Current loaded configuration file path")
	logger.Info().Msg("You can modify the configuration file to change the settings.")
	if !initial {
		logger.Info().Msgf("Configuration details: \n%s\n", formattedJson)
	}
	return nil
}

func init() {
	configCmd.PersistentFlags().BoolVar(&initial, "init", false, fmt.Sprintf("Save configuration to %s", filepath.Join(mlConfig.BasePath, mlConfig.ConfigFile)))
	rootCmd.AddCommand(configCmd)
}
