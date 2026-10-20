package models

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kwkkd/arcana/cmd/installer/checker"
	"github.com/kwkkd/arcana/cmd/installer/files"
	"github.com/kwkkd/arcana/cmd/installer/state"
	"github.com/kwkkd/arcana/cmd/installer/wizard/controller"
	"github.com/kwkkd/arcana/cmd/installer/wizard/styles"
	"github.com/kwkkd/arcana/cmd/installer/wizard/window"
)

func TestMainMenu_LoadItems_HighlightsMaintenanceWhenSomethingCanBeUpdated(t *testing.T) {
	for _, tc := range []struct {
		name    string
		checker checker.CheckResult
		want    bool
	}{
		{"a worker image on the host awaits an update", checker.CheckResult{WorkerImageExists: true}, true},
		{"an installed compose stack awaits an update", checker.CheckResult{PentagiInstalled: true}, true},
		{"nothing on the host is updatable", checker.CheckResult{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envPath := filepath.Join(t.TempDir(), ".env")
			if err := os.WriteFile(envPath, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			st, err := state.NewState(envPath)
			if err != nil {
				t.Fatal(err)
			}
			c := controller.NewController(st, files.NewFiles(), tc.checker)

			items := NewMainMenuHandler(c, styles.New(), window.New()).LoadItems()

			i := slices.IndexFunc(items, func(item ListItem) bool { return item.ID == MaintenanceScreen })
			if i < 0 {
				t.Fatal("maintenance is not offered")
			}
			if items[i].Highlighted != tc.want {
				t.Errorf("maintenance highlighted = %v, want %v", items[i].Highlighted, tc.want)
			}
		})
	}
}
