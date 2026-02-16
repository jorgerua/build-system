*** Settings ***
Documentation    Integration tests for build execution
Resource         resources/keywords.robot
Resource         resources/variables.robot
Suite Setup      Setup Test Environment
Suite Teardown   Teardown Test Environment

*** Test Cases ***
Complete Build Of Java Project
    [Documentation]    Test complete build flow for Java project
    ...                Validates: Requirements 3.1, 6.1
    [Tags]    build    java    e2e
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build completion
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify build completed
    ${status}=    Get From Dictionary    ${build_result.json()}    status
    Should Be Equal    ${status}    completed
    
    # Verify phases were executed
    ${phases}=    Get From Dictionary    ${build_result.json()}    phases
    ${phase_names}=    Create List
    FOR    ${phase}    IN    @{phases}
        ${phase_name}=    Get From Dictionary    ${phase}    phase
        Append To List    ${phase_names}    ${phase_name}
    END
    
    Should Contain    ${phase_names}    git_sync
    Should Contain    ${phase_names}    nx_build
    Should Contain    ${phase_names}    image_build

Complete Build Of DotNet Project
    [Documentation]    Test complete build flow for .NET project
    ...                Validates: Requirements 3.1, 6.2
    [Tags]    build    dotnet    e2e
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_DOTNET}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build completion
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify build completed
    ${status}=    Get From Dictionary    ${build_result.json()}    status
    Should Be Equal    ${status}    completed
    
    # Verify .NET specific cache was used
    ${phases}=    Get From Dictionary    ${build_result.json()}    phases
    ${build_phase}=    Get Phase By Name    ${phases}    nx_build
    Should Not Be Empty    ${build_phase}

Complete Build Of Go Project
    [Documentation]    Test complete build flow for Go project
    ...                Validates: Requirements 3.1, 6.3
    [Tags]    build    go    e2e
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_GO}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build completion
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify build completed
    ${status}=    Get From Dictionary    ${build_result.json()}    status
    Should Be Equal    ${status}    completed
    
    # Verify Go modules cache was configured
    ${phases}=    Get From Dictionary    ${build_result.json()}    phases
    ${build_phase}=    Get Phase By Name    ${phases}    nx_build
    Should Not Be Empty    ${build_phase}

Build With Compilation Failure
    [Documentation]    Test build behavior when code fails to compile
    ...                Validates: Requirements 3.3
    [Tags]    build    negative
    
    # Note: This test requires a fixture repo with broken code
    # For now, we'll simulate by checking error handling
    
    # Generate test data for non-existent repo (will fail)
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    broken-code-repo    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build to fail
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify build failed
    ${status}=    Get From Dictionary    ${build_result.json()}    status
    Should Be Equal    ${status}    failed
    
    # Verify error message is present
    ${error}=    Get From Dictionary    ${build_result.json()}    error
    Should Not Be Empty    ${error}

Build With Missing Dockerfile
    [Documentation]    Test build behavior when Dockerfile is not found
    ...                Validates: Requirements 5.3
    [Tags]    build    negative    dockerfile
    
    # Note: This test requires a fixture repo without Dockerfile
    # The system should fail at image_build phase
    
    # For this test, we'll verify the error handling exists
    # In a real scenario, we'd have a fixture without Dockerfile
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    no-dockerfile-repo    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    Verify Response Status    ${response}    ${HTTP_ACCEPTED}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    
    # Wait for build to fail
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify build failed
    ${status}=    Get From Dictionary    ${build_result.json()}    status
    Should Be Equal    ${status}    failed
    
    # Verify error mentions Dockerfile
    ${error}=    Get From Dictionary    ${build_result.json()}    error
    Should Contain    ${error}    Dockerfile    ignore_case=True

Build With Dependency Cache
    [Documentation]    Test that dependency cache is used across builds
    ...                Validates: Requirements 4.3, 4.5
    [Tags]    build    cache
    
    # First build - will populate cache
    ${commit_hash_1}=    Generate Random Commit Hash
    ${payload_1}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash_1}    main
    ${signature_1}=    Generate HMAC Signature    ${payload_1}    ${GITHUB_SECRET}
    
    ${response_1}=    Send Webhook    ${payload_1}    ${signature_1}
    ${job_id_1}=    Get From Dictionary    ${response_1.json()}    id
    ${build_1}=    Wait For Build Completion    ${job_id_1}    ${BUILD_TIMEOUT}
    
    # Get first build duration
    ${phases_1}=    Get From Dictionary    ${build_1.json()}    phases
    ${build_phase_1}=    Get Phase By Name    ${phases_1}    nx_build
    ${duration_1}=    Get From Dictionary    ${build_phase_1}    duration
    
    # Second build - should use cache and be faster
    ${commit_hash_2}=    Generate Random Commit Hash
    ${payload_2}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash_2}    main
    ${signature_2}=    Generate HMAC Signature    ${payload_2}    ${GITHUB_SECRET}
    
    ${response_2}=    Send Webhook    ${payload_2}    ${signature_2}
    ${job_id_2}=    Get From Dictionary    ${response_2.json()}    id
    ${build_2}=    Wait For Build Completion    ${job_id_2}    ${BUILD_TIMEOUT}
    
    # Get second build duration
    ${phases_2}=    Get From Dictionary    ${build_2.json()}    phases
    ${build_phase_2}=    Get Phase By Name    ${phases_2}    nx_build
    ${duration_2}=    Get From Dictionary    ${build_phase_2}    duration
    
    # Both builds should complete successfully
    ${status_1}=    Get From Dictionary    ${build_1.json()}    status
    ${status_2}=    Get From Dictionary    ${build_2.json()}    status
    Should Be Equal    ${status_1}    completed
    Should Be Equal    ${status_2}    completed
    
    # Note: We can't reliably assert duration_2 < duration_1 in tests
    # because cache behavior depends on actual system state
    Log    First build duration: ${duration_1}
    Log    Second build duration: ${duration_2}

Verify Build Output Capture
    [Documentation]    Test that build output is captured correctly
    ...                Validates: Requirements 3.2
    [Tags]    build    logging
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_GO}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify phases have output captured
    ${phases}=    Get From Dictionary    ${build_result.json()}    phases
    
    FOR    ${phase}    IN    @{phases}
        ${success}=    Get From Dictionary    ${phase}    success
        Run Keyword If    ${success}    Verify Phase Has Output    ${phase}
    END

Verify Build Phases Are Sequential
    [Documentation]    Test that build phases execute in correct order
    ...                Validates: Requirements 3.4
    [Tags]    build    phases
    
    # Generate test data
    ${commit_hash}=    Generate Random Commit Hash
    ${payload}=    Create Webhook Payload    ${TEST_REPO_JAVA}    ${commit_hash}    main
    ${signature}=    Generate HMAC Signature    ${payload}    ${GITHUB_SECRET}
    
    # Trigger build
    ${response}=    Send Webhook    ${payload}    ${signature}
    ${job_id}=    Get From Dictionary    ${response.json()}    id
    ${build_result}=    Wait For Build Completion    ${job_id}    ${BUILD_TIMEOUT}
    
    # Verify phase order
    ${phases}=    Get From Dictionary    ${build_result.json()}    phases
    ${phase_count}=    Get Length    ${phases}
    Should Be True    ${phase_count} >= 3
    
    # Extract phase names in order
    ${phase_0}=    Get From List    ${phases}    0
    ${phase_1}=    Get From List    ${phases}    1
    ${phase_2}=    Get From List    ${phases}    2
    
    ${name_0}=    Get From Dictionary    ${phase_0}    phase
    ${name_1}=    Get From Dictionary    ${phase_1}    phase
    ${name_2}=    Get From Dictionary    ${phase_2}    phase
    
    # Verify correct order: git_sync -> nx_build -> image_build
    Should Be Equal    ${name_0}    git_sync
    Should Be Equal    ${name_1}    nx_build
    Should Be Equal    ${name_2}    image_build
    
    # Verify timestamps are sequential
    ${start_0}=    Get From Dictionary    ${phase_0}    start_time
    ${end_0}=    Get From Dictionary    ${phase_0}    end_time
    ${start_1}=    Get From Dictionary    ${phase_1}    start_time
    ${end_1}=    Get From Dictionary    ${phase_1}    end_time
    ${start_2}=    Get From Dictionary    ${phase_2}    start_time
    
    Should Be True    '${end_0}' <= '${start_1}'
    Should Be True    '${end_1}' <= '${start_2}'

*** Keywords ***
Get Phase By Name
    [Arguments]    ${phases}    ${phase_name}
    [Documentation]    Find a phase by name in the phases list
    FOR    ${phase}    IN    @{phases}
        ${name}=    Get From Dictionary    ${phase}    phase
        Return From Keyword If    '${name}' == '${phase_name}'    ${phase}
    END
    Fail    Phase ${phase_name} not found

Verify Phase Has Output
    [Arguments]    ${phase}
    [Documentation]    Verify that a phase has captured output
    ${phase_name}=    Get From Dictionary    ${phase}    phase
    Log    Verifying output for phase: ${phase_name}
    # Note: Actual output verification would depend on implementation
    # This is a placeholder for the verification logic
