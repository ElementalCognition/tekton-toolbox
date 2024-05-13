package gcslogproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/ElementalCognition/tekton-toolbox/pkg/logproxy"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"gopkg.in/go-playground/pool.v3"
	"knative.dev/pkg/logging"
)

const (
	listTimeout  = time.Second * 10
	fetchTimeout = time.Minute * 1
)

func wrapError(err error) error {
	if err == storage.ErrBucketNotExist {
		return &logproxy.BucketNotExistError{Err: err}
	} else if err == storage.ErrObjectNotExist {
		return &logproxy.ObjectNotExistError{Err: err}
	}
	return err
}

type fetchResult struct {
	idx int
	buf []byte
}

type fetcher struct {
	pool          pool.Pool
	bucket        string
	storageClient *storage.Client
}

var _ logproxy.Service = (*fetcher)(nil)

func (f *fetcher) list(ctx context.Context, dirname string) ([]string, error) {
	var files []string
	ctx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()
	it := f.storageClient.Bucket(f.bucket).Objects(ctx, &storage.Query{
		Prefix:    dirname,
		Delimiter: "",
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, wrapError(err)
		}
		files = append(files, attrs.Name)
	}
	sort.Strings(files)
	return files, nil
}

func (f *fetcher) fetch(ctx context.Context, filename string) ([]byte, error) {
	logger := logging.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	rc, err := f.storageClient.Bucket(f.bucket).Object(filename).NewReader(ctx)
	if err != nil {
		return nil, wrapError(err)
	}
	defer func(_ *fetcher, rc *storage.Reader) {
		err := rc.Close()
		if err != nil {
			logger.Errorw("Service failed to close GCS reader", zap.Error(err))
		}
	}(f, rc)
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *fetcher) fetchWorker(ctx context.Context, idx int, filename string) func(wu pool.WorkUnit) (interface{}, error) {
	return func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		buf, err := f.fetch(ctx, filename)
		return &fetchResult{
			idx: idx,
			buf: buf,
		}, err
	}
}

func (f *fetcher) fetchAll(ctx context.Context, filenames ...string) ([]byte, error) {
	batch := f.pool.Batch()
	for idx, file := range filenames {
		batch.Queue(f.fetchWorker(ctx, idx, file))
	}
	batch.QueueComplete()
	buf := make([][]byte, len(filenames))
	me := new(multierror.Error)
	for r := range batch.Results() {
		wr := r.Value().(*fetchResult)
		buf[wr.idx] = wr.buf
		if r.Error() != nil {
			me = multierror.Append(me, r.Error())
		}
	}
	err := me.ErrorOrNil()
	if err != nil {
		return nil, err
	}
	return bytes.Join(buf, []byte{}), nil
}

func (f *fetcher) Fetch(ctx context.Context, namespace, pod, container string) ([]byte, error) {
	logger := logging.FromContext(ctx)
	logger.Infow("Service started fetching logs",
		zap.String("namespace", namespace),
		zap.String("pod", pod),
		zap.String("container", container),
		zap.String("bucket", f.bucket),
	)
	dirname := fmt.Sprintf("%s/%s/%s", namespace, pod, container)
	files, err := f.list(ctx, dirname)
	if err != nil {
		return nil, err
	}
	return f.fetchAll(ctx, files...)
}

func NewService(
	bucket string,
	storageClient *storage.Client,
	pool pool.Pool,
) logproxy.Service {
	return &fetcher{
		pool:          pool,
		bucket:        bucket,
		storageClient: storageClient,
	}
}
