package joblog

import (
	"sort"

	"go.uber.org/zap"
)

const (
	maxDistinctErrors = 5
	maxSampleIDs      = 10
)

type entry struct {
	err   error
	ids   []int64
	count int
}

// Collapses one log line per failed item into one line per distinct error.
type ErrorGroup struct {
	idField string
	order   []string
	byMsg   map[string]*entry
	total   int
}

func NewErrorGroup(idField string) *ErrorGroup {
	return &ErrorGroup{idField: idField, byMsg: make(map[string]*entry)}
}

func (g *ErrorGroup) Add(id int64, err error) {
	if err == nil {
		return
	}
	g.total++

	msg := err.Error()
	e, ok := g.byMsg[msg]
	if !ok {
		e = &entry{err: err}
		g.byMsg[msg] = e
		g.order = append(g.order, msg)
	}
	e.count++
	if len(e.ids) < maxSampleIDs {
		e.ids = append(e.ids, id)
	}
}

func (g *ErrorGroup) Count() int { return g.total }

func (g *ErrorGroup) Log(logger *zap.Logger, msg string) {
	if g.total == 0 {
		return
	}

	entries := make([]*entry, 0, len(g.order))
	for _, m := range g.order {
		entries = append(entries, g.byMsg[m])
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].count > entries[j].count })

	shown := entries
	if len(shown) > maxDistinctErrors {
		shown = shown[:maxDistinctErrors]
	}

	for _, e := range shown {
		logger.Error(msg,
			zap.Int("affected", e.count),
			zap.Int64s(g.idField+"_sample", e.ids),
			zap.Error(e.err))
	}

	if len(entries) > len(shown) {
		hidden := 0
		for _, e := range entries[len(shown):] {
			hidden += e.count
		}
		logger.Error(msg,
			zap.Int("affected", hidden),
			zap.Int("distinct_errors_not_shown", len(entries)-len(shown)))
	}
}
