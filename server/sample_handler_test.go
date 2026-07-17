package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterSampleLifecycleAndEnvelope(t *testing.T) {
	r := NewRouter(NewMemoryStore(), newMemoryIdempotency())
	create := func(body, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/samples", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	if res := create(`{"sampleType":"血液","tests":["血常规"]}`, "handler-sample-invalid"); res.Code != http.StatusBadRequest {
		t.Fatalf("missing alias status = %d, body=%s", res.Code, res.Body.String())
	}
	first := create(`{"subjectAlias":"受检者-HTTP","sampleType":"血液","tests":["血常规"]}`, "handler-sample-create")
	if first.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body=%s", first.Code, first.Body.String())
	}
	var envelope struct {
		Code    int    `json:"code"`
		TraceID string `json:"traceId"`
		Data    Sample `json:"data"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 || envelope.TraceID == "" || envelope.Data.ID == "" {
		t.Fatalf("bad create envelope: %+v", envelope)
	}
	duplicate := create(`{"subjectAlias":"受检者-HTTP","sampleType":"血液","tests":["血常规"]}`, "handler-sample-create")
	var duplicateEnvelope struct {
		Data Sample `json:"data"`
	}
	if err := json.Unmarshal(duplicate.Body.Bytes(), &duplicateEnvelope); err != nil {
		t.Fatal(err)
	}
	if duplicate.Code != http.StatusCreated || duplicateEnvelope.Data.ID != envelope.Data.ID {
		t.Fatalf("duplicate create = %d/%s", duplicate.Code, duplicate.Body.String())
	}
	id := envelope.Data.ID
	postAction := func(path, body, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/samples/"+id+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	if res := postAction("/review", `{"actor":"审核员"}`, "handler-sample-early-review"); res.Code != http.StatusConflict {
		t.Fatalf("early review status = %d, body=%s", res.Code, res.Body.String())
	}
	if res := postAction("/receive", `{"actor":""}`, "handler-sample-empty-actor"); res.Code != http.StatusBadRequest {
		t.Fatalf("empty actor status = %d, body=%s", res.Code, res.Body.String())
	}
	for _, step := range []struct {
		path string
		body string
		key  string
	}{
		{"/receive", `{"actor":"收样员"}`, "handler-sample-receive"},
		{"/start-test", `{"actor":"检验员"}`, "handler-sample-start"},
	} {
		if res := postAction(step.path, step.body, step.key); res.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body=%s", step.path, res.Code, res.Body.String())
		}
	}
	if res := postAction("/report", `{"result":"阴性","remark":"结果稳定"}`, "handler-sample-report"); res.Code != http.StatusOK {
		t.Fatalf("report status = %d, body=%s", res.Code, res.Body.String())
	}
	if res := postAction("/review", `{"actor":"审核员"}`, "handler-sample-review"); res.Code != http.StatusOK {
		t.Fatalf("review status = %d, body=%s", res.Code, res.Body.String())
	}
	if res := postAction("/archive", `{"actor":"归档员"}`, "handler-sample-archive"); res.Code != http.StatusOK {
		t.Fatalf("archive status = %d, body=%s", res.Code, res.Body.String())
	}
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/samples/"+id, nil)
	detailRes := httptest.NewRecorder()
	r.ServeHTTP(detailRes, detailReq)
	if detailRes.Code != http.StatusOK || !bytes.Contains(detailRes.Body.Bytes(), []byte("已归档")) || !bytes.Contains(detailRes.Body.Bytes(), []byte("提交报告")) {
		t.Fatalf("detail response = %d, body=%s", detailRes.Code, detailRes.Body.String())
	}
	missingReq := httptest.NewRequest(http.MethodGet, "/api/v1/samples/SM-missing", nil)
	missingRes := httptest.NewRecorder()
	r.ServeHTTP(missingRes, missingReq)
	if missingRes.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, body=%s", missingRes.Code, missingRes.Body.String())
	}
}
