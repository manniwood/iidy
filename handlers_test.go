package iidy

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manniwood/iidy/pgstore"
)

type StoreTestingStub struct {
	insertOne      func(ctx context.Context, list string, item string) (int64, error)
	getOne         func(ctx context.Context, list string, item string) (int, bool, error)
	deleteOne      func(ctx context.Context, list string, item string) (int64, error)
	incrementOne   func(ctx context.Context, list string, item string) (int64, error)
	insertBatch    func(ctx context.Context, list string, items []string) (int64, error)
	getBatch       func(ctx context.Context, list string, startID string, count int) ([]pgstore.ListEntry, error)
	deleteBatch    func(ctx context.Context, list string, items []string) (int64, error)
	incrementBatch func(ctx context.Context, list string, items []string) (int64, error)
}

func (sts StoreTestingStub) InsertOne(ctx context.Context, list string, item string) (int64, error) {
	return sts.insertOne(ctx, list, item)
}

func (sts StoreTestingStub) GetOne(ctx context.Context, list string, item string) (int, bool, error) {
	return sts.getOne(ctx, list, item)
}

func (sts StoreTestingStub) DeleteOne(ctx context.Context, list string, item string) (int64, error) {
	return sts.deleteOne(ctx, list, item)
}

func (sts StoreTestingStub) IncrementOne(ctx context.Context, list string, item string) (int64, error) {
	return sts.incrementOne(ctx, list, item)
}

func (sts StoreTestingStub) InsertBatch(ctx context.Context, list string, items []string) (int64, error) {
	return sts.insertBatch(ctx, list, items)
}

func (sts StoreTestingStub) GetBatch(ctx context.Context, list string, startID string, count int) ([]pgstore.ListEntry, error) {
	return sts.getBatch(ctx, list, startID, count)
}

func (sts StoreTestingStub) DeleteBatch(ctx context.Context, list string, items []string) (int64, error) {
	return sts.deleteBatch(ctx, list, items)
}

func (sts StoreTestingStub) IncrementBatch(ctx context.Context, list string, items []string) (int64, error) {
	return sts.incrementBatch(ctx, list, items)
}

func TestHandler(t *testing.T) {
	tests := map[string]struct {
		httpMethod string
		endpoint   string
		mockStore  StoreTestingStub
		wantStatus int
		wantBody   string
	}{
		"InsertOne": {
			httpMethod: http.MethodPost,
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz",
			mockStore: StoreTestingStub{
				insertOne: func(ctx context.Context, list string, item string) (int64, error) {
					return 1, nil
				},
			},
			wantStatus: http.StatusCreated,
			wantBody:   "ADDED 1\n",
		},
		"UnknownMethod": {
			httpMethod: "BLARG",
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz",
			mockStore:  StoreTestingStub{},
			wantStatus: http.StatusBadRequest,
			wantBody:   "Unknown method.\n",
		},
		"GetOne": {
			httpMethod: http.MethodGet,
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz",
			mockStore: StoreTestingStub{
				getOne: func(ctx context.Context, list string, item string) (int, bool, error) {
					return 0, true, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "0\n",
		},
		"GetOne404Item": {
			httpMethod: http.MethodGet,
			endpoint:   "/iidy/v1/lists/downloads/i_do_not_exist.tar.gz",
			mockStore: StoreTestingStub{
				getOne: func(ctx context.Context, list string, item string) (int, bool, error) {
					return 0, false, nil
				},
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found.\n",
		},
		"GetOne404List": {
			httpMethod: http.MethodGet,
			endpoint:   "/iidy/v1/lists/i_to_not_exist/kernel.tar.gz",
			mockStore: StoreTestingStub{
				getOne: func(ctx context.Context, list string, item string) (int, bool, error) {
					return 0, false, nil
				},
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "Not found.\n",
		},
		"IncrementOne": {
			httpMethod: http.MethodPost,
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz?action=increment",
			mockStore: StoreTestingStub{
				incrementOne: func(ctx context.Context, list string, item string) (int64, error) {
					return 1, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "INCREMENTED 1\n",
		},
		"IncrementMissing": {
			httpMethod: http.MethodPost,
			endpoint:   "/iidy/v1/lists/i_do_not_exist/kernel.tar.gz?action=increment",
			mockStore: StoreTestingStub{
				incrementOne: func(ctx context.Context, list string, item string) (int64, error) {
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "INCREMENTED 0\n",
		},
		"DeleteOne": {
			httpMethod: http.MethodDelete,
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz",
			mockStore: StoreTestingStub{
				deleteOne: func(ctx context.Context, list string, item string) (int64, error) {
					return 1, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "DELETED 1\n",
		},
		"DeleteOne404": {
			httpMethod: http.MethodDelete,
			endpoint:   "/iidy/v1/lists/downloads/kernel.tar.gz",
			mockStore: StoreTestingStub{
				deleteOne: func(ctx context.Context, list string, item string) (int64, error) {
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantBody:   "DELETED 0\n",
		},
	}

	for ttName, tt := range tests {
		t.Run(ttName, func(t *testing.T) {
			req, err := http.NewRequest(tt.httpMethod, tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			h := &Handler{Store: tt.mockStore}
			handler := http.Handler(h)
			handler.ServeHTTP(rr, req)
			if gotStatus := rr.Code; gotStatus != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", gotStatus, tt.wantStatus)
			}
			if gotBody := rr.Body.String(); gotBody != tt.wantBody {
				t.Errorf("handler returned unexpected body: got %v want %v", gotBody, tt.wantBody)
			}
		})
	}
}

func TestBatchPostHandler(t *testing.T) {
	var tests = []struct {
		mime           string
		mockStore      StoreTestingStub
		body           []byte
		expectAfterAdd string
		expected       []pgstore.ListEntry
	}{
		{
			mime: "text/plain",
			mockStore: StoreTestingStub{
				insertBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 3, nil
				},
			},
			body: []byte(`kernel.tar.gz
vim.tar.gz
robots.txt`),
			expectAfterAdd: "ADDED 3\n",
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{
				{"kernel.tar.gz", 0},
				{"robots.txt", 0},
				{"vim.tar.gz", 0},
			},
		},
		{
			mime: "application/json",
			mockStore: StoreTestingStub{
				insertBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 3, nil
				},
			},
			body: []byte(`{ "items": ["kernel.tar.gz", "vim.tar.gz", "robots.txt"] }`),
			expectAfterAdd: `{"added":3}
`,
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{
				{"kernel.tar.gz", 0},
				{"robots.txt", 0},
				{"vim.tar.gz", 0},
			},
		},
		{
			mime: "text/plain",
			mockStore: StoreTestingStub{
				insertBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			body:           nil,
			expectAfterAdd: "ADDED 0\n",
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{},
		},
		{
			mime: "application/json",
			mockStore: StoreTestingStub{
				insertBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			body: nil,
			expectAfterAdd: `{"added":0}
`,
			// remember, these come back in alphabetical order
			expected: []pgstore.ListEntry{},
		},
	}

	for _, test := range tests {
		h := &Handler{Store: test.mockStore}

		req, err := http.NewRequest("POST", "/iidy/v1/batch/lists/downloads", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}
		if rr.Body.String() != test.expectAfterAdd {
			t.Errorf(`Unexpected body: got "%v" want "%v"`, rr.Body.String(), test.expectAfterAdd)
		}
	}
}

func TestBatchGetHandler(t *testing.T) {
	// Order of these tests matters. We set up state and go through in order.
	var tests = []struct {
		afterItem string
		want      string
		wantJSON  string
		lastItem  string
		mockStore StoreTestingStub
	}{
		{
			afterItem: "",
			want:      "a 0\nb 0\n",
			wantJSON: `{"listentries":[{"item":"a","attempts":0},{"item":"b","attempts":0}]}
`,
			lastItem: "b",
			mockStore: StoreTestingStub{
				getBatch: func(ctx context.Context, list string, startID string, count int) ([]pgstore.ListEntry, error) {
					return []pgstore.ListEntry{
						pgstore.ListEntry{Item: "a", Attempts: 0},
						pgstore.ListEntry{Item: "b", Attempts: 0},
					}, nil
				},
			},
		},
		{
			afterItem: "b",
			want:      "c 0\nd 0\n",
			wantJSON: `{"listentries":[{"item":"c","attempts":0},{"item":"d","attempts":0}]}
`,
			lastItem: "d",
			mockStore: StoreTestingStub{
				getBatch: func(ctx context.Context, list string, startID string, count int) ([]pgstore.ListEntry, error) {
					return []pgstore.ListEntry{
						pgstore.ListEntry{Item: "c", Attempts: 0},
						pgstore.ListEntry{Item: "d", Attempts: 0},
					}, nil
				},
			},
		},
	}

	for _, mime := range []string{"text/plain", "application/json"} {
		for _, test := range tests {
			var want string
			if mime == "text/plain" {
				want = test.want
			} else {
				want = test.wantJSON
			}

			url := "/iidy/v1/batch/lists/downloads?count=2"
			if test.afterItem != "" {
				url += "&after_id="
				url += test.afterItem
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", mime)
			rr := httptest.NewRecorder()
			h := &Handler{Store: test.mockStore}
			handler := http.Handler(h)
			handler.ServeHTTP(rr, req)
			if status := rr.Code; status != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}
			lastItem := rr.Result().Header.Get("X-IIDY-Last-Item")
			if lastItem != test.lastItem {
				t.Errorf("handler returned wrong last item: got %v want %v", lastItem, test.lastItem)
			}
			if rr.Body.String() != want {
				t.Errorf("handler returned unexpected body: got '%v' want '%v'", rr.Body.String(), want)
			}
		}
	}
}

func TestBatchGetHandlerError(t *testing.T) {
	mockStore := StoreTestingStub{
		getBatch: func(ctx context.Context, list string, startID string, count int) ([]pgstore.ListEntry, error) {
			return []pgstore.ListEntry{}, nil
		},
	}
	// What if we batch get from a list that doesn't exist?
	req, err := http.NewRequest("GET", "/iidy/v1/batch/lists/i_do_not_exist?count=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	h := &Handler{Store: mockStore}
	handler := http.Handler(h)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestBatchIncHandler(t *testing.T) {
	var tests = []struct {
		name      string
		mime      string
		mockStore StoreTestingStub
		body      []byte
		expected  string
	}{
		{
			name: "text",
			mime: "text/plain",
			mockStore: StoreTestingStub{
				incrementBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 5, nil
				},
			},
			body: []byte(`a
b
c
d
e`),
			expected: "INCREMENTED 5\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			mockStore: StoreTestingStub{
				incrementBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 5, nil
				},
			},
			body: []byte(`{ "items": ["a", "b", "c", "d", "e"] }`),
			expected: `{"incremented":5}
`,
		},
	}
	for _, test := range tests {

		// Can we batch increment some of the items' attempts?
		req, err := http.NewRequest("POST", "/iidy/v1/batch/lists/downloads?action=increment", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		h := &Handler{Store: test.mockStore}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}

func TestBatchIncHandlerError(t *testing.T) {
	var tests = []struct {
		name      string
		mime      string
		mockStore StoreTestingStub
		expected  string
	}{
		{
			name: "text",
			mime: "text/plain",
			mockStore: StoreTestingStub{
				incrementBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			expected: "INCREMENTED 0\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			mockStore: StoreTestingStub{
				incrementBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			expected: `{"incremented":0}
`,
		},
	}
	for _, test := range tests {
		// What if we batch increment nothing?
		req, err := http.NewRequest(http.MethodPost, "/iidy/v1/batch/lists/downloads?action=increment", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		h := &Handler{Store: test.mockStore}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}

func TestBatchDelHandler(t *testing.T) {
	var tests = []struct {
		name      string
		mime      string
		mockStore StoreTestingStub
		body      []byte
		expected  string
	}{
		{
			name: "text",
			mime: "text/plain",
			mockStore: StoreTestingStub{
				deleteBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 5, nil
				},
			},
			body: []byte(`a
b
c
d
e`),
			expected: "DELETED 5\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			body: []byte(`{ "items": ["a", "b", "c", "d", "e"] }`),
			mockStore: StoreTestingStub{
				deleteBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 5, nil
				},
			},
			expected: `{"deleted":5}
`,
		},
	}
	for _, test := range tests {
		req, err := http.NewRequest(http.MethodDelete, "/iidy/v1/batch/lists/downloads", bytes.NewBuffer(test.body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		h := &Handler{Store: test.mockStore}
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: handler returned wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: handler returned unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}

func TestBatchDelHandlerError(t *testing.T) {
	var tests = []struct {
		name      string
		mime      string
		mockStore StoreTestingStub
		expected  string
	}{
		{
			name: "text",
			mime: "text/plain",
			mockStore: StoreTestingStub{
				deleteBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			expected: "DELETED 0\n",
		},
		{
			name: "JSON",
			mime: "application/json",
			mockStore: StoreTestingStub{
				deleteBatch: func(ctx context.Context, list string, items []string) (int64, error) {
					return 0, nil
				},
			},
			expected: `{"deleted":0}
`,
		},
	}
	for _, test := range tests {
		h := &Handler{Store: test.mockStore}
		// What if we batch delete nothing?
		req, err := http.NewRequest(http.MethodDelete, "/iidy/v1/batch/lists/downloads", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", test.mime)
		rr := httptest.NewRecorder()
		handler := http.Handler(h)
		handler.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("%s: Wrong status code: got %v want %v", test.name, status, http.StatusOK)
		}
		if rr.Body.String() != test.expected {
			t.Errorf("%s: Unexpected body: got %v want %v", test.name, rr.Body.String(), test.expected)
		}
	}
}
