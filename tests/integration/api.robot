*** Settings ***
Documentation    Integration tests for API endpoints
Resource         resources/keywords.robot
Resource         resources/variables.robot
Suite Setup      Setup Test Environment
Suite Teardown   Teardown Test Environment

*** Test Cases ***
Query Status Of Existing Build
    [Documentation]    Test querying status of an existing build
    ...                Validates: Requirements 8.1, 8.2
    [Tags]    api    status    happy-path
    
    # Create a build first
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Query the build status
    Sleep    2s    Wait for job to be processed
    ${status_response}=    Get Build Status    ${job_id}
    
    # Verify response
    Verify Response Status    ${status_response}    ${HTTP_OK}
    Verify JSON Response    ${status_response}
    
    # Verify response contains expected fields
    ${job_data}=    Set Variable    ${status_response.json()}
    Dictionary Should Contain Key    ${job_data}    id
    Dictionary Should Contain Key    ${job_data}    status
    Dictionary Should Contain Key    ${job_data}    repository
    Dictionary Should Contain Key    ${job_data}    commit_hash
    Dictionary Should Contain Key    ${job_data}    created_at
    
    # Verify job ID matches
    ${returned_id}=    Get From Dictionary    ${job_data}    id
    Should Be Equal    ${returned_id}    ${job_id}

Query Status Of Non-Existent Build
    [Documentation]    Test querying status of a build that doesn't exist
    ...                Validates: Requirements 8.1, 8.5
    [Tags]    api    status    negative
    
    # Query with non-existent job ID
    ${fake_job_id}=    Set Variable    non-existent-job-id-12345
    ${response}=    Get Build Status    ${fake_job_id}
    
    # Verify 404 response
    Verify Response Status    ${response}    ${HTTP_NOT_FOUND}

List Build History
    [Documentation]    Test listing all builds
    ...                Validates: Requirements 8.2, 8.3
    [Tags]    api    list
    
    # Create multiple builds
    ${commit_hash_1}=    Generate Random Commit Hash
    ${commit_hash_2}=    Generate Random Commit Hash
    
    ${payload_1}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash_1}    main
    ${payload_2}=    Create Webhook Payload    ${TEST_REPO_GO}    ${commit_hash_2}    develop
    
    ${signature_1}=    Generate HMAC Signature    ${payload_1}    ${GITHUB_SECRET}
    ${signature_2}=    Generate HMAC Signature    ${payload_2}    ${GITHUB_SECRET}
    
    ${response_1}=    Send Webhook    ${payload_1}    ${signature_1}
    ${response_2}=    Send Webhook    ${payload_2}    ${signature_2}
    
    ${job_id_1}=    Get From Dictionary    ${response_1.json()}    id
    ${job_id_2}=    Get From Dictionary    ${response_2.json()}    id
    
    # List all builds
    Sleep    2s    Wait for jobs to be processed
    ${list_response}=    List Builds
    
    # Verify response
    Verify Response Status    ${list_response}    ${HTTP_OK}
    Verify JSON Response    ${list_response}
    
    # Verify response is a list
    ${builds}=    Set Variable    ${list_response.json()}
    ${builds_type}=    Evaluate    type($builds).__name__
    Should Be Equal    ${builds_type}    list
    
    # Verify our builds are in the list
    ${build_ids}=    Create List
    FOR    ${build}    IN    @{builds}
        ${build_id}=    Get From Dictionary    ${build}    id
        Append To List    ${build_ids}    ${build_id}
    END
    
    Should Contain    ${build_ids}    ${job_id_1}
    Should Contain    ${build_ids}    ${job_id_2}

Health Check Endpoint
    [Documentation]    Test health check endpoint
    ...                Validates: Requirements 8.1
    [Tags]    api    health
    
    # Check health
    ${response}=    Check Health
    
    # Verify response
    Verify Response Status    ${response}    ${HTTP_OK}
    Verify JSON Response    ${response}
    
    # Verify health status
    ${health_data}=    Set Variable    ${response.json()}
    Dictionary Should Contain Key    ${health_data}    status
    
    ${status}=    Get From Dictionary    ${health_data}    status
    Should Be Equal    ${status}    healthy

Authentication With Valid Token
    [Documentation]    Test API authentication with valid token
    ...                Validates: Requirements 8.4
    [Tags]    api    auth    security
    
    # Create a build
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Query with valid token
    Sleep    2s
    ${status_response}=    Get Build Status    ${job_id}
    
    # Verify successful authentication
    Verify Response Status    ${status_response}    ${HTTP_OK}

Authentication With Invalid Token
    [Documentation]    Test API authentication with invalid token
    ...                Validates: Requirements 8.4
    [Tags]    api    auth    security    negative
    
    # Create a build
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Query without authentication
    Sleep    2s
    ${status_response}=    Get Build Status Without Auth    ${job_id}
    
    # Verify authentication failure
    Verify Response Status    ${status_response}    ${HTTP_UNAUTHORIZED}

Authentication Required For List Endpoint
    [Documentation]    Test that list endpoint requires authentication
    ...                Validates: Requirements 8.4
    [Tags]    api    auth    security    negative
    
    # Try to list builds without authentication
    ${headers}=    Create Dictionary    Content-Type=application/json
    ${response}=    GET On Session    api    /builds    headers=${headers}    expected_status=any
    
    # Verify authentication failure
    Verify Response Status    ${response}    ${HTTP_UNAUTHORIZED}

Verify JSON Content Type In Responses
    [Documentation]    Test that API responses have correct Content-Type
    ...                Validates: Requirements 8.3
    [Tags]    api    content-type
    
    # Create a build
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Query status
    Sleep    2s
    ${status_response}=    Get Build Status    ${job_id}
    
    # Verify Content-Type header
    ${content_type}=    Get From Dictionary    ${status_response.headers}    Content-Type
    Should Contain    ${content_type}    application/json

Verify HTTP Status Codes
    [Documentation]    Test that API returns appropriate HTTP status codes
    ...                Validates: Requirements 8.5
    [Tags]    api    status-codes
    
    # Test 202 Accepted for webhook
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    ${webhook_response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${webhook_response}    ${HTTP_ACCEPTED}
    
    # Test 200 OK for status query
    ${job_id}=    Get From Dictionary    ${webhook_response.json()}    id
    Sleep    2s
    ${status_response}=    Get Build Status    ${job_id}
    Verify Response Status    ${status_response}    ${HTTP_OK}
    
    # Test 404 Not Found for non-existent build
    ${not_found_response}=    Get Build Status    fake-id-12345
    Verify Response Status    ${not_found_response}    ${HTTP_NOT_FOUND}
    
    # Test 401 Unauthorized for missing auth
    ${unauth_response}=    Get Build Status Without Auth    ${job_id}
    Verify Response Status    ${unauth_response}    ${HTTP_UNAUTHORIZED}

Query Build With Complete Information
    [Documentation]    Test that build status includes all required information
    ...                Validates: Requirements 8.2, 8.3
    [Tags]    api    status    detailed
    
    # Create a build
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build to progress
    Sleep    5s
    ${status_response}=    Get Build Status    ${job_id}
    
    # Verify all required fields are present
    ${job_data}=    Set Variable    ${status_response.json()}
    
    # Core fields
    Dictionary Should Contain Key    ${job_data}    id
    Dictionary Should Contain Key    ${job_data}    status
    Dictionary Should Contain Key    ${job_data}    created_at
    
    # Repository information
    Dictionary Should Contain Key    ${job_data}    repository
    ${repo_info}=    Get From Dictionary    ${job_data}    repository
    Dictionary Should Contain Key    ${repo_info}    name
    Dictionary Should Contain Key    ${repo_info}    owner
    Dictionary Should Contain Key    ${repo_info}    url
    
    # Commit information
    Dictionary Should Contain Key    ${job_data}    commit_hash
    Dictionary Should Contain Key    ${job_data}    branch
    
    # Timing information
    ${status}=    Get From Dictionary    ${job_data}    status
    Run Keyword If    '${status}' in ['running', 'completed', 'failed']
    ...    Dictionary Should Contain Key    ${job_data}    started_at
    
    Run Keyword If    '${status}' in ['completed', 'failed']
    ...    Dictionary Should Contain Key    ${job_data}    completed_at
    
    Run Keyword If    '${status}' in ['completed', 'failed']
    ...    Dictionary Should Contain Key    ${job_data}    duration

Verify API Response Time
    [Documentation]    Test that API responds within acceptable time
    ...                Validates: Requirements 8.1
    [Tags]    api    performance
    
    # Measure health check response time
    ${start_time}=    Get Time    epoch
    ${response}=    Check Health
    ${end_time}=    Get Time    epoch
    
    ${response_time}=    Evaluate    ${end_time} - ${start_time}
    
    # Verify response time is under 5 seconds
    Should Be True    ${response_time} < 5
    
    # Verify successful response
    Verify Response Status    ${response}    ${HTTP_OK}
