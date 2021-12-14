package fs

import (
	"archive/zip"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/imgproxy/imgproxy/v3/config"
	"github.com/imgproxy/imgproxy/v3/media_source"
)

type transport struct {
	fs http.Dir
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

func New() transport {
	return transport{fs: http.Dir(config.LocalFileSystemRoot)}
}

func (t transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	path := req.URL.Query().Get("path")
	source := req.URL.Query().Get("source")
	sourceDir := t.fs
	sourceId, err := strconv.Atoi(source)
	if err == nil {
		if dirPath, ok := media_source.MediaSources[sourceId]; ok {
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

	header := make(http.Header)

	//if config.ETagEnabled {
	//	etag := BuildEtag(req.URL.Path, fi)
	//	header.Set("ETag", etag)
	//
	//	if etag == req.Header.Get("If-None-Match") {
	//		return &http.Response{
	//			StatusCode:    http.StatusNotModified,
	//			Proto:         "HTTP/1.0",
	//			ProtoMajor:    1,
	//			ProtoMinor:    0,
	//			Header:        header,
	//			ContentLength: 0,
	//			Body:          nil,
	//			Close:         false,
	//			Request:       req,
	//		}, nil
	//	}
	//}

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.0",
		ProtoMajor:    1,
		ProtoMinor:    0,
		Header:        header,
		ContentLength: length,
		Body:          f,
		Close:         true,
		Request:       req,
	}, nil
}

func BuildEtag(path string, fi fs.FileInfo) string {
	tag := fmt.Sprintf("%s__%d__%d", path, fi.Size(), fi.ModTime().UnixNano())
	hash := md5.Sum([]byte(tag))
	return `"` + string(base64.RawURLEncoding.EncodeToString(hash[:])) + `"`
}
