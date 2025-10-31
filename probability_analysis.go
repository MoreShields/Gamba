// Standalone probability analysis tool for Gamba gambling system
// This simulates the exact logic used in gambling_service.go
package main

import (
	"fmt"
	"math"
	"math/rand"
)

func main() {
	fmt.Println("=== Gamba Gambling Probability Analysis ===\n")

	// Test various probabilities
	probabilities := []float64{0.01, 0.05, 0.10, 0.25, 0.50, 0.75, 0.90, 0.95, 0.99}

	for _, prob := range probabilities {
		analyzeProbability(prob, 100000)
	}

	// Detailed analysis of 10% probability
	fmt.Println("\n=== DETAILED 10% PROBABILITY ANALYSIS ===")
	detailedAnalysis(0.10, 100000)
}

// analyzeProb ability simulates many bets at a given probability
// This replicates the exact logic from gambling_service.go line 74:
// won := rand.Float64() < winProbability
func analyzeProbability(winProbability float64, numTrials int) {
	wins := 0

	// Simulate the exact logic from gambling_service.go
	for i := 0; i < numTrials; i++ {
		won := rand.Float64() < winProbability
		if won {
			wins++
		}
	}

	actualWinRate := float64(wins) / float64(numTrials)
	deviation := actualWinRate - winProbability
	deviationPercent := (deviation / winProbability) * 100

	// Calculate chi-squared statistic
	expectedWins := float64(numTrials) * winProbability
	expectedLosses := float64(numTrials) * (1 - winProbability)
	chiSquared := math.Pow(float64(wins)-expectedWins, 2)/expectedWins +
		math.Pow(float64(numTrials-wins)-expectedLosses, 2)/expectedLosses

	fmt.Printf("Probability: %.2f%% | Trials: %d | Wins: %d | Actual: %.4f (%.2f%%) | ",
		winProbability*100, numTrials, wins, actualWinRate, actualWinRate*100)
	fmt.Printf("Deviation: %+.2f%% | χ²: %.2f", deviationPercent, chiSquared)

	// Check if within acceptable range (±2% for most probabilities)
	tolerance := 0.02
	if math.Abs(deviation) <= tolerance {
		fmt.Println(" ✓ PASS")
	} else {
		fmt.Println(" ✗ FAIL")
	}
}

// detailedAnalysis provides in-depth analysis of a specific probability
func detailedAnalysis(winProbability float64, numTrials int) {
	fmt.Printf("\nTesting %.0f%% win probability over %d trials\n", winProbability*100, numTrials)
	fmt.Println("========================================================")

	// Track distribution of random values
	buckets := make([]int, 10)
	wins := 0

	for i := 0; i < numTrials; i++ {
		randomValue := rand.Float64()
		won := randomValue < winProbability

		if won {
			wins++
		}

		// Track which bucket the random value fell into
		bucket := int(randomValue * 10)
		if bucket >= 10 {
			bucket = 9
		}
		buckets[bucket]++
	}

	actualWinRate := float64(wins) / float64(numTrials)

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Expected wins:  %d (%.2f%%)\n", int(float64(numTrials)*winProbability), winProbability*100)
	fmt.Printf("  Actual wins:    %d (%.4f%%)\n", wins, actualWinRate*100)
	fmt.Printf("  Deviation:      %+.4f%% (%+.2f%% relative)\n",
		(actualWinRate-winProbability)*100,
		((actualWinRate-winProbability)/winProbability)*100)

	// Calculate expected value for a 1000 bit bet
	betAmount := int64(1000)
	winAmount := int64(float64(betAmount) * ((1 - winProbability) / winProbability))
	expectedValue := (winProbability * float64(winAmount)) - ((1 - winProbability) * float64(betAmount))

	fmt.Printf("\nFairness Analysis (1000 bit bet):\n")
	fmt.Printf("  Win payout:     %d bits\n", winAmount)
	fmt.Printf("  Expected value: %.2f bits (should be ~0 for fair game)\n", expectedValue)

	// Check distribution uniformity
	fmt.Printf("\nRandom Distribution (each bucket should have ~%d values):\n", numTrials/10)
	expectedPerBucket := float64(numTrials) / 10.0
	for i := 0; i < 10; i++ {
		deviation := float64(buckets[i]) - expectedPerBucket
		deviationPercent := (deviation / expectedPerBucket) * 100
		bar := ""
		barLength := int(float64(buckets[i]) / expectedPerBucket * 20)
		for j := 0; j < barLength; j++ {
			bar += "█"
		}
		fmt.Printf("  [%.1f-%.1f): %6d (%+5.2f%%) %s\n",
			float64(i)/10, float64(i+1)/10, buckets[i], deviationPercent, bar)
	}

	// Chi-squared test for uniformity
	chiSquaredUniform := 0.0
	for i := 0; i < 10; i++ {
		chiSquaredUniform += math.Pow(float64(buckets[i])-expectedPerBucket, 2) / expectedPerBucket
	}

	fmt.Printf("\nStatistical Tests:\n")
	fmt.Printf("  χ² (uniformity): %.2f (should be < 16.92 for 95%% confidence with 9 df)\n", chiSquaredUniform)

	// Calculate how many values fell below the win threshold
	winningRangeCount := 0
	for i := 0; i < int(winProbability*10); i++ {
		winningRangeCount += buckets[i]
	}

	// For 10%, only bucket 0 ([0.0-0.1)) should count as wins
	fmt.Printf("  Values in [0, %.2f): %d (these are wins)\n", winProbability, winningRangeCount)
	fmt.Printf("  Values in [%.2f, 1.0): %d (these are losses)\n", winProbability, numTrials-winningRangeCount)

	fmt.Println("\nConclusion:")
	if math.Abs(actualWinRate-winProbability) <= 0.02 {
		fmt.Printf("  ✓ The %.0f%% probability is ACCURATE - actual win rate is %.4f%% (%.2f%%)\n",
			winProbability*100, actualWinRate*100, actualWinRate*100)
	} else {
		fmt.Printf("  ✗ The %.0f%% probability is INACCURATE - actual win rate is %.4f%% (%.2f%%)\n",
			winProbability*100, actualWinRate*100, actualWinRate*100)
	}

	if math.Abs(expectedValue) <= 1.0 {
		fmt.Println("  ✓ The game is FAIR - expected value is approximately 0")
	} else {
		fmt.Printf("  ✗ The game has a house edge - expected value is %.2f\n", expectedValue)
	}
}
