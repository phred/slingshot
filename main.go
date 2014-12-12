package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"io"
	"os"
)

type Chunk struct {
	Type         string
	Identifier   string
	Count        int
	Index        int
	Size         int
	TotalSize    int
	Filename     string
	ChunkSize    int
	RelativePath string
	Reader io.ReadCloser
}

const (
	Prefix              = "resumable"
	TypeKey             = Prefix + "Type"
	IdentifierKey       = Prefix + "Identifier"
	TotalChunksKey      = Prefix + "TotalChunks"
	ChunkNumKey         = Prefix + "ChunkNumber"
	ChunkSizeKey        = Prefix + "ChunkSize"
	TotalSizeKey        = Prefix + "TotalSize"
	FilenameKey         = Prefix + "Filename"
	CurrentChunkSizeKey = Prefix + "CurrentChunkSize"
	RelativePathKey     = Prefix + "RelativePath"
)

type MyValues struct {
	url.Values
}

func (v MyValues) GetInt(key string) (val int) {
	_, err := fmt.Sscanf(v.Get(key), "%d", &val)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func NewUpload(form MyValues, reader io.ReadCloser) (chunk Chunk) {
	chunk.Type = form.Get(TypeKey)
	chunk.Identifier = form.Get(IdentifierKey)
	chunk.Index = form.GetInt(ChunkNumKey) - 1
	chunk.Size = form.GetInt(CurrentChunkSizeKey)
	chunk.Count = form.GetInt(TotalChunksKey)
	chunk.ChunkSize = form.GetInt(ChunkSizeKey)
	chunk.TotalSize = form.GetInt(TotalSizeKey)
	chunk.Filename = form.Get(FilenameKey)
	chunk.RelativePath = form.Get(RelativePathKey)
	chunk.Reader = reader
	return
}

func parseChunk(req *http.Request, reader io.ReadCloser) (chunk Chunk, err error) {
	chunk = NewUpload(MyValues{req.Form}, reader)
	return
}

func writeChunk(chunk Chunk) (err error) {
	file, err := os.OpenFile(chunk.Identifier, os.O_CREATE | os.O_WRONLY | os.O_SYNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	offset := chunk.Index * chunk.ChunkSize
	endOffset := offset + chunk.Size

	buf := make([]byte, 32*1024)
	for offset < endOffset {
		numRead, err := chunk.Reader.Read(buf)

		log.Printf("%d read, currently at %d", numRead, offset)
		file.WriteAt(buf[0:numRead], int64(offset))
		offset += numRead
		if err != nil {
			log.Fatal(err)
		}
	}
	return
}

func handler(out http.ResponseWriter, req *http.Request) {
	status := http.StatusOK
	defer func() {
		log.Printf("%s %s %d", req.Method, req.URL, status)
	}()

	if req.Method != "POST" {
		status = http.StatusMethodNotAllowed
		http.Error(out, http.StatusText(status), status)
		return
	}
	req.ParseMultipartForm(512)

	file, _, err := req.FormFile("file")
	if err != nil {
		status = http.StatusBadRequest
		http.Error(out, "Missing file upload", status)
		return
	}

	chunk, err := parseChunk(req, file)
	if err != nil {
		status = http.StatusBadRequest
		http.Error(out, "Error parsing upload", status)
		return
	}
	err = writeChunk(chunk)
	if err != nil {
	}

	/*	buf := make([]byte, 8192)
		for bytes, err := file.Read(buf); bytes != 0 && err != nil {
			status = http.StatusInternalServerError
			http.Error(out, "Error reading uploaded file", status)
		}*/

	out.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = fmt.Fprint(out, "OK")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/upload", handler)
	log.Fatal(http.ListenAndServe(":8383", nil))
}
