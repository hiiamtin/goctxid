#!/bin/bash

# Run all benchmarks and generate comparison report
#
# ⚠️ WARNING: THESE BENCHMARKS ARE UNRELIABLE FOR FRAMEWORK COMPARISON ⚠️
#
# These benchmarks use different testing methods:
# - Fiber uses app.Test() which creates full HTTP connections (high overhead)
# - Gin/Echo use ServeHTTP() which directly calls handlers (low overhead)
#
# This makes Fiber appear artificially slow when real-world load testing
# shows Fiber is actually FASTER than Gin (~28% faster).
#
# For accurate performance comparison, use real load testing tools:
# - k6: https://k6.io/
# - wrk: https://github.com/wg/wrk
# - ab: Apache Bench
#
# Real-world results: https://github.com/hiiamtin/go-vs-java/blob/main/COMPARISON_REPORT.md

set -e

echo "======================================"
echo "goctxid Logger Benchmarks"
echo "======================================"
echo ""
echo "⚠️  WARNING: These benchmarks are NOT reliable for framework comparison!"
echo "   Use k6/wrk/ab for accurate performance testing."
echo ""

RESULTS_FILE="benchmark_results.txt"
> "$RESULTS_FILE"

# Array of all examples
EXAMPLES=(
    "zerolog-gin"
    "zerolog-fiber"
    "zerolog-fibernative"
    "zerolog-echo"
    "zap-gin"
    "zap-fiber"
    "zap-fibernative"
    "zap-echo"
    "slog-gin"
    "slog-fiber"
    "slog-fibernative"
    "slog-echo"
)

# Run benchmarks for each example
for example in "${EXAMPLES[@]}"; do
    if [ -d "$example" ]; then
        echo "Running benchmarks for $example..."
        echo "======================================" >> "$RESULTS_FILE"
        echo "$example" >> "$RESULTS_FILE"
        echo "======================================" >> "$RESULTS_FILE"
        cd "$example"
        go test -bench=. -benchmem >> "../$RESULTS_FILE" 2>&1
        echo "" >> "../$RESULTS_FILE"
        cd ..
        echo "✓ Completed $example"
    else
        echo "⚠ Skipping $example (not found)"
    fi
done

echo ""
echo "======================================"
echo "Benchmark Results Summary"
echo "======================================"
echo ""

# Parse and display results
cat "$RESULTS_FILE" | grep -E "(^===|^Benchmark|^goos|^cpu)" | while read line; do
    echo "$line"
done

echo ""
echo "Full results saved to: $RESULTS_FILE"
echo ""
echo "======================================"
echo "Performance Comparison"
echo "======================================"
echo ""

# Extract BenchmarkHealthCheck results for comparison
echo "BenchmarkHealthCheck (lower is better):"
echo "----------------------------------------"
for example in "${EXAMPLES[@]}"; do
    if [ -d "$example" ]; then
        result=$(grep "BenchmarkHealthCheck-" "$RESULTS_FILE" | grep -A1 "$example" | tail -1 || echo "N/A")
        if [ "$result" != "N/A" ]; then
            ns_op=$(echo "$result" | awk '{print $3}')
            allocs=$(echo "$result" | awk '{print $5}')
            echo "$example: $ns_op ns/op, $allocs B/op"
        fi
    fi
done

echo ""
echo "✅ All benchmarks completed!"

