package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/Atonolo/AtlasMail/internal/service"
	"github.com/Atonolo/AtlasMail/internal/store"
	"github.com/Atonolo/AtlasMail/internal/web"
)

//go:embed src/ui/*
var uiFiles embed.FS

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Atlas Mail: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	mode := "desktop"
	if len(args) > 1 {
		return errors.New("usage: atlas-mail [server]")
	}
	if len(args) == 1 {
		if args[0] != "server" {
			return errors.New("usage: atlas-mail [server]")
		}
		mode = "server"
	}
	dataDirectory, err := store.DataDirectory()
	if err != nil {
		return err
	}
	mailService := service.New(store.New(dataDirectory))
	assets, err := fs.Sub(uiFiles, "src/ui")
	if err != nil {
		return fmt.Errorf("load embedded UI: %w", err)
	}
	if mode == "server" {
		return runServer(mailService, assets)
	}
	return runDesktop(mailService, assets)
}

func runServer(mailService *service.MailService, assets fs.FS) error {
	port := os.Getenv("ATLAS_MAIL_PORT")
	address, err := web.ListenAddress(port)
	if err != nil {
		return err
	}
	handler, err := web.NewHandler(web.Config{
		Service:      mailService,
		Assets:       assets,
		Token:        os.Getenv("ATLAS_MAIL_WEB_TOKEN"),
		PublicOrigin: os.Getenv("ATLAS_MAIL_PUBLIC_ORIGIN"),
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}
	return serveUntilSignal(listener, handler)
}

func runDesktop(mailService *service.MailService, assets fs.FS) error {
	handler, err := web.NewHandler(web.Config{
		Service: mailService,
		Assets:  assets,
		Desktop: true,
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local application service: %w", err)
	}
	url := "http://" + listener.Addr().String() + "/desktop"
	if err := openBrowser(url); err != nil {
		_ = listener.Close()
		return fmt.Errorf("open system browser: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Atlas Mail is running in your system browser. Press Ctrl+C to stop.")
	return serveUntilSignal(listener, handler)
}

func serveUntilSignal(listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}
	errorsChannel := make(chan error, 1)
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsChannel <- err
		}
	}()
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChannel)
	select {
	case err := <-errorsChannel:
		return fmt.Errorf("HTTP service failed: %w", err)
	case <-signalChannel:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

func openBrowser(address string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", address}
	case "darwin":
		command = "open"
		args = []string{address}
	default:
		command = "xdg-open"
		args = []string{address}
	}
	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("%s is unavailable: %w", command, err)
	}
	process := exec.Command(command, args...)
	if err := process.Start(); err != nil {
		return err
	}
	go func() {
		if err := process.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Atlas Mail could not open the system browser: %v\n", err)
		}
	}()
	return nil
}
