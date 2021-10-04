package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
)

type oneBlkData map[string]string

type fileInfo struct {
	f *os.File
	e uint64
}

type Reporter struct {
	FileType          map[string]*fileInfo
	LatestShardEpoch  uint64
	LatestBeaconEpoch uint64
	Locker            *sync.RWMutex
	Data              map[string]oneBlkData
}

func NewReporter() *Reporter {
	return &Reporter{
		FileType:          map[string]*fileInfo{},
		LatestShardEpoch:  100000,
		LatestBeaconEpoch: 100000,
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

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func needNewFile(fExists bool, fInfo *fileInfo, e uint64, fileName string) bool {
	if !fExists {
		return !FileExists(fileName)
	}

	if fInfo.e != (e / 100) {
		return true
	}
	return false
}

func (r *Reporter) WriteToFile(isShard bool, blockHeight, epoch uint64, fileName string) {
	r.Locker.Lock()
	defer r.Locker.Unlock()
	fInfo, fileExist := r.FileType[fileName]
	var err error
	newFile := fmt.Sprintf("%v%v", epoch/100, fileName)
	if needNewFile(fileExist, fInfo, epoch, newFile) {
		e := epoch / 100
		f, err := os.OpenFile(newFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			panic(err)
		}
		w := csv.NewWriter(f)
		if listKey, ok := ColByFile[fileName]; ok {
			err = w.Write(listKey)
			if err != nil {
				panic(err)
			}
		}
		w.Flush()
		r.FileType[fileName] = &fileInfo{
			f: f,
			e: e,
		}
	} else {
		if !fileExist {
			e := epoch / 100
			f, err := os.OpenFile(newFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				panic(err)
			}
			r.FileType[fileName] = &fileInfo{
				f: f,
				e: e,
			}
		}
	}
	fInfo = r.FileType[fileName]
	w := csv.NewWriter(fInfo.f)
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
