package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActions(t *testing.T) {
	actionJSON := `{
		"id": "{96BFEAAC-57F3-490A-B47A-EBAB901FD8CC}",
		"capabilities": [
		{
			"type": "devices.capabilities.on_off",
			"state": {
			"instance": "on",
			"value": true
			}
		},
		{
			"type": "devices.capabilities.range",
			"state": {
			"instance": "brightness",
			"relative": false,
			"value": 100
			}
		}
		]
	}`
	deviceJSON := `{
		"id": 9,
		"guid": "{96BFEAAC-57F3-490A-B47A-EBAB901FD8CC}",
		"name": "Прожектор Слева",
		"idRooms": 1,
		"roomsName": "Полностью",
		"idDevices": 17,
		"deviceTypes": "",
		"idFloor": 1,
		"floorName": "ВСЕ",
		"line": 61,
		"idLine": 9,
		"indexLine": 0,
		"active": 1,
		"dimming": 1,
		"idStatus": 0,
		"dimmingValue": 0
  	}`

	var action deviceActionRequestYandex
	err := json.Unmarshal([]byte(actionJSON), &action)
	assert.NoError(t, err)

	var device deviceSmartHome
	err = json.Unmarshal([]byte(deviceJSON), &device)
	assert.NoError(t, err)

	devices := []deviceSmartHome{device}

	actions, err := transformActions(devices, action)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(actions))

	assert.Equal(t, 1, actions[0].ChangeDimming)
	assert.Equal(t, 100, actions[0].DimmingValue)
	assert.Equal(t, 1, actions[0].TurnOn)
}
