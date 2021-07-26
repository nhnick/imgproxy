package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type fsTransport struct {
	fs http.Dir
}

func newFsTransport() fsTransport {
	return fsTransport{fs: http.Dir(conf.LocalFileSystemRoot)}
}

type ZipFile struct {
	zf *zip.ReadCloser
	f  io.ReadCloser
	io.ReadCloser
}

func NewZipFile(zf *zip.ReadCloser, f io.ReadCloser) *ZipFile {
	return &ZipFile{zf: zf, f: f}
}

func (z ZipFile) Read(p []byte) (n int, err error) {
	return z.f.Read(p)
}

func (z ZipFile) Close() error {
	defer z.zf.Close()
	return z.f.Close()
}

func (t fsTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	path := req.URL.Query().Get("path")
	source := req.URL.Query().Get("source")
	sourceDir := t.fs
	sourceId, err := strconv.Atoi(source)
	if err == nil {
		if dirPath, ok := MediaSources[sourceId]; ok {
			sourceDir = http.Dir(dirPath)
		}
	}
	var f io.ReadCloser
	var length int64
	var zf *zip.ReadCloser
	if len(path) > 0 && strings.HasSuffix(req.URL.Path, ".zip") {
		zf, err = zip.OpenReader(string(sourceDir) + req.URL.Path)
		if err != nil {
			return nil, err
		}
		for _, ff := range zf.File {
			if ff.Name == path {
				ffff, err := ff.Open()
				if err != nil {
					return nil, err
				}
				f = NewZipFile(zf, ffff)
				length = int64(ff.UncompressedSize64)
				break
			}
		}
	} else {
		ff, err := sourceDir.Open(req.URL.Path)

		if err != nil {
			return nil, err
		}

		fi, err := ff.Stat()
		if err != nil {
			return nil, err
		}

		if fi.IsDir() {
			return nil, fmt.Errorf("%s is a directory", req.URL.Path)
		}
		length = fi.Size()
		f = ff

	}
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        make(http.Header),
		ContentLength: length,
		Body:          f,
		Close:         true,
		Request:       req,
	}, nil
}
