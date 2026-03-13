package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

const maxBodyBytes = 1024 * 1024

type apiError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

func ReadJSON(w http.ResponseWriter, r *http.Request, data any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(data); err != nil {
		var syntaxErr *json.SyntaxError
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("syntax error in json: %w", err)
		case errors.As(err, &maxErr):
			return fmt.Errorf("request body is too large (maximum 1MB)")
		default:
			return fmt.Errorf("failed to decode JSON: %w", err)
		}
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("invalid json: too many objects")
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("WriteJSON encode error: %v\n", err)
	}
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, errorResponse{
		Error: apiError{
			Code:      status,
			Message:   msg,
			RequestID: uuid.New().String(),
		},
	})
}

func ParsePage(r *http.Request, defaultLimit int) (limit, offset int32) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if lim < 1 {
		lim = defaultLimit
	}
	return int32(lim), int32((page - 1) * lim)
}
