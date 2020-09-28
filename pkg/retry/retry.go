package retry

import (
	"log"
	"time"
)

var (
	sleepBase   = time.Second * 5
	sleepJitter = time.Second
)

type Step struct {
	Name string
	Do   func() error
}

// Retry executes 'fn', if error happened, it will be logged,
// and next attempt will be committed, after short sleep, 'tries' times.
func Run(tries int, reason string, fn func() error) (err error) {
	for t := 0; t < tries; t++ {
		err = fn()

		switch {
		case err == nil:
			return
		case t < tries:
			log.Printf("[retry] %s: try %d last err: %v", reason, t+1, err)
			time.Sleep(sleepBase + sleepJitter*time.Duration(t+1))
		}
	}

	return
}

// RunSteps executes several `steps` one by one, returns first error.
func RunSteps(tries int, steps []Step) (err error) {
	for i := 0; i < len(steps); i++ {
		step := &steps[i]

		if err = Run(tries, step.Name, step.Do); err != nil {
			return
		}
	}

	return
}
