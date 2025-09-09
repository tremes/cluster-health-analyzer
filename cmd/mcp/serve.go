package mcp

import (
	"log/slog"

	"github.com/openshift/cluster-health-analyzer/pkg/mcp"
	"github.com/spf13/cobra"
)

var MCPCmd = mcpCmd()

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP server providing tools reporting the cluster health data",
		Run: func(cmd *cobra.Command, args []string) {
			readTestData, err := cmd.Flags().GetBool("read-test-data")
			if err != nil {
				slog.Error("Failed to get the read-test-data argument", "error", err)
				return
			}

			incidentsTool := mcp.NewIncidentsTool(readTestData)
			mcpServer := mcp.NewMCPSSEServer("cluster-health-mcp-server", "0.0.1", ":8085")
			mcpServer.RegisterTool(incidentsTool.Tool, incidentsTool.IncidentsHandler)

			err = mcpServer.Start()
			if err != nil {
				slog.Error("Failed to start the MCP server", "error", err)
				return
			}
		},
	}

	cmd.Flags().Bool("read-test-data", false, "flag to enable reading the test data from configmap")
	return cmd
}
