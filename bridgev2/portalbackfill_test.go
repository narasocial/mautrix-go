// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package bridgev2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"maunium.net/go/mautrix"
)

func httpErr(status int, respErr mautrix.RespError) error {
	respErr.StatusCode = status
	return mautrix.HTTPError{
		Response:  &http.Response{StatusCode: status},
		RespError: &respErr,
	}
}

// sendErr wraps an error the way sendConvertedMessage does for a part it couldn't send.
func sendErr(err error) error {
	return fmt.Errorf("failed to send message part to Matrix: %w", err)
}

func TestIsRefusedByHomeserver(t *testing.T) {
	tooLarge := httpErr(http.StatusRequestEntityTooLarge, mautrix.MTooLarge)
	forbidden := httpErr(http.StatusForbidden, mautrix.MForbidden)
	badJSON := httpErr(http.StatusBadRequest, mautrix.MBadJSON)
	dbErr := fmt.Errorf("%w: failed to save message part to database: %w", ErrDatabaseError, errors.New("connection reset"))

	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"too large", errors.Join(sendErr(tooLarge)), true},
		{"bad json", errors.Join(sendErr(badJSON)), true},
		{"forbidden", errors.Join(sendErr(forbidden)), true},
		{"every part refused", errors.Join(sendErr(tooLarge), sendErr(badJSON)), true},
		{"unwrapped", tooLarge, true},
		{"nil", nil, false},
		{"unknown token", errors.Join(sendErr(httpErr(http.StatusUnauthorized, mautrix.MUnknownToken))), false},
		{"rate limited", errors.Join(sendErr(httpErr(http.StatusTooManyRequests, mautrix.MLimitExceeded))), false},
		{"server error", errors.Join(sendErr(httpErr(http.StatusInternalServerError, mautrix.MUnknown))), false},
		{"no response", errors.Join(sendErr(mautrix.HTTPError{Message: "dial tcp: connection refused"})), false},
		{"database error", errors.Join(dbErr), false},
		{"refused part and database error", errors.Join(sendErr(tooLarge), dbErr), false},
		{"context cancelled", errors.Join(context.Canceled), false},
		{"empty join", errors.Join(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRefusedByHomeserver(tc.err))
		})
	}
}
