package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"sdvg/internal/generator/common"
	"sdvg/internal/generator/models"
	"sdvg/internal/generator/output/general/writer"
)

const (
	retryWaitMin = 1 * time.Second
	retryWaitMax = 10 * time.Minute
)

type bodyPayload struct {
	ModelName string
	Rows      []map[string]any
}

// Verify interface compliance in compile time.
var _ writer.Writer = (*Writer)(nil)

type Writer struct {
	ctx context.Context //nolint:containedctx

	model  *models.Model
	config *models.HTTPParams

	retryableClient *retryablehttp.Client
	lastErr         error

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
	httpWriter := &Writer{
		ctx:             ctx,
		model:           model,
		config:          config,
		writtenRowsChan: writtenRowsChan,
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

	tmpl := template.New("body").Funcs(template.FuncMap{
		"json": func(v any) (string, error) {
			data, err := json.Marshal(v)

			return string(data), err
		},
		"len": func(v any) int {
			return reflect.ValueOf(v).Len()
		},
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
	// Build a slice of row objects by mapping column names to their corresponding values.
	// Each row is represented as a map[string]any, with column names as keys and values from dataRows.
	rows := make([]map[string]any, 0, len(dataRows))

	for _, dataRow := range dataRows {
		if len(dataRow.Values) != len(w.model.Columns) {
			return nil, errors.New("values count does not match columns count")
		}

		rowObj := make(map[string]any, len(dataRow.Values))
		for i, value := range dataRow.Values {
			rowObj[w.model.Columns[i].Name] = value
		}

		rows = append(rows, rowObj)
	}

	// Prepare the data payload for the request template rendering.
	// The payload includes the model name and structured row data.

	body := bodyPayload{
		ModelName: w.model.Name,
		Rows:      rows,
	}

	var buf bytes.Buffer

	err := w.bodyTemplate.Execute(&buf, body)
	if err != nil {
		return nil, errors.New(err.Error())
	}

	// Construct the HTTP POST request with the generated JSON body and apply configured headers.

	req, err := retryablehttp.NewRequest(
		http.MethodPost,
		w.config.Endpoint,
		&buf,
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

	if resp == nil {
		return errors.New("received nil response")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
