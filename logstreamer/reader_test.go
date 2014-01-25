/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is the Mozilla Foundation.
# Portions created by the Initial Developer are Copyright (C) 2012
# the Initial Developer. All Rights Reserved.
#
# Contributor(s):
#   Ben Bangert (bbangert@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package logstreamer

import (
	gs "github.com/rafrombrc/gospec/src/gospec"
	"os"
	"path/filepath"
	"time"
)

func ReaderSpec(c gs.Context) {
	here, err := os.Getwd()
	c.Assume(err, gs.IsNil)
	dirPath := filepath.Join(here, "testdir", "reader")
	testDirPath := filepath.Join(here, "testdir", "filehandling")

	c.Specify("A journal file can be read", func() {
		l, err := LogstreamLocationFromFile(dirPath + "/location.json")

		// Restore the oldest position
		l.Filename = filepath.Join(testDirPath, "/2010/07/error.log.2")
		l.SeekPosition = 500
		l.Hash = "dc6d00ed4a287968635b8b5b96a505547e9161d3"

		sp := &SortPattern{
			FileMatch:      `/(?P<Year>\d+)/(?P<Month>\d+)/error.log(\.(?P<Seq>\d+))?`,
			Translation:    make(SubmatchTranslationMap),
			Priority:       []string{"Year", "Month", "^Seq"},
			Differentiator: []string{"errorlog"},
		}
		fivey, _ := time.ParseDuration("5y")
		ls := NewLogstreamSet(sp, fivey, testDirPath, dirPath)

		names, err := ls.ScanForLogstreams()
		c.Expect(len(names), gs.Equals, 1)

		stream, ok := ls.GetLogstream("errorlog")
		c.Expect(ok, gs.Equals, true)
		c.Expect(len(stream.logfiles), gs.Equals, 26)

		stream.position = l
		c.Expect(stream.VerifyFileHash(), gs.Equals, true)
		b := make([]byte, 500)
		n, err := stream.Read(b)
		c.Expect(err, gs.IsNil)
		c.Expect(n, gs.Equals, 500)
		c.Expect(string(b[:10]), gs.Equals, "- [05/Feb/")

		// Read the remainder of the file
		n, err = stream.Read(b)
		c.Expect(n, gs.Equals, 160)

		// This should move us to the next file
		n, err = stream.Read(b)
		c.Expect(l.Filename, gs.Equals, filepath.Join(testDirPath, "/2010/07/error.log"))
		c.Expect(err, gs.IsNil)
		c.Expect(n, gs.Equals, 500)
		l.Save()
		c.Expect(l.Hash, gs.Equals, "bc544cb727a809dcf0464da1628aeedd4243e81c")
		c.Expect(string(b[:10]), gs.Equals, "2013-11-05")

		// Now read till we get an err
		var newn int
		for err == nil {
			newn, err = stream.Read(b)
			if newn > 0 {
				n = newn
			}
		}
		// We should be at the last file
		c.Expect(l.Filename, gs.Equals, filepath.Join(testDirPath, "/2013/08/error.log"))
		c.Expect(l.SeekPosition, gs.Equals, int64(1969))
		b = b[:n]
		c.Expect((string(b[len(b)-10:])), gs.Equals, "re.bundle'")
	})

	c.Specify("Multiple journal files can be read", func() {
		var err error
		sp := &SortPattern{
			FileMatch:      `/(?P<Year>\d+)/(?P<Month>\d+)/(?P<Type>.*?).log(\.(?P<Seq>\d+))?`,
			Translation:    make(SubmatchTranslationMap),
			Priority:       []string{"Year", "Month", "^Seq"},
			Differentiator: []string{"Type", "-log"},
		}
		fivey, _ := time.ParseDuration("5y")
		ls := NewLogstreamSet(sp, fivey, testDirPath, dirPath)

		names, err := ls.ScanForLogstreams()
		errs, _ := err.(*MultipleError)
		c.Assume(errs.IsError(), gs.Equals, false)
		c.Expect(len(names), gs.Equals, 2)

		stream, ok := ls.GetLogstream("error-log")
		c.Expect(ok, gs.Equals, true)
		c.Expect(len(stream.logfiles), gs.Equals, 26)

		// Restore the oldest position
		l := stream.position
		l.Filename = filepath.Join(testDirPath, "/2010/07/error.log.2")
		l.SeekPosition = 500
		l.Hash = "dc6d00ed4a287968635b8b5b96a505547e9161d3"

		b := make([]byte, 500)
		n, err := stream.Read(b)
		c.Expect(err, gs.IsNil)
		c.Expect(n, gs.Equals, 500)

		// Now read till we get an err
		var newn int
		for err == nil {
			newn, err = stream.Read(b)
			if newn > 0 {
				n = newn
			}
		}

		// We should be at the last file
		c.Expect(l.Filename, gs.Equals, filepath.Join(testDirPath, "/2013/08/error.log"))
		c.Expect(l.SeekPosition, gs.Equals, int64(1969))
		b = b[:n]
		c.Expect((string(b[len(b)-10:])), gs.Equals, "re.bundle'")

		// Now go through the access log
		stream, ok = ls.GetLogstream("access-log")
		c.Expect(ok, gs.Equals, true)
		c.Expect(len(stream.logfiles), gs.Equals, 26)

		l = stream.position
		l.Filename = filepath.Join(testDirPath, "/2010/05/access.log.3")
		l.SeekPosition = 0
		l.Hash = ""

		// Now read till we get an err
		b = b[:cap(b)]
		err = nil
		for err == nil {
			newn, err = stream.Read(b)
			if newn > 0 {
				n = newn
			}
		}
		l = stream.position
		// We should be at the last file
		c.Expect(l.Filename, gs.Equals, filepath.Join(testDirPath, "/2013/08/access.log"))
		c.Expect(l.SeekPosition, gs.Equals, int64(1174))
		b = b[:n]
		c.Expect((string(b[len(b)-10:])), gs.Equals, "le.bundle'")

	})
}
