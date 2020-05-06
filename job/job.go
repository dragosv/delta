package job

type Job interface {
	Push(config string, location string)
	Pull(config string, location string)
}
