package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/simon/internal/store"
)

func getStore() store.Storage {
	home, _ := os.UserHomeDir()
	simonDir := filepath.Join(home, ".simon")
	storeLayer, err := store.NewSQLiteStore(
		filepath.Join(simonDir, "metadata.db"),
		filepath.Join(simonDir, "artifacts"),
	)
	if err != nil {
		fmt.Printf("Failed to init store: %v\n", err)
		os.Exit(1)
	}
	return storeLayer
}
