package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/nonozone/MailCli/internal/web"
	"github.com/spf13/cobra"
)

type webRunOptions struct {
	ConfigPath     string
	IndexPath      string
	OperationsPath string
	Host           string
	Port           int
	NoOpen         bool
}

var runWebServerFunc = runWebServer

func newWebCmd() *cobra.Command {
	var opts webRunOptions
	opts.Host = "127.0.0.1"
	opts.Port = 5566

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the local-only MailCLI Web control panel",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !web.IsLocalHost(opts.Host) {
				return fmt.Errorf("host must be localhost for local web mode")
			}
			return runWebServerFunc(cmd.Context(), opts, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.IndexPath, "index", "", "local index file path")
	cmd.Flags().StringVar(&opts.OperationsPath, "operations", "", "operations log path")
	cmd.Flags().StringVar(&opts.Host, "host", opts.Host, "bind host; v1 allows localhost only")
	cmd.Flags().IntVar(&opts.Port, "port", opts.Port, "bind port; pass 0 to choose a random available port")
	cmd.Flags().BoolVar(&opts.NoOpen, "no-open", false, "do not open the browser automatically")
	return cmd
}

func runWebServer(ctx context.Context, opts webRunOptions, out io.Writer) error {
	token, err := web.GenerateToken()
	if err != nil {
		return err
	}
	server, err := web.NewServer(web.Options{
		ConfigPath:     opts.ConfigPath,
		IndexPath:      opts.IndexPath,
		OperationsPath: opts.OperationsPath,
		Token:          token,
		Version:        Version,
	})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", opts.Host, opts.Port))
	if err != nil {
		return fmt.Errorf("start local web listener: %w", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("http://%s:%d/?token=%s", opts.Host, addr.Port, server.Token())
	fmt.Fprintf(out, "MailCLI Web: %s\n", url)
	if !opts.NoOpen {
		_ = openBrowser(url)
	}

	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
