package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgtype"
)

type deviceQueryRequest struct {
	Devices []struct {
		ID         string      `json:"id"`
		CustomData interface{} `json:"custom_data"`
	} `json:"devices"`
}

type deviceResponseQueryYandex struct {
	RequestID string `json:"request_id"`
	Payload   struct {
		Devices []struct {
			ID           string `json:"id"`
			Capabilities []interface {
				// {
				// 	Type  string `json:"type"`
				// 	State struct {
				// 		Instance string      `json:"instance"`
				// 		Value    interface{} `json:"value"`
				// 	} `json:"state"`
			} `json:"capabilities"`
		} `json:"devices"`
	} `json:"payload"`
}

func deviceQuery(c context.Context, requestID string, token string, body []byte) (string, error) {
	ctx := c

	devices := make([]deviceSmartHome, 0)
	rows, err := db.QueryContext(ctx, `SELECT name, password, uri FROM controllers WHERE user_id = (SELECT id FROM users WHERE yandex_token = $1)`, token)
	if err != nil {
		return "", err
	}
	count := 0
	for rows.Next() {
		var name, password, uri pgtype.Varchar

		if err = rows.Scan(&name, &password, &uri); err != nil {
			return "", err
		}

		temp, err := getUserDevicesFromSmartHome(ctx, name.String, password.String, uri.String)
		if err != nil {
			msu.Error(ctx, err)
			return "", err
		}

		devices = append(devices, temp...)
		count++
	}

	if count == 0 {
		var id int
		if err = db.QueryRowContext(ctx, `SELECT id FROM users WHERE yandex_token = $1`, token).Scan(&id); err != nil {
			if err == sql.ErrNoRows {
				return "", errors.New("account_linking_error")
			}
		}
	}

	// devices, err := getUserDevicesFromSmartHome(ctx, "", "", "http://185.180.125.234:9010")
	// if err != nil {
	// 	return "", err
	// }

	var requestedDevices deviceQueryRequest
	if err := json.Unmarshal(body, &requestedDevices); err != nil {
		return "", err
	}

	var response deviceResponseQueryYandex
	response.RequestID = requestID

	for _, requestedDevice := range requestedDevices.Devices {
		for _, device := range devices {
			if device.ID == requestedDevice.ID {
				typeYandexID, err := typeYandex(device.DeviceTypeID)
				if err != nil {
					continue
				}
				if typeYandexID == "devices.types.openable" || typeYandexID == "devices.types.openable.curtain" {
					if device.LineIndex == 1 {
						continue
					}
				}

				response.Payload.Devices = append(response.Payload.Devices, struct {
					ID           string        `json:"id"`
					Capabilities []interface{} `json:"capabilities"`
				}{
					ID:           requestedDevice.ID,
					Capabilities: toYandexQueryCapabilities(device),
				})
				break
			}
		}
	}

	var result []byte

	if result, err = json.Marshal(response); err != nil {
		return "", err
	}

	return string(result), nil
}

func toYandexQueryCapabilities(device deviceSmartHome) []interface{} {
	switch device.DeviceTypeID {
	case 1:
		{
			TurnOn := false
			if device.TurnOn == 1 {
				TurnOn = true
			}
			return []interface{}{struct {
				Type  string `json:"type"`
				State struct {
					Instance string      `json:"instance"`
					Value    interface{} `json:"value"`
				} `json:"state"`
			}{
				Type: "devices.capabilities.on_off",
				State: struct {
					Instance string      "json:\"instance\""
					Value    interface{} "json:\"value\""
				}{
					Instance: "on",
					Value:    TurnOn,
				},
			}}
		}
	case 4, 14:
		{
			TurnOn := false
			if device.TurnOn == 1 {
				TurnOn = true
			}
			return []interface{}{struct {
				Type  string `json:"type"`
				State struct {
					Instance string      `json:"instance"`
					Value    interface{} `json:"value"`
				} `json:"state"`
			}{
				Type: "devices.capabilities.on_off",
				State: struct {
					Instance string      "json:\"instance\""
					Value    interface{} "json:\"value\""
				}{
					Instance: "on",
					Value:    TurnOn,
				},
			}, struct {
				Type  string `json:"type"`
				State struct {
					Instance string      `json:"instance"`
					Value    interface{} `json:"value"`
				} `json:"state"`
			}{
				Type: "devices.capabilities.range",
				State: struct {
					Instance string      "json:\"instance\""
					Value    interface{} "json:\"value\""
				}{
					Instance: "range",
					Value:    float32(device.DimmingValue / 100),
				},
			}}
		}
	case 19:
		return []interface{}{struct {
			Type       string `json:"type"`
			Retrivable bool   `json:"retrivable"`
			Reportable bool   `json:"reportable"`
			Parameters struct {
				Split bool `json:"split"`
			} `json:"parameters"`
		}{
			Type:       "devices.capabilities.on_off",
			Retrivable: true,
			Reportable: true,
		}}
	case 25:
		return []interface{}{
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Reportable bool   `json:"reportable"`
				Parameters struct {
					Split bool `json:"split"`
				} `json:"parameters"`
			}{
				Type:       "devices.capabilities.on_off",
				Retrivable: true,
				Reportable: true,
			},
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Reportable bool   `json:"reportable"`
				Parameters struct {
					Instance     string `json:"instance"`
					Unit         string `json:"unit"`
					RandomAccess bool   `json:"random_access"`
					Range        struct {
						Min       float32 `json:"min"`
						Max       float32 `json:"max"`
						Precision float32 `json:"precision"`
					} `json:"range"`
				} `json:"parameters"`
			}{
				Type:       "devices.capabilities.range",
				Retrivable: true,
				Reportable: true,
				Parameters: struct {
					Instance     string `json:"instance"`
					Unit         string `json:"unit"`
					RandomAccess bool   `json:"random_access"`
					Range        struct {
						Min       float32 `json:"min"`
						Max       float32 `json:"max"`
						Precision float32 `json:"precision"`
					} `json:"range"`
				}{
					Instance:     "temperature",
					Unit:         "unit.temperature.celsius",
					RandomAccess: true,
					Range: struct {
						Min       float32 "json:\"min\""
						Max       float32 "json:\"max\""
						Precision float32 "json:\"precision\""
					}{
						Min:       0,
						Max:       75,
						Precision: 1,
					},
				},
			}}
	}

	return make([]interface{}, 0)
}
