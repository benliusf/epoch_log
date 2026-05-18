package log

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	var (
		testData = []*Record{
			&Record{time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix(), 123, []byte("If you can't fly then run,")},
			&Record{time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC).Unix(), 456, []byte("if you can't run then walk,")},
			&Record{time.Date(2026, time.January, 1, 2, 0, 0, 0, time.UTC).Unix(), 789, []byte("if you can't walk then crawl,")},
			&Record{time.Date(2026, time.January, 1, 3, 0, 0, 0, time.UTC).Unix(), 101, []byte("but whatever you do you have to keep moving forward.")},
		}
	)
	dir, err := os.MkdirTemp("", "test_log")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	ctx := context.TODO()

	errs := make(chan *LogError, 10)
	log, err := NewLog(Config{
		Dir:    dir,
		Errors: errs,
	})
	require.NoError(t, err)

	t.Run("append", func(t *testing.T) {
		for _, r := range testData {
			err := log.Append(ctx, r)
			require.NoError(t, err)
		}
		require.NoError(t, log.Close())
		require.Error(t, log.Append(ctx, &Record{}))
		require.Equal(t, len(testData), len(log.segments))

		close(errs)
		require.Equal(t, 0, len(errs))
	})
	t.Run("list", func(t *testing.T) {
		uids, err := log.list()
		require.NoError(t, err)
		require.Equal(t, len(testData), len(uids))

		for i := 0; i < len(testData); i++ {
			require.Equal(t, testData[i].Epoch, uids[i])
			if i > 0 {
				require.Less(t, uids[i-1], uids[i])
			}
		}
	})
	t.Run("read", func(t *testing.T) {
		log, err = NewLog(Config{Dir: log.Config.Dir})
		require.NoError(t, err)

		b, err := log.Read(time.Now().Unix(), 123)
		require.Nil(t, b)
		require.NoError(t, err)

		for _, r := range testData {
			b, err := log.Read(r.Epoch, r.Hash)
			require.NoError(t, err)
			require.NotNil(t, b)
			require.Equal(t, r.Data, b)
		}
		require.NoError(t, log.Close())
	})
	t.Run("remove", func(t *testing.T) {
		tmp, err := newSegment(time.Now().Unix(), log.Config)
		require.NoError(t, err)
		require.NoError(t, tmp.close())

		_, err = os.Create(path.Join(log.Config.Dir, "donotdelete.me"))
		require.NoError(t, err)

		files, err := os.ReadDir(log.Config.Dir)
		require.NoError(t, err)
		require.Equal(t, (len(testData)*2)+3, len(files))

		require.NoError(t, log.Remove())

		files, err = os.ReadDir(log.Config.Dir)
		require.NoError(t, err)
		require.Equal(t, 1, len(files))
	})
}
