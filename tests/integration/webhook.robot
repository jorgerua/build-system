*** Settings ***
Documentation    Integration tests for webhook handling
Resource         resources/keywords.robot
Resource         resources/variables.robot
Suite Setup      Setup Test Environment
Suite Teardown   Teardown Test Environment

*** Test Cases ***
Send Valid Webhook And Verify Enqueuing
    [Documentation]    Test sending a valid webhook and verify it gets enqueued
    ...                Validates: Requirements 1.1, 1.2
    [Tags]    webhook    happy-path
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Send webhook
    ${response}=    Send Webhook    ${payload}    ${signature}
    
    # Verify response
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    Verify JSON Response    ${response}
    
    # Verify job ID is returned
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    Should Not Be Empty    ${job_id}
    
    # Verify job was enqueued
    Sleep    2s    Wait for job to be processed
    ${status_response}=    Get Build Status    ${job_id}
    Verify Response Status    ${status_response}    ${HTTP_OK}
    ${status}=    Get From Dictionary    ${status_response.json()}    status
    Should Be True    '${status}' in ['pending', 'running', 'completed', 'failed']

Send Webhook With Invalid Signature
    [Documentation]    Test sending a webhook with invalid HMAC signature
    ...                Validates: Requirements 1.1, 1.3
    [Tags]    webhook    security    negative
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${invalid_signature}=    Set Variable    sha256=invalid_signature_here
    
    # Send webhook with invalid signature
    ${response}=    Send Webhook    ${payload}    ${invalid_signature}
    
    # Verify rejection
    Verify Response Status    ${response}    ${HTTP_UNAUTHORIZED}

Send Webhook Without Signature
    [Documentation]    Test sending a webhook without signature header
    ...                Validates: Requirements 1.1, 1.3
    [Tags]    webhook    security    negative
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    
    # Send webhook without signature
    ${response}=    Send Webhook Without Signature    ${payload}
    
    # Verify rejection
    Verify Response Status    ${response}    ${HTTP_UNAUTHORIZED}

Send Multiple Webhooks Simultaneously
    [Documentation]    Test sending multiple webhooks at the same time
    ...                Validates: Requirements 1.5
    [Tags]    webhook    concurrency
    
    # Generate test data for multiple webhooks
    ${commit_hash_1}=    Generate Random Commit Hash
    ${commit_hash_2}=    Generate Random Commit Hash
    ${commit_hash_3}=    Generate Random Commit Hash
    
    ${payload_1}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash_1}    main
    ${payload_2}=    Create Webhook Payload    ${TEST_REPO_DOTNET}    ${commit_hash_2}    develop
    ${payload_3}=    Create Webhook Payload    ${TEST_REPO_GO}    ${commit_hash_3}    feature
    
    ${signature_1}=    Generate HMAC Signature    ${payload_1}    ${GITHUB_SECRET}
    ${signature_2}=    Generate HMAC Signature    ${payload_2}    ${GITHUB_SECRET}
    ${signature_3}=    Generate HMAC Signature    ${payload_3}    ${GITHUB_SECRET}
    
    # Send webhooks (Robot Framework executes sequentially, but API should queue them)
    ${response_1}=    Send Webhook    ${payload_1}    ${signature_1}
    ${response_2}=    Send Webhook    ${payload_2}    ${signature_2}
    ${response_3}=    Send Webhook    ${payload_3}    ${signature_3}
    
    # Verify all were accepted
    Verify Response Status    ${response_1}    ${HTTP_ACCEPTED}
    Verify Response Status    ${response_2}    ${HTTP_ACCEPTED}
    Verify Response Status    ${response_3}    ${HTTP_ACCEPTED}
    
    # Verify all job IDs are unique
    ${job_id_1}=    Get From Dictionary    ${response_1.json()}    id
    ${job_id_2}=    Get From Dictionary    ${response_2.json()}    id
    ${job_id_3}=    Get From Dictionary    ${response_3.json()}    id
    
    Should Not Be Equal    ${job_id_1}    ${job_id_2}
    Should Not Be Equal    ${job_id_2}    ${job_id_3}
    Should Not Be Equal    ${job_id_1}    ${job_id_3}
    
    # Verify all jobs are tracked
    Sleep    2s    Wait for jobs to be processed
    ${status_1}=    Get Build Status    ${job_id_1}
    ${status_2}=    Get Build Status    ${job_id_2}
    ${status_3}=    Get Build Status    ${job_id_3}
    
    Verify Response Status    ${status_1}    ${HTTP_OK}
    Verify Response Status    ${status_2}    ${HTTP_OK}
    Verify Response Status    ${status_3}    ${HTTP_OK}

Verify Webhook Payload Parsing
    [Documentation]    Test that webhook payload is correctly parsed
    ...                Validates: Requirements 1.2
    [Tags]    webhook    parsing
    
    # Generate test data with specific values
    ${commit_hash}=    Set Variable    abc123def456789012345678901234567890abcd
    ${repo_name}=    Set Variable    test-parsing-repo
    ${branch}=    Set Variable    feature/test-branch
    
    ${payload}=    Create Webhook Payload    ${repo_name}    ${commit_hash}    ${branch}
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Send webhook
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    
    # Get job details
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    Sleep    2s    Wait for job to be processed
    ${status_response}=    Get Build Status    ${job_id}
    
    # Verify parsed data
    ${job_data}=    Set Variable    ${status_response.json()}
    ${repo_info}=    Get From Dictionary    ${job_data}    repository
    
    ${parsed_name}=    Get From Dictionary    ${repo_info}    name
    ${parsed_owner}=    Get From Dictionary    ${repo_info}    owner
    ${parsed_commit}=    Get From Dictionary    ${job_data}    commit_hash
    ${parsed_branch}=    Get From Dictionary    ${job_data}    branch
    
    Should Be Equal    ${parsed_name}    ${repo_name}
    Should Be Equal    ${parsed_owner}    test-owner
    Should Be Equal    ${parsed_commit}    ${commit_hash}
    Should Contain    ${parsed_branch}    ${branch}

Send Webhook With Malformed JSON
    [Documentation]    Test sending a webhook with malformed JSON payload
    ...                Validates: Requirements 1.3
    [Tags]    webhook    negative
    
    # Create malformed JSON
    ${malformed_payload}=    Set Variable    {"ref": "refs/heads/main", "after": "abc123"
    ${signature}=    Generate HMAC Signature    ${malformed_payload}    ${GITHUB_SECRET}
    
    # Send webhook
    ${response}=    Send Webhook    ${malformed_payload}    ${signature}
    
    # Verify rejection (should be 400 Bad Request)
    Should Be True    ${response.status_code} in [${HTTP_BAD_REQUEST}, ${HTTP_UNAUTHORIZED}]
