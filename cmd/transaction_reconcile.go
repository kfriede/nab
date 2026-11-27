package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/kfriede/nab/internal/api"
)

// txnStore is the minimal client surface used to update one transaction.
type txnStore interface {
	GetJSON(path string, target any) error
	Put(path string, body any) ([]byte, error)
}

// UpdateNotCommittedError means a PUT outcome was ambiguous, but a follow-up GET
// proved the transaction is unchanged. The PUT is never replayed; a later fresh
// dry-run and confirmation may retry.
type UpdateNotCommittedError struct{ Cause error }

func (e *UpdateNotCommittedError) Error() string {
	return "UPDATE_NOT_COMMITTED: the update response was lost and a follow-up GET shows the transaction unchanged; run a fresh dry-run before any retry"
}
func (e *UpdateNotCommittedError) Unwrap() error { return e.Cause }

func getTxn(c txnStore, path string) (map[string]any, error) {
	var r struct {
		Transaction map[string]any `json:"transaction"`
	}
	if err := c.GetJSON(path, &r); err != nil {
		return nil, err
	}
	if r.Transaction == nil {
		return nil, errors.New("transaction missing from GET response")
	}
	return r.Transaction, nil
}

// updateTransaction does GET, one PUT, and on an ambiguous PUT outcome exactly one
// reconciling GET. It never replays the PUT. The returned map is the transaction
// result (the PUT response, or the reconciled GET when the fields all match).
func updateTransaction(c txnStore, path string, fields map[string]any) (map[string]any, error) {
	before, err := getTxn(c, path)
	if err != nil {
		return nil, fmt.Errorf("reading transaction before update: %w", err)
	}
	data, err := c.Put(path, map[string]any{"transaction": fields})
	if err == nil {
		var result map[string]any
		if perr := json.Unmarshal(data, &result); perr != nil {
			return nil, perr
		}
		return result, nil
	}
	var unknown *api.MutationOutcomeUnknownError
	if !errors.As(err, &unknown) {
		return nil, fmt.Errorf("updating transaction: %w", err)
	}
	after, gerr := getTxn(c, path)
	if gerr != nil {
		return nil, fmt.Errorf("updating transaction: %w", unknown)
	}
	if reflect.DeepEqual(before, after) {
		return nil, &UpdateNotCommittedError{Cause: unknown.Cause}
	}
	for k, v := range fields {
		got, ok := after[k]
		if !ok || !reflect.DeepEqual(normalize(v), normalize(got)) {
			return nil, fmt.Errorf("updating transaction: %w", unknown)
		}
	}
	return map[string]any{"transaction": after}, nil
}

func normalize(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if json.Unmarshal(b, &out) != nil {
		return v
	}
	return out
}
