package cmd

import (
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
