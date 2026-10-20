package processor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kwkkd/arcana/cmd/installer/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompose_IsComposeMissingError_RecognisesOnlyAMissingFile(t *testing.T) {
	c := &composeOperationsImpl{}
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"no error", nil, false},
		{"a file that does not exist", os.ErrNotExist, true},
		{"a path that does not exist", &os.PathError{Op: "open", Path: "/path/to/file", Err: os.ErrNotExist}, true},
		{"docker compose naming a missing file", errors.New("open docker-compose-graphiti.yml: no such file or directory"), true},
		{"permission denied", os.ErrPermission, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, c.isComposeMissingError(tc.err))
		})
	}
}

func TestCompose_PerformStackOperation_SkipsATearDownWithNoComposeFile(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(envPath, []byte("GRAPHITI_URL=http://localhost:8000\n"), 0o644))
	appState, err := state.NewState(envPath)
	require.NoError(t, err)

	c := newComposeOperations(&processor{state: appState}).(*composeOperationsImpl)
	require.False(t, c.composeFileExists(ProductStackGraphiti), "no docker-compose-graphiti.yml was written")

	opState := testOperationState(t)
	require.NoError(t, c.performStackOperation(t.Context(), ProductStackGraphiti, opState, ProcessorOperationStop, "stop"))
	assert.Empty(t, opState.msgs, "the stop was announced, so compose was about to run without its file")
}
