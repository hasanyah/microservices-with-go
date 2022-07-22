package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
)

type LoggingMiddleware struct {
	Logger log.Logger
	Next   QueryService
}

func (mw LoggingMiddleware) Search(c context.Context, s string) (output []mediaObject, err error) {
	defer func(begin time.Time) {

		jsonData, err := json.Marshal(&output)
		var printableOutput string
		if err != nil {
			printableOutput = fmt.Sprintf("Entity could not be formatted")
		}
		printableOutput = string(jsonData)
		_ = mw.Logger.Log(
			"method", "userQueryPropagation",
			"input", s,
			"output", printableOutput,
			"err", err,
			"duration", time.Since(begin),
		)
	}(time.Now())

	output, err = mw.Next.Search(c, s)
	return
}

func (mw LoggingMiddleware) ServiceStatus(_ context.Context) (output int, err error) {
	defer func(begin time.Time) {
		_ = mw.Logger.Log(
			"method", "userQueryServiceStatus",
			"output", output,
			"err", err,
			"duration", time.Since(begin),
		)
	}(time.Now())

	return
}

type InstrumentingMiddleware struct {
	RequestCount   metrics.Counter
	RequestLatency metrics.Histogram
	Next           QueryService
}

func (mw InstrumentingMiddleware) Search(c context.Context, s string) (output []mediaObject, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "search", "error", fmt.Sprint(err != nil)}
		mw.RequestCount.With(lvs...).Add(1)
		mw.RequestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	output, err = mw.Next.Search(c, s)
	return
}

func (mw InstrumentingMiddleware) ServiceStatus(_ context.Context) (output int, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "searchservicestatus", "error", fmt.Sprint(err != nil)}
		mw.RequestCount.With(lvs...).Add(1)
		mw.RequestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return
}
