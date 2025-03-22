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

package cmd

import (
	"context"
	"github.com/gojue/moling/cli/cobrautl"
	"github.com/gojue/moling/services"
	"github.com/spf13/cobra"
	"os"
)

type contextKey string

const (
	CliName            = "moling"
	CliNameZh          = "魔灵"
	CliDescription     = "A local office automation magic assistant, Moling can help you perform file viewing, reading, and other related tasks, as well as assist with command-line operations.\n\n"
	CliDescriptionZh   = "本地办公自动化魔法助手，可以帮你实现文件查看、阅读等操作，也可以帮你执行命令行操作。"
	CliHomepage        = "https://gojue.cc/moling"
	CliAuthor          = "CFC4N <cfc4ncs@gmail.com>"
	CliGithubRepo      = "https://github.com/gojue/moling"
	CliDescriptionLong = `
Moling is a computer-based MCP Server that implements system interaction through operating system APIs, enabling file system operations such as reading, writing, merging, statistics, and aggregation, as well as the ability to execute system commands. It is a dependency-free local office automation assistant.

Requiring no installation of any dependencies, Moling can be run directly and is compatible with multiple operating systems, including Windows, Linux, and MacOS. This eliminates the hassle of dealing with environment conflicts involving Node.js, Python, and other development environments.

Usage:
  moling --allow-dir="/tmp,/home/user"
  moling -h
`
	CliDescriptionLongZh = `Moling（魔灵） 是一个computer-use的MCP Server，基于操作系统API实现了系统交互，可以实现文件系统的读写、合并、统计、聚合等操作，也可以执行系统命令操作。是一个无需任何依赖的本地办公自动化助手。
没有任何安装依赖，直接运行，兼容Windows、Linux、MacOS等操作系统。再也不用苦恼NodeJS、Python等环境冲突等问题了。

Usage:
  moling --allow-dir="/tmp,/home/user"
  moling -h
`
)

var (
	GitVersion = "linux-arm64-202503222008-v0.0.1"
)

// MoLingVersion is a context key for storing the version of MoLing
const (
	MoLingVersion contextKey = "moling_version"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:        CliName,
	Short:      CliDescription,
	SuggestFor: []string{"moling"},

	Long: CliDescriptionLong,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE: mlsCommandFunc,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func usageFunc(c *cobra.Command) error {
	return cobrautl.UsageFunc(c, GitVersion)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetUsageFunc(usageFunc)
	rootCmd.SetHelpTemplate(`{{.UsageString}}`)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Version = GitVersion
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version:\t%s" .Version}}
`)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var allowDir []string
var allowCmd []string

func init() {
	cobra.EnablePrefixMatching = true
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringSliceVar(&allowDir, "allow-dir", []string{"/tmp"}, "allow dir")
	rootCmd.PersistentFlags().StringSliceVar(&allowCmd, "allow-cmd", []string{"ifconfig", "whomai", "ip", "ss", "ls", "pwd", "cat", "echo", "date", "whoami", "uname", "top", "ps", "df", "free", "uptime", "ifconfig", "ip", "netstat", "ping", "traceroute", "curl", "wget", "dig", "nslookup", "ssh", "head"}, "allow commands")
	rootCmd.SilenceUsage = true
}
func mlsCommandFunc(command *cobra.Command, args []string) error {
	ctx := context.WithValue(context.Background(), MoLingVersion, GitVersion)
	var srvs []services.Service

	// FileSystemServer
	fs, err := services.NewFilesystemServer(ctx, services.NewFileSystemConfig(allowDir))
	if err != nil {
		return err
	}
	srvs = append(srvs, fs)

	cs, err := services.NewCommandServer(ctx, services.NewCommandConfig(allowCmd))
	if err != nil {
		return err
	}
	srvs = append(srvs, cs)

	srv, err := NewMoLingServer(ctx, srvs)
	if err != nil {
		return err
	}
	err = srv.Serve()
	return err
}
