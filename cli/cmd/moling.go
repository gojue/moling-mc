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
	"github.com/gojue/moling/services"
	"github.com/mark3labs/mcp-go/server"
)

type MoLingServer struct {
	ctx      context.Context
	server   *server.MCPServer
	services []services.Service
}

func NewMoLingServer(ctx context.Context, services []services.Service) (*MoLingServer, error) {
	mcpServer := server.NewMCPServer(
		MCPServerName,
		GitVersion,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
		server.WithPromptCapabilities(true),
	)
	ms := &MoLingServer{
		ctx:      ctx,
		server:   mcpServer,
		services: services,
	}
	err := ms.init()
	return ms, err
}

func (m *MoLingServer) init() error {
	var err error
	for _, srv := range m.services {
		err = m.loadService(srv)
	}
	return err
}

func (m *MoLingServer) loadService(srv services.Service) error {

	// Add resources
	for r, rhf := range srv.Resources() {
		m.server.AddResource(r, rhf)
	}

	// Add Resource Templates
	for rt, rthf := range srv.ResourceTemplates() {
		m.server.AddResourceTemplate(rt, rthf)
	}

	// Add Tools
	m.server.AddTools(srv.Tools()...)

	// Add Notification Handlers
	for n, nhf := range srv.NotificationHandlers() {
		m.server.AddNotificationHandler(n, nhf)
	}

	// Add Prompts
	for _, pe := range srv.Prompts() {
		// Add Prompt
		m.server.AddPrompt(pe.Prompt(), pe.Handler())
	}
	return nil
}

func (s *MoLingServer) Serve() error {
	return server.ServeStdio(s.server)
}
