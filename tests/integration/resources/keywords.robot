*** Settings ***
Library    RequestsLibrary
Library    Collections
Library    String
Library    OperatingSystem

*** Variables ***
${API_BASE_URL}    http://localhost:8080
${NATS_URL}        nats://localhost:4222
${GITHUB_SECRET}   test-secret-key
${AUTH_TOKEN}      test-auth-token

*** Keywords ***
Setup Test Environment
    [Documentation]    Initialize test environment and create HTTP session
    Create Session    api    ${API_BASE_URL}    verify=False

Teardown Test Environment
    [Documentation]    Clean up test environment
    Delete All Sessions

Generate Auth Token
    [Documentation]    Generate a valid authentication token (placeholder)
    [Return]    Bearer ${AUTH_TOKEN}

Generate HMAC Signature
    [Arguments]    ${payload}    ${secret}
    [Documentation]    Generate HMAC-SHA256 signature for webhook payload
    ${signature}=    Evaluate    __import__('hmac').new($secret.encode(), $payload.encode(), __import__('hashlib').sha256).hexdigest()
    [Return]    sha256=${signature}

Create Webhook Payload
    [Arguments]    ${repo_name}    ${commit_hash}    ${branch}=main
    [Documentation]    Create a valid GitHub webhook payload
    ${payload}=    Catenate    SEPARATOR=
    ...    {
    ...    "ref": "refs/heads/${branch}",
    ...    "after": "${commit_hash}",
    ...    "repository": {
    ...    "name": "${repo_name}",
    ...    "full_name": "test-owner/${repo_name}",
    ...    "clone_url": "https://github.com/test-owner/${repo_name}.git",
    ...    "owner": {"login": "test-owner"}
    ...    },
    ...    "head_commit": {
    ...    "id": "${commit_hash}",
    ...    "message": "Test commit",
    ...    "author": {"name": "Test Author", "email": "test@example.com"}
    ...    }
    ...    }
    [Return]    ${payload}

Send Webhook
    [Arguments]    ${payload}    ${signature}
    [Documentation]    Send webhook POST request to API
    ${headers}=    Create Dictionary    
    ...    Content-Type=application/json
    ...    X-Hub-Signature-256=${signature}
    ...    Authorization=Bearer ${AUTH_TOKEN}
    ${response}=    POST On Session    api    /webhook    data=${payload}    headers=${headers}    expected_status=any
    [Return]    ${response}

Send Webhook Without Signature
    [Arguments]    ${payload}
    [Documentation]    Send webhook POST request without signature
    ${headers}=    Create Dictionary    Content-Type=application/json
    ${response}=    POST On Session    api    /webhook    data=${payload}    headers=${headers}    expected_status=any
    [Return]    ${response}

Get Build Status
    [Arguments]    ${build_id}
    [Documentation]    Query build status by ID
    ${headers}=    Create Dictionary    Authorization=Bearer ${AUTH_TOKEN}
    ${response}=    GET On Session    api    /builds/${build_id}    headers=${headers}    expected_status=any
    [Return]    ${response}

Get Build Status Without Auth
    [Arguments]    ${build_id}
    [Documentation]    Query build status without authentication
    ${response}=    GET On Session    api    /builds/${build_id}    expected_status=any
    [Return]    ${response}

List Builds
    [Documentation]    List all builds
    ${headers}=    Create Dictionary    Authorization=Bearer ${AUTH_TOKEN}
    ${response}=    GET On Session    api    /builds    headers=${headers}    expected_status=any
    [Return]    ${response}

Check Health
    [Documentation]    Check API health endpoint
    ${response}=    GET On Session    api    /health    expected_status=any
    [Return]    ${response}

Wait For Build Completion
    [Arguments]    ${build_id}    ${timeout}=300
    [Documentation]    Wait for build to complete (success or failure)
    FOR    ${i}    IN RANGE    ${timeout}
        ${response}=    Get Build Status    ${build_id}
        ${status}=    Get From Dictionary    ${response.json()}    status
        ${is_complete}=    Evaluate    '${status}' in ['completed', 'failed']
        Return From Keyword If    ${is_complete}    ${response}
        Sleep    1s
    END
    Fail    Build did not complete within ${timeout} seconds

Verify Response Status
    [Arguments]    ${response}    ${expected_status}
    [Documentation]    Verify HTTP response status code
    Should Be Equal As Numbers    ${response.status_code}    ${expected_status}

Verify JSON Response
    [Arguments]    ${response}
    [Documentation]    Verify response contains valid JSON
    ${content_type}=    Get From Dictionary    ${response.headers}    Content-Type
    Should Contain    ${content_type}    application/json
    ${json}=    Set Variable    ${response.json()}
    Should Not Be Empty    ${json}

Generate Random Commit Hash
    [Documentation]    Generate a random 40-character hex string (like git commit hash)
    ${hash}=    Evaluate    __import__('secrets').token_hex(20)
    [Return]    ${hash}
