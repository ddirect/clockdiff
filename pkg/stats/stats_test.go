package stats_test

import (
	"clockdiff/pkg/stats"
	"encoding/binary"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const enableLog = true

func core(t *testing.T, offset int64, samples []byte) {
	const maxSamples = 1e6
	n := int64(len(samples))

	if n < 2 || n > maxSamples {
		return
	}

	s := stats.New[int64](maxSamples, 0) // spread is not used here since we always have a number of samples <= maxSamples

	tr := new(big.Rat)
	ti := new(big.Int)

	sum := new(big.Int)
	for _, b := range samples {
		sample := int64(b) + offset
		sum.Add(sum, ti.SetInt64(sample))
		assert.True(t, s.SampleIn(sample))
	}

	avg := new(big.Rat)
	avg.SetFrac(sum, ti.SetInt64(n))

	sumSqDev := new(big.Rat)
	for _, b := range samples {
		sample := int64(b) + offset
		sumSqDev.Add(sumSqDev, tr.SetInt64(sample).Sub(tr, avg).Mul(tr, tr))
	}
	tr.Quo(sumSqDev, tr.SetInt64(n-1))
	stdDevI := ti.Div(tr.Num(), tr.Denom()).Sqrt(ti).Int64()
	avgI := ti.Div(avg.Num(), avg.Denom()).Int64()

	if enableLog {
		logFile.Write(fmt.Appendf(nil, "%5d %v avg: %v stddev: %v\n", offset, samples, avgI, stdDevI))
	}

	assert.Equal(t, avgI, s.Mean())
	assert.Equal(t, stdDevI, s.StdDev())
}

var logFile = func() *os.File {
	if !enableLog {
		return nil
	}
	res, err := os.OpenFile("fuzz.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return res
}()

func Fuzz_Core(f *testing.F) {
	for _, i := range []int64{-255, -127, 0, 127} {
		buf := make([]byte, 4)
		binary.NativeEndian.PutUint32(buf, rand.Uint32())
		f.Add(i, buf)
	}
	f.Fuzz(core)
}
