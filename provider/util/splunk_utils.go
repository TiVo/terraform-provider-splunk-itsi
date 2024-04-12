package util

import "fmt"

var StatusInfoMap = map[string]int{
	"new":         1,
	"in progress": 2,
	"pending":     3,
	"resolved":    4,
	"closed":      5,
}

func GetStatusInfoByValue(value int) (string, error) {
	for label, v := range StatusInfoMap {
		if v == value {
			return label, nil
		}
	}
	return "", fmt.Errorf("unknown status value: %d", value)
}

type SeverityInfo struct {
	// Severity label assigned for the threshold level, including info, warning, critical, etc.
	SeverityLabel string
	// Severity color assigned for the threshold level
	SeverityColor string
	// Severity light color assigned for the threshold level
	SeverityColorLight string
	// Value for threshold level.
	SeverityValue int
}

var SeverityMap = map[string]SeverityInfo{
	"critical": {
		SeverityLabel:      "critical",
		SeverityColor:      "#B50101",
		SeverityColorLight: "#E5A6A6",
		SeverityValue:      6,
	},
	"high": {
		SeverityLabel:      "high",
		SeverityColor:      "#F26A35",
		SeverityColorLight: "#FBCBB9",
		SeverityValue:      5,
	},
	"medium": {
		SeverityLabel:      "medium",
		SeverityColor:      "#FCB64E",
		SeverityColorLight: "#FEE6C1",
		SeverityValue:      4,
	},
	"low": {
		SeverityLabel:      "low",
		SeverityColor:      "#FFE98C",
		SeverityColorLight: "#FFF4C5",
		SeverityValue:      3,
	},
	"normal": {
		SeverityLabel:      "normal",
		SeverityColor:      "#99D18B",
		SeverityColorLight: "#DCEFD7",
		SeverityValue:      2,
	},
	"info": {
		SeverityLabel:      "info",
		SeverityColor:      "#AED3E5",
		SeverityColorLight: "#E3F0F6",
		SeverityValue:      1,
	},
}

func GetSupportedStatuses() (labels []string) {
	for k := range StatusInfoMap {
		labels = append(labels, k)
	}
	return
}

func GetSupportedSeverities() (labels []string) {
	for k := range SeverityMap {
		labels = append(labels, k)
	}
	return
}

func GetSeverityInfoByValue(severityValue int) (*SeverityInfo, error) {
	for _, info := range SeverityMap {
		if info.SeverityValue == severityValue {
			return &info, nil
		}
	}
	return nil, fmt.Errorf("unknown severity value: %d", severityValue)
}
