package testutil

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jorgerua/build-system/libs/shared"
	"github.com/stretchr/testify/assert"
)

// AssertBuildJobValid verifica se BuildJob tem todos os campos obrigatórios
func AssertBuildJobValid(t *testing.T, job *shared.BuildJob) {
	t.Helper()

	assert.NotEmpty(t, job.ID, "Job ID should not be empty")
	assert.NotEmpty(t, job.Repository.Name, "Repository name should not be empty")
	assert.NotEmpty(t, job.CommitHash, "Commit hash should not be empty")
	assert.NotEmpty(t, job.Branch, "Branch should not be empty")
	assert.NotZero(t, job.CreatedAt, "CreatedAt should be set")
	assert.True(t, job.Status.IsValid(), "Job status should be valid")
	assert.True(t, job.Repository.IsValid(), "Repository should be valid")
}

// AssertHTTPStatus verifica código de status HTTP
func AssertHTTPStatus(t *testing.T, expected, actual int, msgAndArgs ...interface{}) {
	t.Helper()

	assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertJSONResponse verifica se resposta é JSON válido
func AssertJSONResponse(t *testing.T, body []byte) {
	t.Helper()

	var js map[string]interface{}
	err := json.Unmarshal(body, &js)
	assert.NoError(t, err, "Response should be valid JSON")
}

// AssertJSONResponseWithKey verifica se resposta JSON contém uma chave específica
func AssertJSONResponseWithKey(t *testing.T, body []byte, key string) {
	t.Helper()

	var js map[string]interface{}
	err := json.Unmarshal(body, &js)
	assert.NoError(t, err, "Response should be valid JSON")
	assert.Contains(t, js, key, "Response should contain key: %s", key)
}

// AssertHTTPSuccess verifica se o status code indica sucesso (2xx)
func AssertHTTPSuccess(t *testing.T, statusCode int, msgAndArgs ...interface{}) {
	t.Helper()

	assert.GreaterOrEqual(t, statusCode, 200, msgAndArgs...)
	assert.Less(t, statusCode, 300, msgAndArgs...)
}

// AssertHTTPError verifica se o status code indica erro (4xx ou 5xx)
func AssertHTTPError(t *testing.T, statusCode int, msgAndArgs ...interface{}) {
	t.Helper()

	assert.True(t, statusCode >= 400, "Status code should indicate error (>= 400), got: %d", statusCode)
}

// AssertHTTPClientError verifica se o status code indica erro do cliente (4xx)
func AssertHTTPClientError(t *testing.T, statusCode int, msgAndArgs ...interface{}) {
	t.Helper()

	assert.GreaterOrEqual(t, statusCode, 400, msgAndArgs...)
	assert.Less(t, statusCode, 500, msgAndArgs...)
}

// AssertHTTPServerError verifica se o status code indica erro do servidor (5xx)
func AssertHTTPServerError(t *testing.T, statusCode int, msgAndArgs ...interface{}) {
	t.Helper()

	assert.GreaterOrEqual(t, statusCode, 500, msgAndArgs...)
	assert.Less(t, statusCode, 600, msgAndArgs...)
}

// AssertResponseHeader verifica se um header específico está presente e tem o valor esperado
func AssertResponseHeader(t *testing.T, headers http.Header, key, expectedValue string) {
	t.Helper()

	actualValue := headers.Get(key)
	assert.NotEmpty(t, actualValue, "Header %s should be present", key)
	assert.Equal(t, expectedValue, actualValue, "Header %s should have expected value", key)
}

// AssertContentType verifica se o Content-Type é o esperado
func AssertContentType(t *testing.T, headers http.Header, expectedContentType string) {
	t.Helper()

	AssertResponseHeader(t, headers, "Content-Type", expectedContentType)
}

// AssertJSONContentType verifica se o Content-Type é application/json
func AssertJSONContentType(t *testing.T, headers http.Header) {
	t.Helper()

	contentType := headers.Get("Content-Type")
	assert.Contains(t, contentType, "application/json", "Content-Type should be application/json")
}

// AssertErrorResponse verifica se a resposta contém um campo de erro
func AssertErrorResponse(t *testing.T, body []byte, expectedErrorMsg string) {
	t.Helper()

	var response map[string]interface{}
	err := json.Unmarshal(body, &response)
	assert.NoError(t, err, "Response should be valid JSON")

	errorField, exists := response["error"]
	assert.True(t, exists, "Response should contain 'error' field")

	if expectedErrorMsg != "" {
		assert.Contains(t, errorField, expectedErrorMsg, "Error message should contain expected text")
	}
}

// AssertRepositoryInfoValid verifica se RepositoryInfo é válido
func AssertRepositoryInfoValid(t *testing.T, repo *shared.RepositoryInfo) {
	t.Helper()

	assert.NotEmpty(t, repo.Name, "Repository name should not be empty")
	assert.NotEmpty(t, repo.FullName(), "Repository full name should not be empty")
	assert.NotEmpty(t, repo.URL, "Repository URL should not be empty")
	assert.NotEmpty(t, repo.Owner, "Repository owner should not be empty")
	assert.True(t, repo.IsValid(), "Repository should be valid")
}

// AssertJobStatusValid verifica se JobStatus é válido
func AssertJobStatusValid(t *testing.T, status shared.JobStatus) {
	t.Helper()

	assert.True(t, status.IsValid(), "Job status should be valid: %s", status)
}

// AssertLanguageValid verifica se Language é válida
func AssertLanguageValid(t *testing.T, language shared.Language) {
	t.Helper()

	validLanguages := []shared.Language{
		shared.LanguageJava,
		shared.LanguageDotNet,
		shared.LanguageGo,
	}

	assert.Contains(t, validLanguages, language, "Language should be valid: %s", language)
}

// AssertNoError é um wrapper para assert.NoError com mensagem customizada
func AssertNoError(t *testing.T, err error, operation string) {
	t.Helper()

	assert.NoError(t, err, "Operation '%s' should not return error", operation)
}

// AssertError verifica se um erro foi retornado
func AssertError(t *testing.T, err error, operation string) {
	t.Helper()

	assert.Error(t, err, "Operation '%s' should return error", operation)
}

// AssertErrorContains verifica se o erro contém uma mensagem específica
func AssertErrorContains(t *testing.T, err error, expectedMsg string) {
	t.Helper()

	assert.Error(t, err, "Error should not be nil")
	assert.Contains(t, err.Error(), expectedMsg, "Error message should contain expected text")
}
