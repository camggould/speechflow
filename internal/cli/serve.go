// `speechflow serve`: launch the embedded HTTP server + UI.
package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/camggould/speechflow/internal/server"
)

func newServeCommand() *cobra.Command {
	var port int
	var openBrowser bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the embedded HTTP server and UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := openStore(cmd)
			if err != nil {
				return Exit(ExitGeneric, "open store: %v", err)
			}
			// We deliberately do not defer s.Close() here so it stays open
			// for the server's lifetime; we close it after Shutdown returns.

			addr := "127.0.0.1:" + strconv.Itoa(port)
			srv := server.New(server.Config{
				Listen: addr,
				Store:  s,
			})

			ln, err := net.Listen("tcp", addr)
			if err != nil {
				_ = s.Close()
				return Exit(ExitGeneric, "listen %s: %v", addr, err)
			}
			url := "http://" + ln.Addr().String() + "/"
			fmt.Fprintf(cmd.OutOrStdout(), "speechflow serve listening on %s\n", url)

			if openBrowser {
				_ = openInBrowser(url)
			}

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					errCh <- err
				} else {
					errCh <- nil
				}
			}()

			select {
			case <-ctx.Done():
				fmt.Fprintln(cmd.OutOrStdout(), "\nshutting down...")
				shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = srv.Shutdown(shut)
				_ = s.Close()
				return <-errCh
			case err := <-errCh:
				_ = s.Close()
				return err
			}
		},
	}
	cmd.Flags().IntVar(&port, "port", 7777, "Port to listen on (always bound to 127.0.0.1)")
	cmd.Flags().BoolVar(&openBrowser, "open", false, "Open the UI in the default browser after starting")
	return cmd
}

// openInBrowser opens url in the OS default browser. Best-effort.
func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported OS for --open: %s", runtime.GOOS)
	}
	return cmd.Start()
}
