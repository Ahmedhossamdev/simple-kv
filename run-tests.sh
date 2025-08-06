#!/bin/bash

# Comprehensive test runner for Simple KV Store
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Test configuration
COVERAGE_THRESHOLD=70
BENCHMARK_ITERATIONS=3
PERFORMANCE_TIMEOUT="30m"

echo "🧪 Simple KV Store - Comprehensive Test Suite"
echo "=============================================="

# Function to print section headers
print_section() {
    echo ""
    echo -e "${BLUE}$1${NC}"
    echo "$(printf '=%.0s' {1..50})"
}

# Function to check command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}❌ $1 not found. Please install it first.${NC}"
        return 1
    fi
}

# Function to run tests with timeout and capture output
run_test() {
    local test_name="$1"
    local test_command="$2"
    local timeout_duration="${3:-5m}"

    echo -e "${YELLOW}⏳ Running $test_name...${NC}"

    if timeout "$timeout_duration" bash -c "$test_command"; then
        echo -e "${GREEN}✅ $test_name passed${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name failed${NC}"
        return 1
    fi
}

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Function to record test result
record_test() {
    if [ $1 -eq 0 ]; then
        ((TESTS_PASSED++))
    else
        ((TESTS_FAILED++))
        FAILED_TESTS+=("$2")
    fi
}

print_section "📋 Pre-flight Checks"

# Check Go installation
if ! check_command go; then
    echo -e "${RED}Go is required but not installed.${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${GREEN}✅ Go version: $GO_VERSION${NC}"

# Check Docker installation
if ! check_command docker; then
    echo -e "${YELLOW}⚠️  Docker not found. Docker tests will be skipped.${NC}"
    DOCKER_AVAILABLE=false
else
    echo -e "${GREEN}✅ Docker available${NC}"
    DOCKER_AVAILABLE=true
fi

print_section "🧹 Setup and Cleanup"
echo "Cleaning previous test artifacts..."
rm -f coverage.out coverage.html benchmark.txt test_results.xml
go clean -testcache

print_section "📦 Dependencies Check"
echo "Downloading Go modules..."
go mod download
go mod tidy

print_section "🔍 Code Quality Checks"

# Format check
echo -e "${YELLOW}⏳ Checking code formatting...${NC}"
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo -e "${RED}❌ The following files are not formatted:${NC}"
    echo "$UNFORMATTED"
    record_test 1 "Code formatting"
else
    echo -e "${GREEN}✅ All files are properly formatted${NC}"
    record_test 0 "Code formatting"
fi

# Go vet
run_test "Go vet" "go vet ./..."
record_test $? "Go vet"

# Check for staticcheck if available
if check_command staticcheck; then
    run_test "Staticcheck" "staticcheck ./..."
    record_test $? "Staticcheck"
fi

# Check for golangci-lint if available
if check_command golangci-lint; then
    run_test "GolangCI-Lint" "golangci-lint run ./..."
    record_test $? "GolangCI-Lint"
fi

print_section "🧪 Unit Tests"

# Run unit tests with coverage
run_test "Unit tests with coverage" "go test -v -race -coverprofile=coverage.out ./store/... ./server/..."
UNIT_TEST_RESULT=$?
record_test $UNIT_TEST_RESULT "Unit tests"

if [ $UNIT_TEST_RESULT -eq 0 ] && [ -f coverage.out ]; then
    # Check coverage
    COVERAGE=$(go tool cover -func=coverage.out | grep total | grep -Eo '[0-9]+\.[0-9]+')
    echo -e "${BLUE}📊 Total test coverage: $COVERAGE%${NC}"

    if (( $(echo "$COVERAGE >= $COVERAGE_THRESHOLD" | bc -l) )); then
        echo -e "${GREEN}✅ Coverage meets threshold ($COVERAGE_THRESHOLD%)${NC}"
        record_test 0 "Coverage threshold"
    else
        echo -e "${RED}❌ Coverage below threshold ($COVERAGE_THRESHOLD%)${NC}"
        record_test 1 "Coverage threshold"
    fi

    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo -e "${BLUE}📈 Coverage report generated: coverage.html${NC}"
fi

print_section "🏁 Race Condition Tests"
run_test "Race condition detection" "go test -race -count=5 ./store/... ./server/..."
record_test $? "Race conditions"

print_section "🔗 Integration Tests"
if [ -f integration_test.go ]; then
    run_test "Integration tests" "go test -v -tags=integration ./integration_test.go"
    record_test $? "Integration tests"
else
    echo -e "${YELLOW}⚠️  integration_test.go not found, skipping integration tests${NC}"
fi

print_section "⚡ Performance Tests"
if [ -f performance_test.go ]; then
    run_test "Performance tests" "go test -v -run='^Test.*' -timeout=$PERFORMANCE_TIMEOUT ./performance_test.go" "$PERFORMANCE_TIMEOUT"
    record_test $? "Performance tests"
else
    echo -e "${YELLOW}⚠️  performance_test.go not found, skipping performance tests${NC}"
fi

print_section "🚀 Benchmark Tests"
run_test "Benchmark tests" "go test -bench=. -benchmem -count=$BENCHMARK_ITERATIONS ./store/... ./server/... | tee benchmark.txt"
record_test $? "Benchmarks"

if [ -f benchmark.txt ]; then
    echo -e "${BLUE}📊 Benchmark results saved to benchmark.txt${NC}"
fi

print_section "🛡️  Security Tests"
if check_command gosec; then
    run_test "Security scan (gosec)" "gosec ./..."
    record_test $? "Security scan"
else
    echo -e "${YELLOW}⚠️  gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest${NC}"
fi

print_section "🐳 Docker Tests"
if [ "$DOCKER_AVAILABLE" = true ]; then
    if [ -f Dockerfile.test ]; then
        run_test "Docker test build" "docker build -f Dockerfile.test -t simple-kv:test ."
        record_test $? "Docker test build"

        run_test "Docker test execution" "docker run --rm simple-kv:test echo 'Docker tests completed'"
        record_test $? "Docker test execution"
    else
        echo -e "${YELLOW}⚠️  Dockerfile.test not found, skipping Docker tests${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Docker not available, skipping Docker tests${NC}"
fi

print_section "🎭 End-to-End Tests"
if [ "$DOCKER_AVAILABLE" = true ] && [ -f test-auto-recovery.sh ]; then
    echo -e "${YELLOW}⏳ Running E2E tests (this may take a while)...${NC}"
    chmod +x test-auto-recovery.sh

    # Run E2E tests with timeout
    if timeout 15m ./test-auto-recovery.sh; then
        echo -e "${GREEN}✅ E2E tests passed${NC}"
        record_test 0 "E2E tests"
    else
        echo -e "${RED}❌ E2E tests failed or timed out${NC}"
        record_test 1 "E2E tests"
    fi
else
    echo -e "${YELLOW}⚠️  E2E tests require Docker and test-auto-recovery.sh${NC}"
fi

print_section "📊 Test Results Summary"

echo "Test execution completed!"
echo ""
echo -e "${GREEN}✅ Tests passed: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Tests failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed tests:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "${RED}  • $test${NC}"
    done
    echo ""
fi

# Overall result
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed successfully!${NC}"
    echo -e "${GREEN}✨ Your code is ready for production!${NC}"
    exit 0
else
    echo -e "${RED}💥 Some tests failed. Please fix the issues before proceeding.${NC}"
    echo ""
    echo -e "${BLUE}💡 Quick fixes:${NC}"
    echo -e "${BLUE}  • Run 'make format' to fix formatting issues${NC}"
    echo -e "${BLUE}  • Run 'make lint' to check for code issues${NC}"
    echo -e "${BLUE}  • Check test logs for specific failures${NC}"
    exit 1
fi
