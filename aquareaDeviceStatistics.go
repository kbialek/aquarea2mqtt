package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Gets most recent data from the statistics page
func (aq *aquarea) getDeviceLogInformation(user aquareaEndUserJSON, shiesuahruefutohkun string) (map[string]string, error) {
	// Build list of all possible values to log
	var valueList strings.Builder
	valueList.WriteString("{\"logItems\":[")
	for i := range aq.logItems {
		valueList.WriteString(strconv.Itoa(i))
		valueList.WriteString(",")
	}
	valueList.WriteString("]}")

	b, err := aq.httpPost(aq.AquareaServiceCloudURL+"/installer/api/data/log", url.Values{
		"var.deviceId":        {user.DeviceID},
		"shiesuahruefutohkun": {shiesuahruefutohkun},
		"var.target":          {"0"},
		"var.startDate":       {fmt.Sprintf("%d000", time.Now().Unix()-aq.logSecOffset)},
		"var.logItems":        {valueList.String()},
	})
	if err != nil {
		return nil, err
	}
	var aquareaLogData aquareaLogDataJSON
	err = json.Unmarshal(b, &aquareaLogData)
	if err != nil {
		return nil, err
	}

	var deviceLog map[int64][]string
	err = json.Unmarshal([]byte(aquareaLogData.LogData), &deviceLog)
	if err != nil {
		return nil, err
	}
	if len(deviceLog) < 1 {
		// no data in log
		return nil, nil
	}

	// we're interested in the most recent snapshot only
	var lastKey int64 = 0
	for k := range deviceLog {
		if lastKey < k {
			lastKey = k
		}
	}

	unitRegexp := regexp.MustCompile(`(.+)\[(.+)\]`)               // extract unit from name
	unitMultiChoiceRegexp := regexp.MustCompile(`(\d+):([^,]+),?`) // extract multi choice values

	stats := make(map[string]string)
	for i, val := range deviceLog[lastKey] {
		split := unitRegexp.FindStringSubmatch(aq.logItems[i])

		topic := fmt.Sprintf("aquarea/%s/log/", user.Gwid) + strings.ReplaceAll(strings.Title(split[1]), " ", "")

		subs := unitMultiChoiceRegexp.FindAllStringSubmatch(split[2], -1)
		if len(subs) > 0 {
			for _, m := range subs {
				if m[1] == val {
					val = m[2]
					break
				}
			}
		} else {
			stats[topic+"/unit"] = split[2] // unit of the value, extracted from name
		}
		stats[topic] = val
	}
	stats[fmt.Sprintf("aquarea/%s/log/Timestamp", user.Gwid)] = strconv.FormatInt(lastKey, 10)
	stats[fmt.Sprintf("aquarea/%s/log/CurrentError", user.Gwid)] = strconv.Itoa(aquareaLogData.ErrorCode)
	return stats, nil
}
