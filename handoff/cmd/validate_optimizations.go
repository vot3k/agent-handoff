//go:build tools

// This file is part of a validation tool and is not part of the main build.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("🔍 Redis Pool Optimization Validator")
	fmt.Println("=====================================")

	// Create performance validator
	validator, err := handoff.NewPerformanceValidator()
	if err != nil {
		log.Fatalf("Failed to create validator: %v", err)
	}
	defer validator.Close()

	fmt.Println("📊 Running comprehensive performance validation...")
	fmt.Println()

	// Run validation
	report, err := validator.ValidatePerformance()
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Print summary to console
	printSummary(report)

	// Save detailed report
	err = saveReport(report, "redis_optimization_validation_report.json")
	if err != nil {
		log.Printf("Warning: Failed to save report: %v", err)
	} else {
		fmt.Printf("📄 Detailed report saved to: redis_optimization_validation_report.json\n")
	}

	// Exit with appropriate code
	if report.OverallResult == "EXCELLENT - Production Ready" {
		fmt.Println("\n✅ VALIDATION PASSED - Ready for production deployment!")
		os.Exit(0)
	} else if report.FailedTests > 0 {
		fmt.Println("\n❌ VALIDATION FAILED - Issues need to be addressed")
		os.Exit(1)
	} else {
		fmt.Println("\n⚠️ VALIDATION PASSED WITH WARNINGS - Review recommendations")
		// Exit with code 1 for warnings to indicate non-production ready state
		os.Exit(1)
	}
}

func printSummary(report *handoff.ValidationReport) {
	fmt.Printf("📋 Test Results Summary\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Passed: %d\n", report.PassedTests)
	fmt.Printf("Failed: %d\n", report.FailedTests)
	fmt.Printf("Overall Result: %s\n\n", report.OverallResult)

	fmt.Printf("📈 Performance Metrics\n")
	fmt.Printf("====================\n")
	fmt.Printf("Average Improvement: %.2fx\n", report.Summary.AverageImprovement)
	fmt.Printf("Max Throughput Gain: %.2fx\n", report.Summary.MaxThroughputGain)
	fmt.Printf("Connection Efficiency: %.2fx\n", report.Summary.ConnectionEfficiency)
	fmt.Printf("Latency Reduction: %.1f%%\n", report.Summary.LatencyReduction*100)
	fmt.Printf("Error Rate: %.3f%%\n", report.Summary.ErrorRate*100)
	fmt.Printf("Production Readiness: %s\n\n", report.Summary.ProductionReadiness)

	fmt.Printf("🏆 Individual Test Results\n")
	fmt.Printf("=========================\n")
	for _, result := range report.PerformanceGains {
		fmt.Printf("%-20s: %.2fx improvement (%.0f ops vs %.0f ops)\n",
			result.TestName,
			result.ImprovementRatio,
			float64(result.OptimizedOps),
			float64(result.UnoptimizedOps))
	}

	fmt.Printf("\n💡 Recommendations\n")
	fmt.Printf("==================\n")
	for _, rec := range report.Recommendations {
		fmt.Printf("• %s\n", rec)
	}
	fmt.Println()
}

func saveReport(report *handoff.ValidationReport, filename string) error {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create full path
	fullPath := filepath.Join(wd, filename)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Write to file
	err = os.WriteFile(fullPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
