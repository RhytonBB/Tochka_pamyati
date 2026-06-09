package mailer

import (
	"context"
	"log"
	"sync"
)

type Queue struct {
	wg     sync.WaitGroup
	jobs   chan job
	closed chan struct{}
}

type job struct {
	sender Sender
	msg    Message
}

func NewQueue(workerCount int) *Queue {
	if workerCount <= 0 {
		workerCount = 1
	}
	q := &Queue{
		jobs:   make(chan job, 100),
		closed: make(chan struct{}),
	}
	for i := 0; i < workerCount; i++ {
		q.wg.Add(1)
		go func(workerID int) {
			defer q.wg.Done()
			for j := range q.jobs {
				if err := j.sender.Send(context.Background(), j.msg); err != nil {
					log.Printf("[MAILER] worker %d error sending to %s: %v", workerID, j.msg.To, err)
				} else {
					log.Printf("[MAILER] worker %d sent email to %s: %s", workerID, j.msg.To, j.msg.Subject)
				}
			}
		}(i)
	}
	return q
}

func (q *Queue) Enqueue(sender Sender, msg Message) {
	select {
	case <-q.closed:
		log.Printf("[MAILER] attempt to enqueue to closed queue")
		return
	default:
	}
	select {
	case q.jobs <- job{sender: sender, msg: msg}:
	default:
		log.Printf("[MAILER] queue full, dropping message to %s", msg.To)
	}
}

func (q *Queue) Close() {
	select {
	case <-q.closed:
		return
	default:
		close(q.closed)
		close(q.jobs)
		q.wg.Wait()
	}
}
