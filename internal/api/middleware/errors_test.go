package middleware

import (
	"encoding/json"
	"errors"
	apierr "i9pay/internal/api/err"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func TestErrors_ValidationErrors_Returns400WithFieldErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type testInput struct {
		Email string `validate:"required"`
	}
	validate := validator.New()
	valErr := validate.Struct(testInput{})

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", Errors(), func(c *gin.Context) {
		c.Error(valErr)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var body map[string][]apierr.ValidationFieldError
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	errs, ok := body["errors"]
	if !ok || len(errs) == 0 {
		t.Fatal("expected 'errors' array in response body")
	}

	found := false
	for _, fe := range errs {
		if fe.Field == "email" && fe.Message == "email is required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected field error for 'email' with 'email is required', got %+v", errs)
	}
}

func TestErrors_AppError_Returns400WithErrorMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", Errors(), func(c *gin.Context) {
		c.Error(apierr.BadRequest("bad request"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	expected := `{"error":"bad request"}`
	if got := w.Body.String(); got != expected {
		t.Fatalf("expected body %s, got %s", expected, got)
	}
}

func TestErrors_UnknownError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", Errors(), func(c *gin.Context) {
		c.Error(errors.New("something went wrong"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	expected := `{"error":"internal server error"}`
	if got := w.Body.String(); got != expected {
		t.Fatalf("expected body %s, got %s", expected, got)
	}
}

func TestErrors_NoErrors_PassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", Errors(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	expected := `{"status":"ok"}`
	if got := w.Body.String(); got != expected {
		t.Fatalf("expected body %s, got %s", expected, got)
	}
}
