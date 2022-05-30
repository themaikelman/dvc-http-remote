package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/atekoa/dvc-http-remote/pkg/pool"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

type StorageSiteLoader interface {
	LoadConfig(local bool) (*pool.ConnectionConfig, error)
}

type Handler struct {
	StorageLoader StorageSiteLoader
}

func (h *Handler) getConnection(params params, w http.ResponseWriter, r *http.Request) (*pool.CloudConn, error) {
	connectionConfig, errLoad := h.StorageLoader.LoadConfig(params.remoteID == 0)
	if errLoad != nil {
		// Write an error and stop the handler chain
		log.
			WithField("remoteID", params.remoteID).
			WithError(errLoad).
			Error("Cannot load configuration")
		http.Error(w, "Cannot load configuration", http.StatusForbidden)
		return nil, errLoad
	}
	remoteType := connectionConfig.Type

	switch remoteType {
	case pool.ConfigTypeAzure:
		connJisap, errGet := connectionConfig.OpenAzure(r.Context())
		if errGet != nil {
			// Write an error and stop the handler chain
			log.
				WithField("remoteID", params.remoteID).
				WithError(errGet).
				Error("Cannot load configuration")
			http.Error(w, "Cannot load configuration", http.StatusForbidden)
			return nil, errGet
		}
		return &connJisap, nil
	case pool.ConfigTypeHttp:
		connJisap, errGet := connectionConfig.OpenHttp(r.Context())
		if errGet != nil {
			// Write an error and stop the handler chain
			log.
				WithField("remoteID", params.remoteID).
				WithError(errGet).
				Error("Cannot load configuration")
			http.Error(w, "Cannot load configuration", http.StatusForbidden)
			return nil, errGet
		}
		return &connJisap, nil
	default:
		log.
			WithField("remoteID", params.remoteID).
			Error("Cannot create connections")
		http.Error(w, "Cannot create connections", http.StatusForbidden)
		return nil, errors.New("Cannot create connections")
	}
}

func (h Handler) HeadFile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))

	params, errVars := parseVars(r)
	if errVars != nil {
		// Write an error and stop the handler chain
		log.
			WithField("err", errVars).
			WithField("key", params.key).
			Error("Cannot parse vars")
		http.Error(w, "Cannot parse vars", http.StatusBadRequest)
		return
	}

	conn, errGet := h.getConnection(params, w, r)
	if errGet != nil {
		// Write an error and stop the handler chain
		log.
			WithError(errGet).
			WithField("key", params.key).
			Error("GetConnection Error")
		http.Error(w, "GetConnection Error", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	blobExists, errEx := conn.Bucket.Exists(r.Context(), params.key)
	if errEx != nil {
		// Write an error and stop the handler chain
		log.
			WithError(errEx).
			WithField("key", params.key).
			Error("Error checking if bucket exists")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if !blobExists {
		log.
			WithField("key", params.key).
			Warn("Blob does not exists")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	attrs, err := conn.Attributes(r.Context(), params.key)
	if gcerrors.Code(err) == gcerrors.NotFound {
		// Write an error and stop the handler chain
		log.
			WithError(err).
			Error("File do not exist")
		http.Error(w, "File do not exist", http.StatusNotFound)
		return
	}
	if err != nil {
		// Write an error and stop the handler chain
		log.
			WithField("Code", gcerrors.Code(err)).
			WithError(err).
			Error("Cannot get Attributes")
		http.Error(w, "Cannot get Attributes", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(attrs.Size, 10))
	w.Header().Set("Content-Type", attrs.ContentType)
	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(attrs.MD5)))
	w.Header().Set("Last-Modified", attrs.ModTime.Format(time.RFC1123))
	w.Header().Set("Cache-Control", attrs.CacheControl)

}

func (h Handler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))

	params, errVars := parseVars(r)
	if errVars != nil {
		// Write an error and stop the handler chain
		log.
			WithField("err", errVars).
			WithField("key", params.key).
			Error("Cannot parse vars")
		http.Error(w, "Cannot parse vars", http.StatusBadRequest)
		return
	}

	conn, errGet := h.getConnection(params, w, r)
	if errGet != nil {
		// Write an error and stop the handler chain
		log.
			WithError(errGet).
			WithField("key", params.key).
			Error("GetConnection Error")
		http.Error(w, "GetConnection Error", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	blobExists, errEx := conn.Bucket.Exists(r.Context(), params.key)
	if errEx != nil {
		// Write an error and stop the handler chain
		log.
			WithError(errEx).
			WithField("key", params.key).
			Error("Error checking if bucket exists")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if !blobExists {
		log.
			WithField("key", params.key).
			Warn("Blob does not exists")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	reader, err := conn.Bucket.NewReader(r.Context(), params.key, &blob.ReaderOptions{})
	if err != nil {
		// Write an error and stop the handler chain
		log.
			WithError(err).
			WithField("key", params.key).
			Error("Error creating Bucket Reader")
		http.Error(w, "Error creating Bucket Reader", http.StatusBadGateway)
		return
	}
	defer reader.Close()

	attrs, err := conn.Attributes(r.Context(), params.key)
	if gcerrors.Code(err) == gcerrors.NotFound {
		// Write an error and stop the handler chain
		log.
			WithError(err).
			Error("File do not exist")
		http.Error(w, "File do not exist", http.StatusNotFound)
		return
	}
	if err != nil {
		// Write an error and stop the handler chain
		log.
			WithField("Code", gcerrors.Code(err)).
			WithError(err).
			Error("Cannot get Attributes")
		http.Error(w, "Cannot get Attributes", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Length", strconv.FormatInt(attrs.Size, 10))
	w.Header().Set("Content-Type", attrs.ContentType)
	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(attrs.MD5)))
	w.Header().Set("Last-Modified", attrs.ModTime.Format(time.RFC1123))
	w.Header().Set("Cache-Control", attrs.CacheControl)

	n, err := reader.WriteTo(w)
	if err != nil {
		switch err {
		case context.Canceled:
			// canceled by user
			log.WithField("key", params.key).
				WithError(err).Error("Download ERROR!")
			return

		case context.DeadlineExceeded:
			// timed out
			log.WithField("key", params.key).
				WithError(err).Error("Download ERROR!")
			return

		default:
			// some other error
			log.WithField("key", params.key).
				WithError(err).Error("Download ERROR!")
			return
		}
	} else {
		log.
			WithField("key", params.key).
			WithField("bytes", n).
			Info("Download FINISH!")
	}
}

func (h Handler) UploadFile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	requestDump, err := httputil.DumpRequest(r, false)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))

	params, errVars := parseVars(r)
	if errVars != nil {
		// Write an error and stop the handler chain
		log.
			WithField("err", errVars).
			WithField("key", params.key).
			Error("Cannot parse vars")
		http.Error(w, "Cannot parse vars", http.StatusBadRequest)
		return
	}

	conn, errGet := h.getConnection(params, w, r)
	if errGet != nil {
		// Write an error and stop the handler chain
		log.
			WithError(errGet).
			WithField("key", params.key).
			Error("GetConnection Error")
		http.Error(w, "GetConnection Error", http.StatusBadGateway)
		return
	}
	defer conn.Close()

	writer, errWriter := conn.Bucket.NewWriter(r.Context(), params.key, &blob.WriterOptions{
		ContentType: params.contentType,
		BufferSize:  getEnvInt("UPLOAD_BUFFER_SIZE"),
	})
	if errWriter != nil {
		log.
			WithError(errWriter).
			WithField("key", params.key).
			Error("Error writting data")
		http.Error(w, "Error writting data", http.StatusBadGateway)
		return
	}

	num_bytes, errCopy := io.Copy(writer, r.Body)
	if errCopy != nil {
		log.
			WithField("key", params.key).
			WithField("Bytes written", num_bytes).
			WithField("Content-Length", r.ContentLength).
			WithError(errCopy).
			Error("Failed to copy content")
		http.Error(w, "Failed to copy content", http.StatusConflict)
		return
	}
	defer writer.Close()

	if r.ContentLength != -1 && int64(num_bytes) != r.ContentLength {
		log.
			WithField("key", params.key).
			WithField("Bytes written", num_bytes).
			WithField("Content-Length", r.ContentLength).
			Warn("Content Length is different from copied bytes")
	} else {
		log.
			WithField("key", params.key).
			WithField("Bytes written", num_bytes).
			WithField("Content-Length", r.ContentLength).
			Info("Upload FINISH!")
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func Attach(r *mux.Router, pathPrefix string, storage StorageSiteLoader) {
	handler := Handler{
		StorageLoader: storage,
	}

	UpDownV1 := r.
		Path(pathPrefix+"/{folder}/{file}").
		Queries("remote", "{remote}").
		Subrouter()
	UpDownV1.Methods("HEAD").HandlerFunc(handler.HeadFile)
	UpDownV1.Methods("GET").HandlerFunc(handler.DownloadFile)
	UpDownV1.Methods("POST").HandlerFunc(handler.UploadFile)

	UpDownV2 := r.
		Path(pathPrefix).
		Queries("remote", "{remote}/{folder}/{file}").
		Subrouter()
	UpDownV2.Methods("HEAD").HandlerFunc(handler.HeadFile)
	UpDownV2.Methods("GET").HandlerFunc(handler.DownloadFile)
	UpDownV2.Methods("POST").HandlerFunc(handler.UploadFile)
}

type params struct {
	remoteID       int
	key            string
	checksum       string
	contentType    string
	acceptEncoding string
	rangeBytes     string
}

func parseVars(r *http.Request) (params, error) {
	params := params{}

	vars := mux.Vars(r)
	folder := vars["folder"]
	file := vars["file"]

	params.key = folder + "/" + file
	params.checksum = folder + strings.TrimSuffix(file, ".dir")

	remote := vars["remote"]
	remoteID, err := strconv.Atoi(remote)
	if err != nil {
		return params, fmt.Errorf("remote id is not a valid integer")
	}
	params.remoteID = remoteID

	params.contentType = r.Header.Get("Content-Type")
	params.acceptEncoding = r.Header.Get("Accept-Encoding")
	params.rangeBytes = r.Header.Get("Range")

	return params, nil
}

func getEnvInt(key string) int {
	value := os.Getenv(key)
	integer, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return integer
}
