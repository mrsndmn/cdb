package cdb

import (
	"hash/fnv"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testCDBRecord struct {
	key []byte
	val []byte
}
type CDBTestSuite struct {
	suite.Suite
	cdbFile     *os.File
	cdbHandle   *CDB
	testRecords []testCDBRecord
}

func TestCDBTestSuite(t *testing.T) {
	suite.Run(t, new(CDBTestSuite))
}

func (suite *CDBTestSuite) getCDBReader() Reader {
	// initialize reader
	reader, err := suite.cdbHandle.GetReader(suite.cdbFile)
	suite.Require().Nilf(err, "Can't get CDB reader: %#v", err)
	return reader
}

func (suite *CDBTestSuite) getCDBWriter() Writer {
	// initialize writer
	writer, err := suite.cdbHandle.GetWriter(suite.cdbFile)
	suite.Require().Nilf(err, "Can't get CDB writer: %#v", err)
	return writer
}

func (suite *CDBTestSuite) SetupTest() {
	f, err := ioutil.TempFile("", "test_*.cdb")
	suite.Require().Nilf(err, "Can't open temporary file: %#v", err)

	suite.cdbFile = f
	suite.cdbHandle = New() // create new cdb handle

	// generate test records
	testRecords := make([]testCDBRecord, 10)
	for i := 0; i < len(testRecords); i++ {
		stri := strconv.Itoa(i)
		testRecords[i].key = []byte("key" + stri)
		testRecords[i].val = []byte("val" + stri)
	}
	suite.testRecords = testRecords
}

func (suite *CDBTestSuite) TearDownTest() {
	err := suite.cdbFile.Close()
	suite.Nilf(err, "Can't close cdb file: %#v", err)
	err = os.Remove(suite.cdbFile.Name())
	suite.Nilf(err, "Can't remove cdb file: %#v", err)
}

func (suite *CDBTestSuite) fillTestCDB() {

	writer := suite.getCDBWriter()
	defer func() {
		err := writer.Close()
		suite.Require().Nilf(err, "Can't close cdb writer: %#v", err)
	}()

	for _, rec := range suite.testRecords {
		err := writer.Put(rec.key, rec.val)
		suite.Require().Nilf(err, "Cant put new value to cdb: %#v", err)
	}

}

func (suite *CDBTestSuite) TestWriteCDB() {
	suite.fillTestCDB()
}

func (suite *CDBTestSuite) TestShouldReturnAllValues() {
	suite.fillTestCDB()

	reader := suite.getCDBReader()

	for _, rec := range suite.testRecords {
		value, err := reader.Get(rec.key)
		suite.Nilf(err, "Can't get from cdb key: %s", string(rec.key))

		suite.Equalf(value, rec.val, "Test record value differs from the cdb's one")
		exists, err := reader.Has(rec.key)
		suite.Nilf(err, "error on check has key")
		suite.True(exists)
	}

	suite.Equal(reader.Size(), len(suite.testRecords), "check records count")
}

func (suite *CDBTestSuite) writeEmptyCDB() {
	writer := suite.getCDBWriter()
	err := writer.Close()
	suite.Require().Nilf(err, "Can't close writer %#v", err)
}

func (suite *CDBTestSuite) TestShouldReturnNilOnNonExistingKeys() {
	suite.writeEmptyCDB()

	reader := suite.getCDBReader()

	for _, rec := range suite.testRecords {
		value, err := reader.Get(rec.key)
		suite.EqualError(err, ErrEntryNotFound.Error(), "Can't get from cdb key: %s", string(rec.key))
		suite.Nil(value, "Value must be nil for non existing keys")

		exists, err := reader.Has(rec.key)
		suite.Nilf(err, "error on check has key")
		suite.False(exists)
	}
}

func (suite *CDBTestSuite) TestConcurrentGet() {
	suite.fillTestCDB()

	reader := suite.getCDBReader()

	wg := &sync.WaitGroup{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for _, rec := range suite.testRecords {
				value, err := reader.Get(rec.key)
				suite.Nilf(err, "Error while getting key")
				suite.Equal(value, rec.val)
			}
		}()
	}

	wg.Wait()
}

func (suite *CDBTestSuite) TestSetHash() {
	suite.cdbHandle.SetHash(fnv.New32)
	suite.TestShouldReturnAllValues()
}

func BenchmarkGetReader(b *testing.B) {

	n := 1000
	f, _ := os.Create("test.cdb")
	defer f.Close()
	defer os.Remove("test.cdb")

	handle := New()
	writer, _ := handle.GetWriter(f)

	keys := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		writer.Put(keys[i], keys[i])
	}

	writer.Close()

	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		handle.GetReader(f)
	}

}

func BenchmarkReaderGet(b *testing.B) {

	n := 1000
	f, _ := os.Create("test.cdb")
	defer f.Close()
	defer os.Remove("test.cdb")

	handle := New()
	writer, _ := handle.GetWriter(f)

	keys := make([][]byte, n)
	for i := 0; i < n; i++ {
		keys[i] = []byte(strconv.Itoa(i))
		writer.Put(keys[i], keys[i])
	}

	writer.Close()
	reader, _ := handle.GetReader(f)

	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		reader.Get(keys[j%n])
	}

}

func BenchmarkWriterPut(b *testing.B) {
	f, _ := os.Create("test.cdb")
	defer f.Close()
	defer os.Remove("test.cdb")

	handle := New()
	writer, _ := handle.GetWriter(f)

	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		key := []byte(strconv.Itoa(j))
		writer.Put(key, key)
	}

	writer.Close()
}
