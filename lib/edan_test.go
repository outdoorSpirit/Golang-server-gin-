package lib

import (
	//"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEdan_Parse1000TRC(t *testing.T) {
	data, err := ioutil.ReadFile("../data/test/trc/20211201-161653-2112011616.trc")

	if err != nil {
		assert.FailNow(t, err.Error())
	}

	trc, err := ParseTRC(data)

	if err != nil {
		assert.FailNow(t, err.Error())
	}

	assert.EqualValues(t, "2112011616", trc.PatientId)
	assert.EqualValues(t, "2021-12-01 16:16:53", trc.StartTime.Format("2006-01-02 15:04:05"))
	assert.EqualValues(t, time.Second, trc.SamplingTime)
	assert.EqualValues(t, 3146, len(trc.FHR1))
	assert.EqualValues(t, 3146, len(trc.FHR2))
	assert.EqualValues(t, 3146, len(trc.TOCO))
}

func TestEdan_Parse0400TRC(t *testing.T) {
	data, err := ioutil.ReadFile("../data/test/trc/20211124-022546-2111240225.trc")

	if err != nil {
		assert.FailNow(t, err.Error())
	}

	trc, err := ParseTRC(data)

	if err != nil {
		assert.FailNow(t, err.Error())
	}

	assert.EqualValues(t, "2111240225", trc.PatientId)
	assert.EqualValues(t, "2021-11-24 02:25:46", trc.StartTime.Format("2006-01-02 15:04:05"))
	assert.EqualValues(t, time.Second, trc.SamplingTime)
	assert.EqualValues(t, 386, len(trc.FHR1))
	assert.EqualValues(t, 386, len(trc.FHR2))
	assert.EqualValues(t, 386, len(trc.TOCO))
}