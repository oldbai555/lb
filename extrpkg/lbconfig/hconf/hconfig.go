package hconf

type Data struct {
	Key string
	Val interface{}
}

type DataSource interface {
	Load() ([]*Data, error)
	Watch() (DataWatcher, error)
}

type DataWatcher interface {
	Change() ([]*Data, error)
	Close() error
}
