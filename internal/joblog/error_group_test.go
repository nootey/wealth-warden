package joblog

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel)
	return zap.New(core), logs
}

func TestErrorGroupCollapsesRepeatedErrors(t *testing.T) {
	g := NewErrorGroup("userID")
	boom := errors.New("connection refused")

	for i := int64(1); i <= 1000; i++ {
		g.Add(i, boom)
	}
	g.Add(1001, errors.New("asset not found"))
	g.Add(1002, nil)

	assert.Equal(t, 1001, g.Count())

	logger, logs := newTestLogger()
	g.Log(logger, "Failed")

	entries := logs.All()
	require.Len(t, entries, 2)

	assert.Equal(t, int64(1000), entries[0].ContextMap()["affected"])
	assert.Equal(t, "connection refused", entries[0].ContextMap()["error"])
	assert.Len(t, entries[0].ContextMap()["userID_sample"], maxSampleIDs)

	assert.Equal(t, int64(1), entries[1].ContextMap()["affected"])
	assert.Equal(t, "asset not found", entries[1].ContextMap()["error"])
}

func TestErrorGroupCapsDistinctErrors(t *testing.T) {
	g := NewErrorGroup("userID")
	for i := int64(0); i < 50; i++ {
		g.Add(i, fmt.Errorf("unique failure %d", i))
	}

	logger, logs := newTestLogger()
	g.Log(logger, "Failed")

	entries := logs.All()
	require.Len(t, entries, maxDistinctErrors+1)

	last := entries[len(entries)-1].ContextMap()
	assert.Equal(t, int64(50-maxDistinctErrors), last["affected"])
	assert.Equal(t, int64(50-maxDistinctErrors), last["distinct_errors_not_shown"])
}

func TestErrorGroupSilentWhenEmpty(t *testing.T) {
	g := NewErrorGroup("userID")
	g.Add(1, nil)

	logger, logs := newTestLogger()
	g.Log(logger, "Failed")

	assert.Equal(t, 0, g.Count())
	assert.Empty(t, logs.All())
}
