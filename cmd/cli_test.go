package cmd

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRootParams(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Can't get user home directory: %v", err)
	}

	// Check default config and db
	config := rootCmd.PersistentFlags().Lookup("config").Value.String()
	inventory := rootCmd.PersistentFlags().Lookup("db").Value.String()

	if config != home+"/.katyusha/katyusha.yaml" {
		t.Errorf("Default config mismatch shoulb be %s but is %s", home+"/.katyusha/katyusha.yaml", config)
	}

	if inventory != home+"/.katyusha/inventory.db" {
		t.Errorf("Default db location mismatch should be %s but is %s", home+"/.katyusha/inventory.db", inventory)
	}
}

func TestBenchmark(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Version string `json:"version"`
		}{
			Version: "0.1",
		}

		if r.URL.Path == "/" {
			w.Write([]byte("Katyusha"))
		}

		if r.URL.Path == "/version" {
			data, err := json.Marshal(response)
			if err != nil {
				t.Errorf("Could not marshal version endpoint response: %v", err)
			}

			w.Write(data)
		}
	}

	b := bytes.NewBufferString("")

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	cmd := NewBenchmarkCmd()
	cmd.SetOut(b)
	cmd.SetArgs([]string{"--host", server.URL})
	cmd.Execute()

	_, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatalf("Can't read from buffer: %v", err)
	}
}
