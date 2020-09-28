//nolint:testpackage
package retry

import (
	"errors"
	"testing"
	"time"
)

const maxTries = 3

var errFail = errors.New("test fail")

type failer struct {
	fn  func()
	lim int
}

func newFailer(f func()) *failer {
	return &failer{fn: f}
}

func (f *failer) Fail() (err error) {
	f.fn()

	if f.lim > 0 {
		f.lim--

		err = errFail
	}

	return
}

func (f *failer) Reset(limit int) {
	f.lim = limit
}

func TestRun(t *testing.T) {
	sleepBase = time.Millisecond * 100
	sleepJitter = time.Millisecond * 20

	var table = []struct {
		errCount    int
		countExpext int
		errExpect   error
	}{
		{1, 2, nil},
		{2, 3, nil},
		{maxTries, 3, errFail},
	}

	var (
		count int
		err   error
	)

	fail := newFailer(func() { count++ })

	for n, s := range table {
		fail.Reset(s.errCount)

		err = Run(maxTries, "", fail.Fail)
		if errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if count != s.countExpext {
			t.Fatalf("step %d: count = %d (want: %d)", n, count, s.countExpext)
		}

		count = 0
	}
}

func TestRunSteps(t *testing.T) {
	sleepBase = time.Millisecond * 100
	sleepJitter = time.Millisecond * 20

	var table = []struct {
		errCountA    int
		countAExpext int
		errCountB    int
		countBExpext int
		errExpect    error
	}{
		{1, 2, 0, 1, nil},
		{maxTries, 3, 0, 0, errFail},
		{1, 2, maxTries, 3, errFail},
	}

	var (
		countA int
		countB int
		err    error
	)

	failA := newFailer(func() { countA++ })
	failB := newFailer(func() { countB++ })

	steps := []Step{
		{Name: "A", Do: failA.Fail},
		{Name: "B", Do: failB.Fail},
	}

	for n, s := range table {
		failA.Reset(s.errCountA)
		failB.Reset(s.errCountB)

		err = RunSteps(maxTries, steps)
		if errors.Is(err, s.errExpect) {
			t.Fatalf("step %d: err == %v", n, err)
		}

		if countA != s.countAExpext {
			t.Fatalf("step %d: countA = %d (want: %d)", n, countA, s.countAExpext)
		}

		if countB != s.countBExpext {
			t.Fatalf("step %d: countB = %d (want: %d)", n, countB, s.countBExpext)
		}

		countA, countB = 0, 0
	}
}
