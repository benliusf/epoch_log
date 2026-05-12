package log

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	logTestData = []*Record{
		&Record{time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), 123, []byte("If you can't fly then run,")},
		&Record{time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC).Unix(), 456, []byte("if you can't run then walk,")},
		&Record{time.Date(2026, time.January, 1, 2, 0, 0, 0, time.UTC).Unix(), 789, []byte("if you can't walk then crawl,")},
		&Record{time.Date(2026, time.January, 1, 3, 0, 0, 0, time.UTC).Unix(), 101, []byte("but whatever you do you have to keep moving forward.")},
	}
)

func TestLog(t *testing.T) {
	dir, err := os.MkdirTemp("", "test_log")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	errs := make(chan *LogError, 10)
	log, err := NewLog(Config{
		Dir:    dir,
		Errors: errs,
	})
	require.NoError(t, err)

	testAppend(t, log)
	close(errs)
	require.Equal(t, 0, len(errs))

	testList(t, log)
	testLogRead(t, log)
	testRemove(t, log)
}

func testAppend(t *testing.T, log *Log) {
	t.Helper()

	ctx := context.TODO()
	for _, r := range logTestData {
		err := log.Append(ctx, r)
		require.NoError(t, err)
	}
	require.NoError(t, log.Close())
	require.Equal(t, len(logTestData), len(log.segments))
}

func testList(t *testing.T, log *Log) {
	t.Helper()

	uids, err := log.list()
	require.NoError(t, err)
	require.Equal(t, len(logTestData), len(uids))

	for i := 0; i < len(logTestData); i++ {
		require.Equal(t, logTestData[i].Epoch, uids[i])
	}
}

func testLogRead(t *testing.T, log *Log) {
	t.Helper()

	log, err := NewLog(Config{
		Dir: log.Config.Dir,
	})
	require.NoError(t, err)

	for _, r := range logTestData {
		b, err := log.Read(r.Epoch, r.Hash)
		require.NoError(t, err)
		require.NotNil(t, b)
		require.Equal(t, r.Data, b)
	}
	b, err := log.Read(time.Now().Unix(), 123)
	require.Nil(t, b)
	require.NoError(t, err)

	require.NoError(t, log.Close())
}

func testRemove(t *testing.T, log *Log) {
	t.Helper()

	_, err := os.Create(path.Join(log.Config.Dir, "donotdelete.me"))
	require.NoError(t, err)

	files, err := os.ReadDir(log.Config.Dir)
	require.NoError(t, err)
	require.Equal(t, (len(logTestData)*2)+1, len(files))

	require.NoError(t, log.Remove())

	files, err = os.ReadDir(log.Config.Dir)
	require.NoError(t, err)
	require.Equal(t, 1, len(files))
}
