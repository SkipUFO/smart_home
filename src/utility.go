package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func arrayMapStringToArrayInterface(input []map[string]string) []interface{} {
	result := []interface{}{}

	for _, val := range input {
		result = append(result, val)
	}

	return result
}

func arrayMapIntToArrayInterface(input []map[string]int) []interface{} {
	result := []interface{}{}

	for _, val := range input {
		result = append(result, val)
	}

	return result
}

type dateTimeArray []time.Time

func (dates *dateTimeArray) MarshalCSV() (string, error) {
	var temp []string
	for _, val := range *dates {
		temp = append(temp, val.Format(time.RFC3339))
	}
	return strings.Join(temp, ","), nil
}

type stringArray []string

func (strs *stringArray) MarshalCSV() (string, error) {
	var temp []string
	for _, val := range *strs {
		temp = append(temp, val)
	}
	return strings.Join(temp, ","), nil
}

type floatArray []float32

func (floats *floatArray) MarshalCSV() (string, error) {
	var temp []string
	for _, val := range *floats {
		f := fmt.Sprintf("%f", val)
		temp = append(temp, f)
	}
	return strings.Join(temp, ","), nil
}

type intArray []int32

func (ints *intArray) MarshalCSV() (string, error) {
	var temp []string
	for _, val := range *ints {
		temp = append(temp, strconv.Itoa(int(val)))
	}
	return strings.Join(temp, ","), nil
}

func (strs *stringArray) Join(sep string) string {
	var temp []string
	for _, val := range *strs {
		if strings.Contains(val, " ") {
			temp = append(temp, "\""+val+"\"")
		} else {
			temp = append(temp, val)
		}
	}
	return strings.Join(temp, sep)
}

func join(input []string, sep string) string {
	var temp []string
	for _, val := range input {
		if strings.Contains(val, " ") {
			temp = append(temp, "\""+val+"\"")
		} else {
			temp = append(temp, val)
		}
	}
	return strings.Join(temp, sep)
}

func getString(val *string) string {
	if val != nil {
		return *val
	}

	return ""
}

func toInt(vars map[string]string, name string) (int, error) {
	temp, ok := vars[name]
	if !ok {
		return 0, errors.New("Param '" + name + "' not found")
	}

	result, err := strconv.Atoi(temp)
	if err != nil {
		return 0, err
	}

	return result, nil
}
