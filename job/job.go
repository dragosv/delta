package job

type Job interface {
	Push(config string, location string) error
	Pull(config string, location string) error
}
