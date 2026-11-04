package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Atonolo/AtlasMail/internal/model"
	"github.com/Atonolo/AtlasMail/internal/service"
	"github.com/Atonolo/AtlasMail/internal/store"
	"github.com/Atonolo/AtlasMail/internal/web"
)

const smokeToken = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"

func TestServiceProcess(t *testing.T) {
	if os.Getenv("ATLAS_MAIL_TEST_PROCESS") != "1" {
		return
	}
	directory, err := store.DataDirectory()
	if err != nil {
		t.Fatal(err)
	}
	assets, err := fs.Sub(uiFiles, "src/ui")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := web.NewHandler(web.Config{
		Service: service.New(store.New(directory)), Assets: assets, Token: smokeToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("http://" + listener.Addr().String())
	if err := serveUntilSignal(listener, handler); err != nil {
		t.Fatal(err)
	}
}

func TestServiceSurvivesProcessRestart(t *testing.T) {
	directory := t.TempDir()
	address, stop := startServiceProcess(t, directory)
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(address + "/")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("embedded UI failed: %d", response.StatusCode)
	}
	response, err = client.Get(address + "/api/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatal("unauthenticated process API was accepted")
	}
	request, err := http.NewRequest(http.MethodPost, address+"/api/v1/accounts",
		strings.NewReader(`{"label":"Process smoke","address":"process@example.invalid","protocol":"jmap"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+smokeToken)
	request.Header.Set("Content-Type", "application/json")
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		Data model.Account `json:"data"`
	}
	err = json.NewDecoder(response.Body).Decode(&created)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusCreated {
		t.Fatalf("process did not save account: %d, %v", response.StatusCode, err)
	}
	stop()

	address, _ = startServiceProcess(t, directory)
	request, err = http.NewRequest(http.MethodGet, address+"/api/v1/status", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+smokeToken)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var status struct {
		Data service.Status `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if len(status.Data.Accounts) != 1 || status.Data.Accounts[0] != created.Data {
		t.Fatalf("stable account identity lost on process restart: %#v", status)
	}
	if _, err := os.Stat(filepath.Join(directory, "state-v1.json.lock")); !os.IsNotExist(err) {
		t.Fatalf("write lock not released before process stop: %v", err)
	}
}

func startServiceProcess(t *testing.T, directory string) (string, func()) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	command := exec.CommandContext(ctx, executable, "-test.run=^TestServiceProcess$")
	command.Env = append(os.Environ(), "ATLAS_MAIL_TEST_PROCESS=1", "ATLAS_MAIL_DATA_DIR="+directory)
	command.Stderr = os.Stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			cancel()
			_ = command.Wait() // CommandContext intentionally terminates this test child.
		}
	}
	t.Cleanup(stop)
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || !strings.HasPrefix(scanner.Text(), "http://127.0.0.1:") {
		t.Fatalf("process did not report readiness: %s, %v", scanner.Text(), scanner.Err())
	}
	return scanner.Text(), stop
}
