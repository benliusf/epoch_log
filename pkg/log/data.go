package log

type Record struct {
	Epoch int64

	Hash int64
	Data []byte
}
