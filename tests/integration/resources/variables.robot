*** Variables ***
# API Configuration
${API_BASE_URL}        http://localhost:8080
${API_TIMEOUT}         30

# NATS Configuration
${NATS_URL}            nats://localhost:4222

# Authentication
${GITHUB_SECRET}       test-secret-key
${AUTH_TOKEN}          test-auth-token

# Test Data
${TEST_REPO_JAVA}      sample-java-repo
${TEST_REPO_DOTNET}    sample-dotnet-repo
${TEST_REPO_GO}        sample-go-repo

# Timeouts
${BUILD_TIMEOUT}       300
${WEBHOOK_TIMEOUT}     10

# Expected Status Codes
${HTTP_OK}             200
${HTTP_ACCEPTED}       202
${HTTP_BAD_REQUEST}    400
${HTTP_UNAUTHORIZED}   401
${HTTP_NOT_FOUND}      404
${HTTP_SERVER_ERROR}   500
${HTTP_UNAVAILABLE}    503
