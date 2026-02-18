package testutil

import (
	"net/http"
	"testing"
	"time"

	"github.com/oci-build-system/libs/shared"
)

func TestAssertBuildJobValid_ValidJob(t *testing.T) {
	job := &shared.BuildJob{
		ID: "test-job-123",
		Repository: shared.RepositoryInfo{
			Name:   "test-repo",
			URL:    "https://github.com/owner/test-repo.git",
			Owner:  "owner",
			Branch: "main",
		},
		CommitHash: "abc123def456",
		Branch:     "main",
		Status:     shared.JobStatusPending,
		CreatedAt:  time.Now(),
	}

	// This should not panic or fail
	AssertBuildJobValid(t, job)
}

func TestAssertHTTPStatus_MatchingCodes(t *testing.T) {
	// This should not fail
	AssertHTTPStatus(t, 200, 200)
}

func TestAssertJSONResponse_ValidJSON(t *testing.T) {
	body := []byte(`{"status":"ok","message":"success"}`)
	
	// This should not fail
	AssertJSONResponse(t, body)
}

func TestAssertHTTPSuccess_SuccessCodes(t *testing.T) {
	// These should not fail
	AssertHTTPSuccess(t, 200)
	AssertHTTPSuccess(t, 201)
	AssertHTTPSuccess(t, 204)
}

func TestAssertHTTPError_ErrorCodes(t *testing.T) {
	// These should not fail
	AssertHTTPError(t, 400)
	AssertHTTPError(t, 404)
	AssertHTTPError(t, 500)
}

func TestAssertContentType_MatchingType(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}
	
	// This should not fail
	AssertContentType(t, headers, "application/json")
}

func TestAssertJSONContentType_ValidJSON(t *testing.T) {
	headers := http.Header{
		"Content-Type": []string{"application/json; charset=utf-8"},
	}
	
	// This should not fail
	AssertJSONContentType(t, headers)
}

func TestAssertErrorResponse_ValidError(t *testing.T) {
	body := []byte(`{"error":"something went wrong"}`)
	
	// This should not fail
	AssertErrorResponse(t, body, "went wrong")
}

func TestAssertRepositoryInfoValid_ValidRepo(t *testing.T) {
	repo := &shared.RepositoryInfo{
		Name:   "test-repo",
		URL:    "https://github.com/owner/test-repo.git",
		Owner:  "owner",
		Branch: "main",
	}
	
	// This should not fail
	AssertRepositoryInfoValid(t, repo)
}

func TestAssertNoError_NoError(t *testing.T) {
	// This should not fail
	AssertNoError(t, nil, "test operation")
}

func TestAssertErrorContains_WithError(t *testing.T) {
	err := &testError{msg: "test error message"}
	
	// This should not fail
	AssertErrorContains(t, err, "error message")
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

