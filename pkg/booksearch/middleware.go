package book

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
	Next   BookService
}

func (mw LoggingMiddleware) Find(c context.Context, s string) (output []Book, err error) {
	defer func(begin time.Time) {

		jsonData, err := json.Marshal(&output)
		var printableOutput string
		if err != nil {
			printableOutput = fmt.Sprintf("Entity could not be formatted")
		}
		printableOutput = string(jsonData)

		_ = mw.Logger.Log(
			"method", "findBookRequest",
			"input", s,
			"output", printableOutput,
			"err", err,
			"duration", time.Since(begin),
		)
	}(time.Now())

	output, err = mw.Next.Find(c, s)
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
	Next           BookService
}

func (mw InstrumentingMiddleware) Find(c context.Context, s string) (output []Book, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "search", "error", fmt.Sprint(err != nil)}
		mw.RequestCount.With(lvs...).Add(1)
		mw.RequestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())

	output, err = mw.Next.Find(c, s)
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
