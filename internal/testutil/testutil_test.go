package testutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/georgysavva/dbscan/internal/testutil"
)

func TestStartCrdbServer(t *testing.T) {
	ts, err := testutil.StartCrdbServer()
	require.NoError(t, err)
	defer ts.Stop()
}
