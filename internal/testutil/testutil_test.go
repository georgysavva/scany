// +build integration

package testutil_test

import (
	"github.com/georgysavva/dbscan/internal/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStartCrdbServer(t *testing.T) {
	ts, err := testutil.StartCrdbServer()
	require.NoError(t, err)
	defer ts.Stop()
}
