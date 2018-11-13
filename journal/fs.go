package journal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	utils "github.com/Laisky/go-utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	DataFileNameReg = regexp.MustCompile(`\d{8}_\d{8}\.buf`)
	IdFileNameReg   = regexp.MustCompile(`\d{8}_\d{8}\.ids`)
	layout          = "20060102"
	layoutWithTZ    = "20060102-0700"
)

func PrepareDir(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		utils.Logger.Info("create new directory", zap.String("path", path))
		if err = os.MkdirAll(path, DirMode); err != nil {
			return errors.Wrap(err, "try to create buf directory got error")
		}
		return nil
	} else if err != nil {
		return errors.Wrap(err, "try to check buf directory got error")
	}

	if !info.IsDir() {
		return fmt.Errorf("path `%v` should be directory", path)
	}

	return nil
}

type BufFileStat struct {
	NewDataFp, NewIdsDataFp, NextDataFp, NextIdsDataFp *os.File
	OldDataFnames, OldIdsDataFname                     []string
}

func PrepareNewBufFile(dirPath string, oldFileStat *BufFileStat) (ret *BufFileStat, err error) {
	utils.Logger.Debug("PrepareNewBufFile", zap.String("dirPath", dirPath))
	ret = &BufFileStat{
		OldDataFnames:   []string{},
		OldIdsDataFname: []string{},
	}

	// scan directories
	var (
		latestDataFName, latestIDsFName, nextDataFName, nextIDsFName string
		fname, absFname                                              string
		fs                                                           []os.FileInfo
	)
	fs, err = ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, errors.Wrap(err, "try to list dir got error")
	}

	// scan existing buf files
	for _, f := range fs {
		fname = f.Name()
		absFname = path.Join(dirPath, fname)

		// macos fs bug, could get removed files
		if _, err := os.Stat(absFname); os.IsNotExist(err) {
			utils.Logger.Warn("file not exists", zap.String("fname", absFname))
			return nil, nil
		}

		if DataFileNameReg.MatchString(fname) &&
			(oldFileStat.NextDataFp == nil || absFname < oldFileStat.NextDataFp.Name()) {
			utils.Logger.Debug("add data file into queue", zap.String("fname", absFname))
			ret.OldDataFnames = append(ret.OldDataFnames, absFname)
			if fname > latestDataFName {
				latestDataFName = fname
			}
		} else if IdFileNameReg.MatchString(fname) &&
			(oldFileStat.NextIdsDataFp == nil || absFname < oldFileStat.NextIdsDataFp.Name()) {
			utils.Logger.Debug("add ids file into queue", zap.String("fname", absFname))
			ret.OldIdsDataFname = append(ret.OldIdsDataFname, absFname)
			if fname > latestIDsFName {
				latestIDsFName = fname
			}
		}
	}
	utils.Logger.Debug("got data files", zap.Strings("fs", ret.OldDataFnames))
	utils.Logger.Debug("got ids files", zap.Strings("fs", ret.OldIdsDataFname))

	// generate new buf data file name
	// `latestxxxFName` means new buf file name now
	now := utils.UTCNow()
	if latestDataFName == "" {
		latestDataFName = now.Format(layout) + "_00000001.buf"
		nextDataFName = now.Format(layout) + "_00000002.buf"
	} else if oldFileStat.NextDataFp != nil {
		_, latestDataFName = filepath.Split(oldFileStat.NextDataFp.Name())
		if nextDataFName, err = GenerateNewBufFName(now, latestDataFName); err != nil {
			return nil, errors.Wrapf(err, "generate new data fname `%v` got error", nextDataFName)
		}
	} else if oldFileStat.NextDataFp == nil {
		if latestDataFName, err = GenerateNewBufFName(now, latestDataFName); err != nil {
			return nil, errors.Wrapf(err, "generate new data fname `%v` got error", latestDataFName)
		}
		if nextDataFName, err = GenerateNewBufFName(now, latestDataFName); err != nil {
			return nil, errors.Wrapf(err, "generate new data fname `%v` got error", latestDataFName)
		}
	}

	// generate new buf ids file name
	if latestIDsFName == "" {
		latestIDsFName = now.Format(layout) + "_00000001.ids"
		nextIDsFName = now.Format(layout) + "_00000002.ids"
	} else if oldFileStat.NextIdsDataFp != nil { // update new nextIDsFName
		_, latestIDsFName = filepath.Split(oldFileStat.NextIdsDataFp.Name())
		if nextIDsFName, err = GenerateNewBufFName(now, latestIDsFName); err != nil {
			return nil, errors.Wrapf(err, "generate new ids fname `%v` got error", nextIDsFName)
		}
	} else if oldFileStat.NextIdsDataFp == nil {
		if latestIDsFName, err = GenerateNewBufFName(now, latestIDsFName); err != nil {
			return nil, errors.Wrapf(err, "generate new ids fname `%v` got error", latestIDsFName)
		}
		if nextIDsFName, err = GenerateNewBufFName(now, latestIDsFName); err != nil {
			return nil, errors.Wrapf(err, "generate new ids fname `%v` got error", latestIDsFName)
		}
	}

	utils.Logger.Debug("PrepareNewBufFile",
		zap.String("new ids fname", latestIDsFName),
		zap.String("new data fname", latestDataFName))
	if oldFileStat.NextDataFp != nil {
		ret.NewDataFp = oldFileStat.NextDataFp
	} else {
		utils.Logger.Warn("create buf data file blocking", zap.String("file", latestDataFName))
		if ret.NewDataFp, err = OpenBufFile(path.Join(dirPath, latestDataFName)); err != nil {
			return nil, err
		}
	}

	if oldFileStat.NextIdsDataFp != nil {
		ret.NewIdsDataFp = oldFileStat.NextIdsDataFp
	} else {
		utils.Logger.Warn("create buf ids file blocking", zap.String("file", latestIDsFName))
		if ret.NewIdsDataFp, err = OpenBufFile(path.Join(dirPath, latestIDsFName)); err != nil {
			return nil, err
		}
	}

	go func() {
		if ret.NextDataFp, err = OpenBufFile(path.Join(dirPath, nextDataFName)); err != nil {
			ret.NextDataFp = nil
			utils.Logger.Error("prepare journal next data file got error", zap.Error(err))
		}
		if ret.NextIdsDataFp, err = OpenBufFile(path.Join(dirPath, nextIDsFName)); err != nil {
			ret.NewIdsDataFp = nil
			utils.Logger.Error("prepare journal next ids file got error", zap.Error(err))
		}
	}()

	return ret, nil
}

func OpenBufFile(filepath string) (fp *os.File, err error) {
	utils.Logger.Info("create new buf file", zap.String("fname", filepath))
	if fp, err = os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, FileMode); err != nil {
		return nil, errors.Wrapf(err, "open file got error: %+v", filepath)
	}

	return fp, nil
}

// GenerateNewBufFName return new buf file name depends on current time
// file name looks like `yyyymmddnnnn.ids`, nnnn begin from 0001 for each day
func GenerateNewBufFName(now time.Time, oldFName string) (string, error) {
	utils.Logger.Debug("GenerateNewBufFName", zap.Time("now", now), zap.String("oldFName", oldFName))
	finfo := strings.Split(oldFName, ".") // {name, ext}
	if len(finfo) < 2 {
		return oldFName, fmt.Errorf("oldFname `%v` not correct", oldFName)
	}
	fts := finfo[0][:8]
	fidx := finfo[0][9:]
	fext := finfo[1]

	ft, err := time.Parse(layoutWithTZ, fts+"+0000")
	if err != nil {
		return oldFName, errors.Wrapf(err, "parse buf file name `%v` got error", oldFName)
	}

	if now.Sub(ft) > 24*time.Hour {
		return now.Format(layout) + "_00000001." + fext, nil
	}

	idx, err := strconv.ParseInt(fidx, 10, 64)
	if err != nil {
		return oldFName, errors.Wrapf(err, "parse buf file's idx `%v` got error", fidx)
	}
	return fmt.Sprintf("%v_%08d.%v", fts, idx+1, fext), nil
}
