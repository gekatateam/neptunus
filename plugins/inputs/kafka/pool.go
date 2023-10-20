package kafka

type readersPool struct {
	new func() *topicReader
}

type topicReader struct {}
