package stats

import (
	"math/big"

	"github.com/ddirect/container/fifo"
	"golang.org/x/exp/constraints"
)

type Stats[T constraints.Signed] struct {
	// settings
	maxSamples       int
	maxInvalidStreak int
	maxSpread        float64

	// temp values
	t1 big.Int
	t2 big.Int
	t3 big.Int

	// state
	sum           big.Int
	sum2          big.Int
	samples       fifo.Fifo[T]
	mean          T
	stdDev        T
	invalidStreak int
}

func New[T constraints.Signed](maxSamples int, maxSpread float64, maxInvalidStreak int) *Stats[T] {
	return &Stats[T]{
		maxSamples:       maxSamples,
		maxInvalidStreak: maxInvalidStreak,
		maxSpread:        maxSpread,
	}
}

func (s *Stats[T]) Reset() {
	s.sum.SetUint64(0)
	s.sum2.SetUint64(0)
	s.samples.Clear()
	s.mean = 0
	s.stdDev = 0
}

func (s *Stats[T]) SampleIn(x T) bool {
	validRange := T(float64(s.stdDev) * s.maxSpread)
	inRange := validRange <= 0 || (x >= s.mean-validRange && x <= s.mean+validRange)

	var valid bool
	if s.SampleCount() < s.maxSamples {
		valid = true
	} else {
		valid = inRange
		if valid {
			s.SampleOut()
		}
	}

	if !valid {
		s.invalidStreak++
		if s.maxInvalidStreak == 0 || s.invalidStreak < s.maxInvalidStreak {
			return false
		}
		s.Reset()
	}
	s.invalidStreak = 0

	t := s.t1.SetInt64(int64(x))
	s.sum.Add(&s.sum, t)
	s.sum2.Add(&s.sum2, t.Mul(t, t))
	s.samples.Enqueue(x)
	s.mean = s.getMean()
	s.stdDev = s.getStdDev()
	return true
}

func (s *Stats[T]) SampleOut() {
	if x, ok := s.samples.Dequeue(); ok {
		t := s.t1.SetInt64(int64(x))
		s.sum.Sub(&s.sum, t)
		s.sum2.Sub(&s.sum2, t.Mul(t, t))
	}
}

func (s *Stats[T]) getMean() T {
	n := s.SampleCount()
	if n < 1 {
		return 0
	}
	return T(s.t2.Div(&s.sum, s.t1.SetUint64(uint64(n))).Int64())
}

func (s *Stats[T]) getStdDev() T {
	n := uint64(s.SampleCount())
	if n < 2 {
		return 0
	}
	// Sqrt((n*sum2 - (sum*sum)/(n*(n-1)))
	t1 := &s.t1
	t2 := &s.t2
	t3 := &s.t3

	t1.SetUint64(n)                                     // t1 = n
	t2.Sub(t2.Mul(t1, &s.sum2), t3.Mul(&s.sum, &s.sum)) // t2 = n*sum2 - (sum*sum)
	t3.Mul(t1, t3.SetUint64(n-1))                       // t3 = n*(n-1)

	return T(t2.Div(t2, t3).Sqrt(t2).Uint64())
}

func (s *Stats[T]) SampleCount() int {
	return s.samples.Len()
}

func (s *Stats[T]) Mean() T {
	return s.mean
}

func (s *Stats[T]) StdDev() T {
	return s.stdDev
}
