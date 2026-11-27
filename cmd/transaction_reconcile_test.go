package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kfriede/nab/internal/api"
)

type fakeStore struct {
	gets  []string
	puts  int
	putEr error
	putOK string
}

func (f *fakeStore) GetJSON(_ string, target any) error {
	s := f.gets[0]
	if len(f.gets) > 1 {
		f.gets = f.gets[1:]
	}
	return json.Unmarshal([]byte(s), target)
}
func (f *fakeStore) Put(string, any) ([]byte, error) { f.puts++; return []byte(f.putOK), f.putEr }

var unk = &api.MutationOutcomeUnknownError{Cause: io.ErrUnexpectedEOF}

const empty = `{"transaction":{"id":"t","memo":null,"amount":-1}}`
const set = `{"transaction":{"id":"t","memo":"m","amount":-1}}`

func TestUpdateSuccessNoReconcile(t *testing.T) {
	f := &fakeStore{gets: []string{empty}, putOK: set}
	if _, err := updateTransaction(f, "p", map[string]any{"memo": "m"}); err != nil || f.puts != 1 {
		t.Fatal(err, f.puts)
	}
}
func TestUpdateAmbiguousUnchanged(t *testing.T) {
	f := &fakeStore{gets: []string{empty, empty}, putEr: unk}
	_, err := updateTransaction(f, "p", map[string]any{"memo": "m"})
	var nc *UpdateNotCommittedError
	if !errors.As(err, &nc) || f.puts != 1 || classifyError(err).Code != "UPDATE_NOT_COMMITTED" {
		t.Fatal(err, f.puts)
	}
}
func TestUpdateAmbiguousCommitted(t *testing.T) {
	f := &fakeStore{gets: []string{empty, set}, putEr: unk}
	r, err := updateTransaction(f, "p", map[string]any{"memo": "m"})
	if err != nil || f.puts != 1 || r["transaction"] == nil {
		t.Fatal(err, f.puts)
	}
}
func TestUpdateAmbiguousChangedMismatch(t *testing.T) {
	f := &fakeStore{gets: []string{empty, `{"transaction":{"id":"t","memo":"x","amount":-1}}`}, putEr: unk}
	_, err := updateTransaction(f, "p", map[string]any{"memo": "m"})
	if classifyError(err).Code != "MUTATION_OUTCOME_UNKNOWN" || f.puts != 1 {
		t.Fatal(err, f.puts)
	}
}
func TestUpdateAmbiguousSecondGetFails(t *testing.T) {
	f := &fakeStore{gets: []string{empty, "{bad"}, putEr: unk}
	_, err := updateTransaction(f, "p", map[string]any{"memo": "m"})
	if classifyError(err).Code != "MUTATION_OUTCOME_UNKNOWN" || f.puts != 1 {
		t.Fatal(err)
	}
}
func TestUpdateRejectionNotReconciled(t *testing.T) {
	f := &fakeStore{gets: []string{empty}, putEr: errors.New("400 bad")}
	_, err := updateTransaction(f, "p", map[string]any{"memo": "m"})
	if err == nil || strings.Contains(err.Error(), "UPDATE_NOT") || f.puts != 1 {
		t.Fatal(err)
	}
}
