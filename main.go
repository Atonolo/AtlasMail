package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
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

	"github.com/Atonolo/AtlasMail/internal/credentials"
	"github.com/Atonolo/AtlasMail/internal/model"
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
	if len(args) == 1 && args[0] == "credentials-smoke" {
		return credentialSmoke()
	}
	mode := "desktop"
	if len(args) > 1 {
		return errors.New("usage: atlas-mail [server|credentials-smoke]")
	}
	if len(args) == 1 {
		if args[0] != "server" {
			return errors.New("usage: atlas-mail [server|credentials-smoke]")
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

func credentialSmoke() (result error) {
	id, err := model.NewIdentifier()
	if err != nil {
		return err
	}
	store := credentials.New()
	const secret = "atlas-mail-synthetic-keyring-smoke"
	if err := store.Put(id, secret); err != nil {
		return err
	}
	defer func() {
		if result != nil {
			result = errors.Join(result, store.Delete(id))
		}
	}()
	value, err := store.Get(id)
	if err != nil {
		return err
	}
	if value != secret {
		return errors.New("CREDENTIAL_ROUNDTRIP_FAILED")
	}
	if err := store.Delete(id); err != nil {
		return err
	}
	value, err = store.Get(id)
	if err != nil || value != "" {
		return errors.Join(errors.New("CREDENTIAL_DELETE_VERIFICATION_FAILED"), err)
	}
	fmt.Println("Native credential-store write/read/delete smoke passed (synthetic data).")
	return nil
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
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return fmt.Errorf("generate local session token: %w", err)
	}
	accessToken := base64.RawURLEncoding.EncodeToString(token)
	handler, err := web.NewHandler(web.Config{
		Service: mailService,
		Assets:  assets,
		Token:   accessToken,
		Desktop: true,
	})
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local application service: %w", err)
	}
	url := "http://" + listener.Addr().String() + "/desktop#session=" + accessToken
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
