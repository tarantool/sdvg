package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/tarantool/sdvg/internal/generator/common"
	"github.com/tarantool/sdvg/internal/generator/models"
	"github.com/tarantool/sdvg/internal/generator/output/general/writer"
)

const (
	maxBodySize  = 1 << 20 // 1 Mb
	retryWaitMin = 1 * time.Second
	retryWaitMax = 10 * time.Minute
)

type bodyPayload struct {
	ModelName   string
	ColumnNames []string
	Rows        [][]any
}

// Verify interface compliance in compile time.
var _ writer.Writer = (*Writer)(nil)

type Writer struct {
	ctx context.Context //nolint:containedctx

	model  *models.Model
	config *models.HTTPParams

	retryableClient *retryablehttp.Client
	lastErr         error

	payloadPool *sync.Pool

	buffer       []*models.DataRow
	bodyTemplate *template.Template

	writtenRows     uint64
	writtenRowsChan chan<- uint64

	writerChan  chan []*models.DataRow
	errorsChan  chan error
	writerWg    *sync.WaitGroup
	writerMutex *sync.Mutex
	started     bool
}

func NewWriter(
	ctx context.Context,
	model *models.Model,
	config *models.HTTPParams,
	writtenRowsChan chan<- uint64,
) *Writer {
	columnNames := make([]string, len(model.Columns))
	for i, columns := range model.Columns {
		columnNames[i] = columns.Name
	}

	payloadPool := &sync.Pool{
		New: func() any {
			return &bodyPayload{
				ModelName:   model.Name,
				ColumnNames: columnNames,
				Rows:        make([][]any, 0, config.BatchSize),
			}
		},
	}

	httpWriter := &Writer{
		ctx:             ctx,
		model:           model,
		config:          config,
		writtenRowsChan: writtenRowsChan,
		payloadPool:     payloadPool,
		buffer:          make([]*models.DataRow, 0, config.BatchSize),
		writerChan:      make(chan []*models.DataRow),
		errorsChan:      make(chan error, 1),
		writerWg:        &sync.WaitGroup{},
		writerMutex:     &sync.Mutex{},
		started:         false,
	}

	httpWriter.initRetryableClient()

	return httpWriter
}

func (w *Writer) initRetryableClient() {
	retryableClient := retryablehttp.NewClient()
	retryableClient.Logger = nil
	retryableClient.RetryWaitMin = retryWaitMin
	retryableClient.RetryWaitMax = retryWaitMax
	retryableClient.RetryMax = w.calculateRetryMax(
		w.config.Timeout,
		retryableClient.RetryWaitMin,
		retryableClient.RetryWaitMax,
	)
	retryableClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			w.lastErr = err
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	w.retryableClient = retryableClient
}

func (w *Writer) calculateRetryMax(timeout, waitMin, waitMax time.Duration) int {
	if timeout <= 0 || waitMin <= 0 {
		return 0
	}

	retries := 1
	remaining := timeout
	wait := waitMin

	for {
		if wait > waitMax {
			wait = waitMax
		}

		if remaining < wait {
			break
		}

		remaining -= wait
		retries++

		wait *= 2
	}

	return retries
}

func (w *Writer) Init() error {
	if w.started {
		return errors.New("the writer has already been initialized")
	}

	tmpl := template.New("format_template").Funcs(template.FuncMap{
		"json":     toJSON,
		"len":      length,
		"rowsJson": rowsJSON,
	})

	tmpl, err := tmpl.Parse(w.config.FormatTemplate)
	if err != nil {
		return errors.New(err.Error())
	}

	w.writerWg.Add(1)
	w.bodyTemplate = tmpl
	w.started = true

	go w.writer()

	return nil
}

func (w *Writer) writer() {
	defer w.writerWg.Done()

	pool := common.NewWorkerPool(w.handleBatch, 0, w.config.WorkersCount)
	pool.Start()
	defer pool.Stop()

	pool.Add(1)

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer pool.Done()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-done:
				return
			case batch, ok := <-w.writerChan:
				if !ok {
					return
				}

				pool.Submit(batch)
			}
		}
	}()

	if err := pool.WaitOrError(); err != nil {
		w.errorsChan <- err
	}
}

func (w *Writer) handleBatch(batch []*models.DataRow) error {
	req, err := w.buildRequest(batch)
	if err != nil {
		return errors.WithMessage(err, "failed to build request")
	}

	err = w.sendRequest(req)
	if err != nil {
		return errors.WithMessage(err, "failed to send request")
	}

	if w.writtenRowsChan != nil {
		w.writtenRowsChan <- uint64(len(batch))
	}

	return nil
}

func (w *Writer) buildRequest(dataRows []*models.DataRow) (*retryablehttp.Request, error) {
	// Grab a payload with a ready slice and reset length to zero, keep capacity.
	//
	//nolint:forcetypeassert
	payload := w.payloadPool.Get().(*bodyPayload)
	payload.Rows = payload.Rows[:0]

	for _, dataRow := range dataRows {
		payload.Rows = append(payload.Rows, dataRow.Values)
	}

	// Prepare the data payload for the request template rendering.
	// The payload includes the model name and structured row data.

	buffer := new(bytes.Buffer)

	err := w.bodyTemplate.Execute(buffer, payload)
	if err != nil {
		w.payloadPool.Put(payload)

		return nil, errors.New(err.Error())
	}

	w.payloadPool.Put(payload)

	// Construct the HTTP POST request with the generated JSON body and apply configured headers.

	req, err := retryablehttp.NewRequest(
		http.MethodPost,
		w.config.Endpoint,
		buffer,
	)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	return req, nil
}

func (w *Writer) sendRequest(req *retryablehttp.Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), w.config.Timeout)
	defer cancel()

	resp, err := w.retryableClient.Do(req.WithContext(ctx))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && w.lastErr != nil {
			return errors.Errorf("%s, last error: %s", err.Error(), w.lastErr.Error())
		}

		return errors.New(err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return errors.New(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received non-OK status code %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func (w *Writer) WriteRow(row *models.DataRow) error {
	w.buffer = append(w.buffer, row)
	w.writtenRows++

	if w.writtenRows%uint64(w.config.BatchSize) == 0 || w.writtenRows >= w.model.RowsCount {
		select {
		case <-w.ctx.Done():
			return errors.Errorf("failed to write batch: %s", w.ctx.Err().Error())
		case err := <-w.errorsChan:
			return errors.WithMessage(err, "failed to write batch")
		case w.writerChan <- w.buffer:
			w.buffer = make([]*models.DataRow, 0, w.config.BatchSize)
		}
	}

	return nil
}

func (w *Writer) Teardown() error {
	close(w.writerChan)

	w.writerWg.Wait()
	w.retryableClient.HTTPClient.CloseIdleConnections()

	select {
	case err := <-w.errorsChan:
		return errors.WithMessage(err, "failed write batch")
	default:
		return nil
	}
}
