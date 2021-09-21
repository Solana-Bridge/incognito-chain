package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

type oneBlkData map[string]string

type Reporter struct {
	FileType          map[string]*os.File
	LatestShardEpoch  uint64
	LatestBeaconEpoch uint64
	Locker            *sync.RWMutex
	Data              map[string]oneBlkData
}

func NewReporter() *Reporter {
	return &Reporter{
		FileType:          map[string]*os.File{},
		LatestShardEpoch:  0,
		LatestBeaconEpoch: 0,
		Locker:            &sync.RWMutex{},
		Data:              map[string]oneBlkData{},
	}
}

func (r *Reporter) RecordData(blockHeight uint64, fileName, key, value string) {
	dataKey := fmt.Sprintf("%v%v", fileName, blockHeight)
	r.Locker.Lock()
	if _, ok := r.Data[dataKey]; ok {
		r.Data[dataKey][key] = value
	} else {
		blkData := oneBlkData{}
		blkData[key] = value
		r.Data[dataKey] = blkData
	}
	r.Locker.Unlock()
}

func (r *Reporter) WriteToFile(isShard bool, blockHeight, epoch uint64, fileName string) {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	latestEpoch := r.LatestShardEpoch
	if !isShard {
		latestEpoch = r.LatestBeaconEpoch
	}
	f, fileExist := r.FileType[fileName]
	var err error
	if ((epoch / 100) != (latestEpoch / 100)) || (!fileExist) {
		e := epoch / 100
		newFile := fmt.Sprintf("%v%v", e, fileName)
		f, err = os.OpenFile(newFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
		r.FileType[fileName] = f
		if isShard {
			r.LatestShardEpoch = epoch
		} else {
			r.LatestBeaconEpoch = epoch
		}
	}
	w := csv.NewWriter(f)
	if listKey, ok := ColByFile[fileName]; ok {
		dataKey := fmt.Sprintf("%v%v", fileName, blockHeight)
		vals := []string{}
		for _, k := range listKey {
			if blkData, ok1 := r.Data[dataKey]; ok1 {
				if blkValue, ok2 := blkData[k]; ok2 {
					vals = append(vals, blkValue)
					continue
				}
			}
			vals = append(vals, "")
		}
		err = w.Write(vals)
		if err != nil {
			panic(err)
		}
		w.Flush()
	}
}
