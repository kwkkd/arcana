package anonymizer

import (
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewReplacer_RejectsListsThatDoNotPairUp(t *testing.T) {
	_, err := NewReplacer([]*regexp.Regexp{regexp.MustCompile(`a`)}, nil)
	require.Error(t, err)

	_, err = NewReplacer([]*regexp.Regexp{nil}, []string{"nil"})
	require.Error(t, err)

	r, err := NewReplacer(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "untouched", r.ReplaceString("untouched"))
}

func TestReplacer_ReplaceString_MasksTheGroupThePatternNames(t *testing.T) {
	r, err := NewReplacer(
		[]*regexp.Regexp{regexp.MustCompile(`token=(?P<replace>[a-z0-9]+)`)},
		[]string{"Token"},
	)
	require.NoError(t, err)

	got := r.ReplaceString("a token=abc123 and b token=def456")
	assert.Equal(t, "a token=§*Token*§ and b token=§*Token*§", got)
}

func TestReplacer_ReplaceString_MasksTheWholeMatchWithoutAGroup(t *testing.T) {
	r, err := NewReplacer(
		[]*regexp.Regexp{regexp.MustCompile(`secret-\d+`)},
		[]string{"The Secret"},
	)
	require.NoError(t, err)

	assert.Equal(t, "x §*The Secret*§ y", r.ReplaceString("x secret-42 y"))
}

func TestReplacer_ReplaceString_LeavesTextNoPatternMatches(t *testing.T) {
	r, err := NewReplacer([]*regexp.Regexp{regexp.MustCompile(`zzz`)}, []string{"Z"})
	require.NoError(t, err)

	assert.Equal(t, "nothing to mask", r.ReplaceString("nothing to mask"))
}

func TestReplacer_ReplaceBytes_MasksLikeReplaceString(t *testing.T) {
	r, err := NewReplacer(
		[]*regexp.Regexp{regexp.MustCompile(`秘密=(?P<replace>\d+)`)},
		[]string{"Number"},
	)
	require.NoError(t, err)

	text := "秘密=1234, 끝"
	assert.Equal(t, r.ReplaceString(text), string(r.ReplaceBytes([]byte(text))))
	assert.Equal(t, "秘密=§*Number*§, 끝", r.ReplaceString(text))
}

func TestReplacer_WrapReader_MasksWhatItServes(t *testing.T) {
	r, err := NewReplacer([]*regexp.Regexp{regexp.MustCompile(`hunter2`)}, []string{"Password"})
	require.NoError(t, err)

	out, err := io.ReadAll(r.WrapReader(strings.NewReader("pw hunter2 end")))
	require.NoError(t, err)
	assert.Equal(t, "pw §*Password*§ end", string(out))
}

type failingReader struct{ err error }

func (f failingReader) Read([]byte) (int, error) { return 0, f.err }

func TestReplacer_WrapReader_HandsTheSourceErrorThrough(t *testing.T) {
	sentinel := errors.New("broken source")

	r, err := NewReplacer([]*regexp.Regexp{regexp.MustCompile(`x`)}, []string{"X"})
	require.NoError(t, err)

	_, err = io.ReadAll(r.WrapReader(failingReader{sentinel}))
	assert.ErrorIs(t, err, sentinel)
}
