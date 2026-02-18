# Test Coverage Priority Action Plan

**Target:** Achieve 75% overall test coverage
**Current Status:** Mixed (44.1% - 90.9% across components)

## Priority Matrix

Components are prioritized based on:
- **Business Impact:** How critical is this component to system functionality?
- **Current Coverage:** How far below target is it?
- **Test Complexity:** How difficult is it to test?
- **Risk Level:** What's the impact of bugs in this component?

## 🔴 Priority 1: Critical Components (Immediate Action Required)

### 1.1 Worker Service (44.1% → Target: 75%)
**Gap:** 30.9 percentage points
**Business Impact:** CRITICAL - Core build orchestration
**Risk:** High - Failures affect all builds

**Action Items:**
- [ ] Add integration tests for worker lifecycle (Start/Shutdown)
- [ ] Add tests for job processing pipeline (processJob)
- [ ] Add tests for NATS message handling (handleMessage)
- [ ] Add tests for status publishing (publishJobStatus, publishJobComplete)
- [ ] Mock external dependencies (NATS, Git, NX, Image services)

**Estimated Effort:** 8-10 hours
**Expected Coverage Gain:** +30%

**Test Files to Create:**
- `apps/worker-service/worker_integration_test.go`
- Enhance `apps/worker-service/orchestrator_test.go`

### 1.2 API Service - Middleware (53.8% → Target: 75%)
**Gap:** 21.2 percentage points
**Business Impact:** HIGH - Request logging and auth
**Risk:** Medium - Affects observability and security

**Action Items:**
- [ ] Create `middleware/logging_test.go`
- [ ] Test LoggingMiddleware with normal requests
- [ ] Test LoggingMiddleware with errors
- [ ] Test LoggingMiddleware includes latency
- [ ] Test log format and content
- [ ] Enhance auth middleware tests for edge cases

**Estimated Effort:** 2-3 hours
**Expected Coverage Gain:** +21%

**Test Files to Create:**
- `apps/api-service/middleware/logging_test.go`

### 1.3 Fix Build Failures (Blocking)
**Components Affected:** Cache Service, Git Service, NX Service
**Business Impact:** CRITICAL - Cannot measure coverage
**Risk:** High - Unknown coverage levels

**Action Items:**
- [ ] Run `go mod tidy` in `libs/cache-service`
- [ ] Run `go mod tidy` in `libs/git-service`
- [ ] Run `go mod tidy` in `libs/nx-service`
- [ ] Re-run coverage tests for these components
- [ ] Identify coverage gaps in these services

**Estimated Effort:** 1 hour
**Expected Coverage Gain:** Unknown until measured

## 🟡 Priority 2: High-Impact Components (Next Sprint)

### 2.1 API Service - Handlers (71.3% → Target: 75%)
**Gap:** 3.7 percentage points
**Business Impact:** HIGH - Main API endpoints
**Risk:** Medium - User-facing functionality

**Action Items:**
- [ ] Add error case tests for GetBuildStatus (currently 54.5%)
- [ ] Add pagination tests for ListBuilds (currently 60.9%)
- [ ] Add validation tests for Webhook.Handle (currently 61.3%)
- [ ] Add edge case tests for extractBuildJob (currently 72.2%)

**Estimated Effort:** 3-4 hours
**Expected Coverage Gain:** +4%

**Test Files to Enhance:**
- `apps/api-service/handlers/status_test.go`
- `apps/api-service/handlers/webhook_test.go`

### 2.2 Test Utilities (67.3% → Target: 75%)
**Gap:** 7.7 percentage points
**Business Impact:** MEDIUM - Test infrastructure
**Risk:** Low - Doesn't affect production

**Action Items:**
- [ ] Add tests for unused assertion helpers OR remove them
- [ ] Add tests for unused mock methods OR remove them
- [ ] Document which helpers are intentionally unused
- [ ] Clean up dead code

**Estimated Effort:** 2-3 hours
**Expected Coverage Gain:** +8%

**Test Files to Enhance:**
- `tests/testutil/assertions_test.go`
- `tests/testutil/mocks_test.go`

## 🟢 Priority 3: Measure Unknown Components

### 3.1 Cache Service (Unknown → Target: 75%)
**Business Impact:** MEDIUM - Build caching
**Risk:** Medium - Affects build performance

**Action Items:**
- [ ] Fix build issues (go mod tidy)
- [ ] Run coverage analysis
- [ ] Identify gaps
- [ ] Add missing tests

**Estimated Effort:** 4-6 hours (after build fix)

### 3.2 Git Service (Unknown → Target: 75%)
**Business Impact:** HIGH - Repository management
**Risk:** High - Affects all builds

**Action Items:**
- [ ] Fix build issues (go mod tidy)
- [ ] Run coverage analysis
- [ ] Add tests for SyncRepository edge cases
- [ ] Add tests for clone/pull retry logic
- [ ] Add tests for cache fallback

**Estimated Effort:** 4-6 hours (after build fix)

### 3.3 NX Service (Unknown → Target: 75%)
**Business Impact:** HIGH - Build execution
**Risk:** High - Core functionality

**Action Items:**
- [ ] Fix build issues (go mod tidy)
- [ ] Run coverage analysis
- [ ] Add tests for language detection
- [ ] Add tests for build timeout
- [ ] Add tests for build errors

**Estimated Effort:** 4-6 hours (after build fix)

## 🔵 Priority 4: Already Meeting Target (Maintain)

### 4.1 Shared Library (90.9%) ✅
**Status:** Excellent coverage
**Action:** Maintain current level, add tests for new code

### 4.2 NATS Client (83.1%) ✅
**Status:** Good coverage
**Action:** Maintain current level, add tests for new features

### 4.3 Image Service (80.6%) ✅
**Status:** Above target
**Action:** Maintain current level

## Implementation Roadmap

### Week 1: Critical Fixes
**Goal:** Fix blockers and critical gaps

1. **Day 1-2:** Fix build issues (Priority 1.3)
   - Run go mod tidy on all libs
   - Measure coverage for cache, git, nx services
   
2. **Day 3-4:** Middleware tests (Priority 1.2)
   - Create logging_test.go
   - Achieve 75%+ coverage
   
3. **Day 5:** Handler improvements (Priority 2.1)
   - Add missing error case tests
   - Achieve 75%+ coverage

### Week 2: Worker Service
**Goal:** Bring worker service to 75%

4. **Day 1-3:** Worker lifecycle tests (Priority 1.1)
   - Integration tests for Start/Shutdown
   - Tests for job processing
   
5. **Day 4-5:** Worker message handling (Priority 1.1)
   - Tests for NATS integration
   - Tests for status publishing

### Week 3: Remaining Components
**Goal:** Complete coverage for all components

6. **Day 1-2:** Cache Service tests (Priority 3.1)
7. **Day 3-4:** Git Service tests (Priority 3.2)
8. **Day 5:** NX Service tests (Priority 3.3)

### Week 4: Cleanup and Validation
**Goal:** Verify 75% target met

9. **Day 1-2:** Test utilities cleanup (Priority 2.2)
10. **Day 3-4:** Final coverage analysis
11. **Day 5:** Documentation and reporting

## Success Metrics

### Coverage Targets by Component

| Component | Current | Target | Priority |
|-----------|---------|--------|----------|
| Worker Service | 44.1% | 75% | P1 |
| API Middleware | 53.8% | 75% | P1 |
| Test Utilities | 67.3% | 75% | P2 |
| API Handlers | 71.3% | 75% | P2 |
| Cache Service | TBD | 75% | P3 |
| Git Service | TBD | 75% | P3 |
| NX Service | TBD | 75% | P3 |
| Image Service | 80.6% | 75% | ✅ |
| NATS Client | 83.1% | 75% | ✅ |
| Shared | 90.9% | 75% | ✅ |

### Overall Target
**Goal:** 75% average coverage across all components
**Current:** Cannot calculate (3 components not measured)
**Estimated After Fixes:** ~70-72%
**Estimated After All Work:** 76-78%

## Risk Mitigation

### High-Risk Areas Requiring Extra Attention

1. **Worker Service Job Processing**
   - Complex state management
   - Multiple failure modes
   - Requires comprehensive integration tests

2. **Git Service Repository Sync**
   - Network failures
   - Concurrent access
   - Cache invalidation

3. **NX Service Build Execution**
   - Timeout handling
   - Process management
   - Error propagation

### Testing Strategy

- **Unit Tests:** Fast, isolated, mock dependencies
- **Integration Tests:** Test component interactions
- **Property-Based Tests:** Verify invariants across inputs
- **Table-Driven Tests:** Cover multiple scenarios efficiently

## Notes

- Focus on critical paths first
- Don't sacrifice test quality for coverage numbers
- Tests should catch real bugs, not just increase coverage
- Maintain existing high-coverage components
- Document intentionally untested code (e.g., main functions)

## Next Steps

1. Review and approve this priority plan
2. Fix build issues (Priority 1.3)
3. Begin Priority 1 tasks
4. Track progress weekly
5. Adjust priorities based on findings
