package log

type worker struct {
	log *Log

	done chan struct{}
}

func newWorker(l *Log) *worker {
	return &worker{
		log:  l,
		done: make(chan struct{}),
	}
}

func (w *worker) run() {
	go func() {
		for record := range w.log.buf {
			if err := w.write(record); err != nil {
				w.error(err, record)
			}
		}
		w.done <- struct{}{}
	}()
}

func (w *worker) write(record *Record) error {
	s, ok := w.log.segments.get(record.Epoch)
	if !ok {
		tmp, err := w.log.newSegment(record.Epoch)
		if err != nil {
			return err
		}
		s = tmp
	}
	if s != nil {
		return s.append(record)
	}
	return nil
}

func (w *worker) flush() {
	<-w.done
	for _, segment := range w.log.segments.iter() {
		if err := segment.flush(); err != nil {
			w.error(err, nil)
		}
	}
}

func (w *worker) error(err error, record *Record) {
	if w.log.errs != nil {
		w.log.errs <- &LogError{
			error:  err,
			Record: record,
		}
	}
}
