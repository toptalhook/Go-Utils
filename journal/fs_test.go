package journal_test

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	utils "github.com/Laisky/go-utils"
	"github.com/Laisky/go-utils/journal"
)

type FNameCase struct {
	OldFName, ExpectFName, NowTS string
}

func TestGenerateNewBufFName(t *testing.T) {
	var (
		err      error
		now      time.Time
		newFName string
		cases    = []*FNameCase{
			&FNameCase{
				OldFName:    "200601020001.buf",
				ExpectFName: "200601020002.buf",
				NowTS:       "20060102-0700",
			},
			&FNameCase{
				OldFName:    "200601020001.ids",
				ExpectFName: "200601020002.ids",
				NowTS:       "20060102-0700",
			},
			&FNameCase{
				OldFName:    "200601020002.buf",
				ExpectFName: "200601040001.buf",
				NowTS:       "20060104-0700",
			},
			&FNameCase{
				OldFName:    "200601020002.buf",
				ExpectFName: "200601030001.buf",
				NowTS:       "20060103-0600",
			},
		}
	)

	for _, testcase := range cases {
		now, err = time.Parse("20060102-0700", testcase.NowTS)
		if err != nil {
			t.Fatalf("got error: %+v", err)
		}
		newFName, err = journal.GenerateNewBufFName(now, testcase.OldFName)
		if err != nil {
			t.Fatalf("got error: %+v", err)
		}
		if newFName != testcase.ExpectFName {
			t.Errorf("expect %v, got %v", testcase.ExpectFName, newFName)
		}
	}
}

func TestPrepareNewBufFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "golang-test")
	if err != nil {
		log.Fatal(err)
	}
	t.Logf("create directory: %v", dir)
	defer os.RemoveAll(dir)

	bufStat, err := journal.PrepareNewBufFile(dir)
	if err != nil {
		t.Fatalf("got error: %+v", err)
	}
	newDataFp, err := os.Create(bufStat.NewDataFName)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer newDataFp.Close()

	newIdsFp, err := os.Create(bufStat.NewIdsDataFname)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer newIdsFp.Close()

	_, err = newDataFp.WriteString("test data")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	_, err = newIdsFp.WriteString("test ids")
	if err != nil {
		t.Fatalf("%+v", err)
	}

	err = newDataFp.Sync()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	err = newIdsFp.Sync()
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

func init() {
	utils.SetupLogger("debug")
}