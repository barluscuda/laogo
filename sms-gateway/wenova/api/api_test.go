package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func validParams() SendSMSParams {
	return SendSMSParams{
		Header:      "WNV-OTP",
		PhoneNumber: "2012345678",
		Message:     "hello",
		Token:       "tok",
		UsePackage:  true,
	}
}

func TestSendSMS_Success(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != sendPath {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"success":true,"code":0,"message":"ok"}`)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	resp, err := c.SendSMS(context.Background(), validParams())
	if err != nil {
		t.Fatalf("SendSMS: %v", err)
	}
	if !resp.Success {
		t.Errorf("Success = false, want true")
	}
	if want := `{"data":{"header":"WNV-OTP","phoneNumber":"2012345678","message":"hello","token":"tok","usePackage":true}}`; string(gotBody) != want {
		t.Errorf("request body = %s\nwant %s", gotBody, want)
	}
}

func TestSendSMS_APIErrorCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"success":false,"code":30105,"message":"SMS package quota is insufficient"}`)
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).SendSMS(context.Background(), validParams())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Code != CodePackageQuotaInsufficient {
		t.Errorf("Code = %d, want %d", apiErr.Code, CodePackageQuotaInsufficient)
	}
}

func TestSendSMS_NonJSONBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "<html>502 Bad Gateway</html>")
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL).SendSMS(context.Background(), validParams())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if want := "502 Bad Gateway"; !contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err.Error(), want)
	}
}

func TestSendSMS_ValidationShortCircuits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be reached on invalid params")
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	cases := map[string]struct {
		mutate func(*SendSMSParams)
		want   error
	}{
		"missing header":  {func(p *SendSMSParams) { p.Header = "" }, ErrMissingHeader},
		"missing phone":   {func(p *SendSMSParams) { p.PhoneNumber = "" }, ErrMissingPhone},
		"missing message": {func(p *SendSMSParams) { p.Message = "" }, ErrMissingMessage},
		"no credential":   {func(p *SendSMSParams) { p.Token = ""; p.ScriptID = 0 }, ErrMissingCredential},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := validParams()
			tc.mutate(&p)
			_, err := c.SendSMS(context.Background(), p)
			if !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
