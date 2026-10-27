package main

import (
	"fmt"
	"net"
	"os"

	"github.com/sokinpui/coder/internal/config"
	"github.com/sokinpui/coder/internal/server"
	"github.com/sokinpui/coder/pkg/version"
	"github.com/spf13/cobra"
)

var (
	serverPort   int
	serverSocket string
	wsMode       bool
	stdioMode    bool
	runModel     string
	runProtocol  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "coder-server [flags]",
		Short:   "Headless JSON-RPC and WebSocket server for Coder",
		Long:    "coder-server runs the Coder backend as a headless service over stdio, TCP, Unix domain socket, or WebSocket/HTTP.",
		Version: version.Get(),
		Example: `  coder-server
  coder-server --port 9005
  coder-server --ws --port 9005
  coder-server --socket /tmp/coder.sock`,
		Run: func(cmd *cobra.Command, args []string) {
			runServer(cmd)
		},
	}

	rootCmd.Flags().IntVarP(&serverPort, "port", "p", 0, "Listen on TCP port")
	rootCmd.Flags().StringVarP(&serverSocket, "socket", "s", "", "Listen on Unix domain socket path")
	rootCmd.Flags().BoolVar(&wsMode, "ws", false, "Run with WebSocket/HTTP support (default port 9005 if --port not specified)")
	rootCmd.Flags().BoolVar(&stdioMode, "stdio", false, "Run over stdio (default if no port, socket, or ws specified)")
	rootCmd.Flags().StringVarP(&runModel, "model", "m", "", "Override model for generation")
	rootCmd.Flags().StringVarP(&runProtocol, "protocol", "P", "responses", "Protocol to use: chat or responses")

	_ = rootCmd.RegisterFlagCompletionFunc("protocol", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"chat", "responses"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	if cmd.Flags().Changed("protocol") && runProtocol != "" {
		cfg.Server.Protocol = runProtocol
	}
	if runModel != "" {
		cfg.Coder.ModelCode = runModel
		cfg.Agent.ModelCode = runModel
	}

	srv := server.New(cfg)

	if wsMode {
		port := serverPort
		if port == 0 {
			port = 9005
		}
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on port %d: %v\n", port, err)
			os.Exit(1)
		}
		fmt.Printf("Coder WebSocket server listening on ws://localhost:%d/ws\n", port)
		if err := srv.ServeHTTP(listener); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if serverPort > 0 {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on port %d: %v\n", serverPort, err)
			os.Exit(1)
		}
		if err := srv.ServeListener(listener); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if serverSocket != "" {
		listener, err := net.Listen("unix", serverSocket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to listen on socket %s: %v\n", serverSocket, err)
			os.Exit(1)
		}
		if err := srv.ServeListener(listener); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := srv.ServeStdio(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
