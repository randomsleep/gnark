package lzss

import (
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/compress"
	test_vector_utils "github.com/consensys/gnark/std/utils/test_vectors_utils"
	"github.com/consensys/gnark/test"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCompression1ZeroE2E(t *testing.T) {
	testCompressionE2E(t, []byte{0}, nil, "1_zero")
}

func BenchmarkCompression26KBE2E(b *testing.B) {
	_, err := BenchCompressionE2ECompilation(nil, "../test_cases/3c2943")
	assert.NoError(b, err)
}

func testCompressionE2E(t *testing.T, d, dict []byte, name string) {
	if d == nil {
		var err error
		d, err = os.ReadFile("../test_cases/" + name + "/data.bin")
		assert.NoError(t, err)
	}

	// compress

	compressor, err := NewCompressor(dict, BestCompression)
	assert.NoError(t, err)

	c, err := compressor.Compress(d)
	assert.NoError(t, err)

	cStream := ReadIntoStream(c, dict, BestCompression)

	cSum, err := check(cStream, cStream.Len())
	assert.NoError(t, err)

	dSum, err := check(compress.NewStreamFromBytes(d), len(d))
	assert.NoError(t, err)

	circuit := compressionCircuit{
		C:               make([]frontend.Variable, cStream.Len()),
		D:               make([]frontend.Variable, len(d)),
		Dict:            make([]byte, len(dict)),
		CompressionMode: BestCompression,
	}

	// solve the circuit or only compile it

	assignment := compressionCircuit{
		CChecksum: cSum,
		DChecksum: dSum,
		C:         test_vector_utils.ToVariableSlice(cStream.D),
		D:         test_vector_utils.ToVariableSlice(d),
		Dict:      dict,
		CLen:      cStream.Len(),
		DLen:      len(d),
	}
	test.NewAssert(t).SolvingSucceeded(&circuit, &assignment, test.WithBackends(backend.PLONK), test.WithCurves(ecc.BN254))
}

func TestChecksum0(t *testing.T) {
	testChecksum(t, compress.Stream{D: []int{}, NbSymbs: 256})
}

func testChecksum(t *testing.T, d compress.Stream) {
	circuit := checksumTestCircuit{
		Inputs:   make([]frontend.Variable, d.Len()),
		InputLen: d.Len(),
	}

	sum, err := check(d, d.Len())
	assert.NoError(t, err)

	assignment := checksumTestCircuit{
		Inputs:   test_vector_utils.ToVariableSlice(d.D),
		InputLen: d.Len(),
		Sum:      sum,
	}
	test.NewAssert(t).SolvingSucceeded(&circuit, &assignment, test.WithBackends(backend.PLONK), test.WithCurves(ecc.BN254))
}

type checksumTestCircuit struct {
	Inputs   []frontend.Variable
	InputLen frontend.Variable
	Sum      frontend.Variable
}

func (c *checksumTestCircuit) Define(api frontend.API) error {
	if err := checkSnark(api, c.Inputs, len(c.Inputs), c.Sum); err != nil {
		return err
	}
	return nil
}
