package common

import (
	"math"
	"sort"
)

const (
	MedianAbsoluteDeviationZscoreConst float64 = 1.486
	MeanAbsoluteDeviationZscoreConst   float64 = 1.253314
	ZscoreOutlierThreshold             float64 = 3
	ExpectedThreshold                  float64 = .5
	SumRatioThreshold                  float64 = .8
	SetRatioThreshold                  float64 = .6

	representationBalanced = "representation_balanced"
	representationUnder    = "representation_under"
	representationOver     = "representation_over"
	representationZero     = "n/a"
)

func Median(lists ...[]int) []float64 {
	result := []float64{}
	for _, list := range lists {
		list_length := len(list)
		if list_length == 0 {
			result = append(result, 0)
		}
		sort.Ints(list)
		if list_length%2 == 0 {
			result = append(result, (float64(list[list_length/2-1]+list[list_length/2]))/2)
		} else {
			result = append(result, float64(list[(list_length-1)/2]))
		}
	}
	return result
}

func MeanInts(nums []int) float64 {
	sum := 0
	for _, num := range nums {
		sum += num
	}
	return float64(sum) / float64(len(nums))
}

func MeanFloats(nums []float64) float64 {
	sum := 0.0
	for _, num := range nums {
		sum += num
	}
	return float64(sum) / float64(len(nums))
}

func Std(nums []int, mean float64) float64 {
	var sum float64 = 0
	for _, num := range nums {
		sum += math.Pow((float64(num) - mean), 2)
	}
	sum = sum / float64(len(nums))
	return math.Sqrt(sum)
}

func AbsoluteDistances(nums []int, target float64) []float64 {
	results := []float64{}
	for _, num := range nums {
		results = append(results, math.Abs(float64(num)-target))
	}
	return results
}

func medianAbsoluteDeviation(nums []int, median float64) float64 {
	absDistances := AbsoluteDistances(nums, median)
	sort.Float64s(absDistances)
	length := len(absDistances)
	if length%2 == 1 {
		return float64(absDistances[length/2])
	} else {
		return float64(absDistances[length/2]+absDistances[length/2-1]) / 2
	}
}

func meanAbsoluteDeviation(nums []int) float64 {
	mean := MeanInts(nums)
	absDistancesMean := AbsoluteDistances(nums, mean)
	meanAbsoluteDeviation := MeanFloats(absDistancesMean)
	return meanAbsoluteDeviation
}

func checkForBisect(nums []int) (bool, int, int) {
	found := make(map[int]bool)
	for _, num := range nums {
		found[num] = true
	}

	low := 0
	high := 0
	if len(found) == 2 {
		for k := range found {
			if k > high {
				low = high
				high = k
			} else {
				low = k
			}
		}
		return true, low, high
	}

	return false, 0, 0
}

func Zscores(nums []int) (zscores []float64) {
	// Empty list check
	if len(nums) == 0 {
		return []float64{}
	}

	// Always will be 0 with 1 number
	if len(nums) == 1 {
		return []float64{0}
	}

	numsList := make([]int, len(nums))
	copy(numsList, nums)

	// Find median and median absolute deviation
	median := float64(Median(numsList)[0])
	medianAbsoluteDeviation := medianAbsoluteDeviation(numsList, median)

	// If median absolute deviation is 0, we need to use mean absolute deviation instead.
	if medianAbsoluteDeviation == 0 {
		meanAbsoluteDeviation := meanAbsoluteDeviation(numsList)
		// Would all have to be the same number. So will all be balanced.
		if meanAbsoluteDeviation == 0 {
			return zscores
		}
		// Calculate zscores with mean absolute deviation.
		for _, num := range nums {
			zscore := (float64(num) - median) / (meanAbsoluteDeviation * MeanAbsoluteDeviationZscoreConst)
			zscores = append(zscores, zscore)
		}

	} else {
		// Calculate zscores with median absolute deviation
		for _, num := range nums {
			zscore := (float64(num) - median) / (medianAbsoluteDeviation * MedianAbsoluteDeviationZscoreConst)
			zscores = append(zscores, zscore)
		}
	}

	return zscores
}

func FindBalances(nums []int) []string {
	balances := make([]string, len(nums))

	// Exclude 0's
	numsClean := []int{}
	for _, x := range nums {
		if x != 0 {
			numsClean = append(numsClean, x)
		}
	}

	// Check if there are only two numbers split b/c this will always be balanced.
	isBisect, low, high := checkForBisect(numsClean)
	if isBisect {
		// Check if the 2 numbers are close in value or not.
		bisectSpread := float64(high) * ExpectedThreshold
		var lowIsClose bool
		if (float64(high) - bisectSpread) <= float64(low) {
			lowIsClose = true
		} else {
			lowIsClose = false
		}

		for i, num := range nums {
			if num == 0 {
				balances[i] = representationZero
			} else if num == high {
				balances[i] = representationBalanced
			} else {
				if lowIsClose {
					balances[i] = representationBalanced
				} else {
					balances[i] = representationUnder
				}
			}
		}
		return balances
	}

	// Find zscores
	zscores := Zscores(numsClean)

	// If no zscores, then all numbers are equal and return balanced.
	if len(zscores) == 0 {
		for i := range nums {
			if nums[i] == 0 {
				balances[i] = representationZero
			} else {
				balances[i] = representationBalanced
			}
		}
		return balances
	}

	sumAll := 0
	sumNonOutliers := 0
	numberNonOutliers := 0

	for i, zscore := range zscores {
		if !(zscore >= ZscoreOutlierThreshold || zscore <= -1*ZscoreOutlierThreshold) {
			sumNonOutliers += numsClean[i]
			numberNonOutliers += 1
		}
		sumAll += numsClean[i]
	}

	// Check if there are a lot of outliers warping the balance for expected.
	sumRatio := float64(sumNonOutliers) / float64(sumAll)
	setLengthRatio := float64(numberNonOutliers) / float64(len(nums))

	var mean float64
	if sumRatio > SumRatioThreshold && setLengthRatio < SetRatioThreshold {
		mean = float64(sumNonOutliers) / float64(numberNonOutliers)
	} else {
		mean = float64(sumAll) / float64(len(numsClean))
	}
	expectedThresholdSpread := float64(mean) * ExpectedThreshold

	var expectedLowThreshold float64 = mean - expectedThresholdSpread
	var expectedHighThreshold float64 = mean + expectedThresholdSpread

	// Return balances
	j := 0
	for i := range nums {
		if nums[i] == 0 {
			balances[i] = representationZero
			continue
		}
		if zscores[j] >= ZscoreOutlierThreshold || float64(nums[i]) > expectedHighThreshold {
			balances[i] = representationOver
		} else if zscores[j] <= -1*ZscoreOutlierThreshold || float64(nums[i]) < expectedLowThreshold {
			balances[i] = representationUnder
		} else {
			balances[i] = representationBalanced
		}
		j += 1
	}

	return balances
}
