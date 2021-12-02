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
	defer rows.Close()
	count := 0
	for rows.Next() {
		var name, password, uri pgtype.Varchar

		if err = rows.Scan(&name, &password, &uri); err != nil {
			return "", err
		}

		temp, err := getUserDevicesFromSmartHome(ctx, name.String, password.String, uri.String)
		if err != nil {
			msu.Error(ctx, err)
			// return "", err
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
			if device.Guid == requestedDevice.ID {
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
					Capabilities: toYandexQueryCapabilities(typeYandexID, device),
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

func toYandexQueryCapabilities(yandexType string, device deviceSmartHome) []interface{} {
	switch yandexType {
	case "devices.types.light":
		{
			if device.Dimming == 0 {
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
			} else {
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
		}
	case "devices.types.socket":
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
	case "devices.types.openable.curtain":
		{
			TurnOn := false
			if device.TurnOn == 1 && device.LineIndex%2 == 0 {
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
	case "devices.types.openable":
		{
			TurnOn := false
			if device.TurnOn == 1 && device.LineIndex%2 == 0 {
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
	case "devices.types.other":
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
	}

	return make([]interface{}, 0)
}
