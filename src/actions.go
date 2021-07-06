package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jackc/pgtype"
	"go.uber.org/zap"
)

type actionRequestYandex struct {
	Payload struct {
		Devices []deviceActionRequestYandex `json:"devices"`
	} `json:"payload"`
}

type deviceActionRequestYandex struct {
	ID           string `json:"id"`
	CustomData   interface{}
	Capabilities []struct {
		Type  string `json:"type"`
		State struct {
			Instance string      `json:"instance"`
			Value    interface{} `json:"value"`
			Relative bool        `json:"relative,omitempty"`
		} `json:"state"`
	} `json:"capabilities"`
}

type actionResponseYandex struct {
	RequestID string `json:"request_id"`
	Payload   struct {
		Devices []deviceActionResponseYandex `json:"devices"`
	} `json:"payload"`
}

type deviceActionResponseYandex struct {
	ID           string `json:"id"`
	Capabilities []struct {
		Type  string `json:"type"`
		State struct {
			Instance     string `json:"instance"`
			ActionResult struct {
				Status       string `json:"status,omitempty"`
				ErrorCode    string `json:"error_code,omitempty"`
				ErrorMessage string `json:"error_message,omitempty"`
			} `json:"action_result,omitempty"`
		} `json:"state"`
	} `json:"capabilities"`
	ActionResult struct {
		Status       string `json:"status,omitempty"`
		ErrorCode    string `json:"error_code,omitempty"`
		ErrorMessage string `json:"error_message,omitempty"`
	} `json:"action_result,omitempty"`
}

type deviceActionSmartHome struct {
	SetCommand    string `json:"setcommand"`
	Login         string `json:"login"`
	Password      string `json:"password"`
	ID            string `json:"id"`
	FloorID       int    `json:"id_floor"`
	RoomID        int    `json:"id_room"`
	LineID        int    `json:"id_line"`
	Line          int    `json:"line"`
	LineIndex     int    `json:"index_line"`
	TurnOn        int    `json:"turn_on"`
	ChangeDimming int    `json:"change_dimming"`
	Dimming       int    `json:"dimming"`
	DimmingValue  int    `json:"dimming_value"`
	ColorDraw     string `json:"color_draw"`
	ColorDrawOff  string `json:"color_draw_off"`
	SetPassword   string `json:"set_password"`
	ColorText     string `json:"color_text"`
}

func deviceAction(c context.Context, requestID string, token string, body []byte) (string, error) {
	ctx := c
	var request actionRequestYandex
	var response actionResponseYandex

	var err error
	if err := json.Unmarshal(body, &request); err != nil {
		return "", err
	}

	response.RequestID = requestID

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

	for _, val := range request.Payload.Devices {
		ds := make([]deviceSmartHome, 0)
		for _, device := range devices {
			if val.ID == device.ID {
				ds = append(ds, device)
			}
		}

		if len(ds) != 0 {
			if err := actionToSmartHome(ctx, ds, ds[0].host, ds[0].username, ds[0].password, val); err != nil {
				response.Payload.Devices = append(response.Payload.Devices,
					deviceActionResponseYandex{
						ID: val.ID,
						ActionResult: struct {
							Status       string "json:\"status,omitempty\""
							ErrorCode    string "json:\"error_code,omitempty\""
							ErrorMessage string "json:\"error_message,omitempty\""
						}{
							Status: "ERROR",
						},
					},
				)
			}

			var caps []struct {
				Type  string "json:\"type\""
				State struct {
					Instance     string "json:\"instance\""
					ActionResult struct {
						Status       string "json:\"status,omitempty\""
						ErrorCode    string "json:\"error_code,omitempty\""
						ErrorMessage string "json:\"error_message,omitempty\""
					} "json:\"action_result,omitempty\""
				} "json:\"state\""
			}
			for _, cap := range val.Capabilities {
				caps = append(caps, struct {
					Type  string "json:\"type\""
					State struct {
						Instance     string "json:\"instance\""
						ActionResult struct {
							Status       string "json:\"status,omitempty\""
							ErrorCode    string "json:\"error_code,omitempty\""
							ErrorMessage string "json:\"error_message,omitempty\""
						} "json:\"action_result,omitempty\""
					} "json:\"state\""
				}{
					Type: cap.Type,
					State: struct {
						Instance     string "json:\"instance\""
						ActionResult struct {
							Status       string "json:\"status,omitempty\""
							ErrorCode    string "json:\"error_code,omitempty\""
							ErrorMessage string "json:\"error_message,omitempty\""
						} "json:\"action_result,omitempty\""
					}{
						Instance: cap.State.Instance,
						ActionResult: struct {
							Status       string "json:\"status,omitempty\""
							ErrorCode    string "json:\"error_code,omitempty\""
							ErrorMessage string "json:\"error_message,omitempty\""
						}{
							Status: "DONE",
						},
					},
				})
			}
			response.Payload.Devices = append(response.Payload.Devices,
				deviceActionResponseYandex{
					ID:           val.ID,
					Capabilities: caps,
				},
			)
		}
	}

	var result []byte

	if result, err = json.Marshal(response); err != nil {
		return "", err
	}

	return string(result), nil
}

func actionToSmartHome(c context.Context, devices []deviceSmartHome, host string, username string, password string, action deviceActionRequestYandex) error {
	ctx := c

	var TurnOn int
	var DimmingValue float64
	var actions []deviceActionSmartHome

	if len(devices) == 1 {
		device := devices[0]
		TurnOn = devices[0].TurnOn
		if action.Capabilities[0].Type == "devices.capabilities.on_off" {
			if action.Capabilities[0].State.Instance == "on" {
				if action.Capabilities[0].State.Value.(bool) {
					TurnOn = 1
				} else {
					TurnOn = 0
				}
			}
		}
		DimmingValue = float64(devices[0].DimmingValue)
		if action.Capabilities[0].Type == "devices.capabilities.range" {
			if action.Capabilities[0].State.Instance == "brightness" {
				if action.Capabilities[0].State.Relative {
					DimmingValue += action.Capabilities[0].State.Value.(float64)
				} else {
					DimmingValue = action.Capabilities[0].State.Value.(float64)
				}
			}
		}

		actions = append(actions, deviceActionSmartHome{
			SetCommand:    "true",
			Login:         username,
			Password:      password,
			ID:            device.ID,
			FloorID:       device.FloorID,
			RoomID:        device.RoomID,
			LineID:        device.LineID,
			Line:          device.Line,
			LineIndex:     device.LineIndex,
			TurnOn:        TurnOn,
			ChangeDimming: 0,
			Dimming:       device.Dimming,
			DimmingValue:  int(DimmingValue),
			ColorDraw:     "0xff010000",
			ColorDrawOff:  "0xff000000",
			SetPassword:   "",
			ColorText:     "",
		})

	} else {
		if action.Capabilities[0].Type == "devices.capabilities.on_off" {
			if action.Capabilities[0].State.Instance == "on" {
				if action.Capabilities[0].State.Value.(bool) {
					actions = append(actions, deviceActionSmartHome{
						SetCommand:    "true",
						Login:         username,
						Password:      password,
						ID:            devices[0].ID,
						FloorID:       devices[0].FloorID,
						RoomID:        devices[0].RoomID,
						LineID:        devices[0].LineID,
						Line:          devices[0].Line,
						LineIndex:     devices[0].LineIndex,
						TurnOn:        0,
						ChangeDimming: 0,
						Dimming:       devices[0].Dimming,
						DimmingValue:  int(DimmingValue),
						ColorDraw:     "0xff010000",
						ColorDrawOff:  "0xff000000",
						SetPassword:   "",
						ColorText:     "",
					})

					actions = append(actions, deviceActionSmartHome{
						SetCommand:    "true",
						Login:         username,
						Password:      password,
						ID:            devices[1].ID,
						FloorID:       devices[1].FloorID,
						RoomID:        devices[1].RoomID,
						LineID:        devices[1].LineID,
						Line:          devices[1].Line,
						LineIndex:     devices[1].LineIndex,
						TurnOn:        1,
						ChangeDimming: 0,
						Dimming:       devices[1].Dimming,
						DimmingValue:  int(DimmingValue),
						ColorDraw:     "0xff010000",
						ColorDrawOff:  "0xff000000",
						SetPassword:   "",
						ColorText:     "",
					})
				} else {
					actions = append(actions, deviceActionSmartHome{
						SetCommand:    "true",
						Login:         username,
						Password:      password,
						ID:            devices[0].ID,
						FloorID:       devices[0].FloorID,
						RoomID:        devices[0].RoomID,
						LineID:        devices[0].LineID,
						Line:          devices[0].Line,
						LineIndex:     devices[0].LineIndex,
						TurnOn:        1,
						ChangeDimming: 0,
						Dimming:       devices[0].Dimming,
						DimmingValue:  int(DimmingValue),
						ColorDraw:     "0xff010000",
						ColorDrawOff:  "0xff000000",
						SetPassword:   "",
						ColorText:     "",
					})

					actions = append(actions, deviceActionSmartHome{
						SetCommand:    "true",
						Login:         username,
						Password:      password,
						ID:            devices[1].ID,
						FloorID:       devices[1].FloorID,
						RoomID:        devices[1].RoomID,
						LineID:        devices[1].LineID,
						Line:          devices[1].Line,
						LineIndex:     devices[1].LineIndex,
						TurnOn:        0,
						ChangeDimming: 0,
						Dimming:       devices[1].Dimming,
						DimmingValue:  int(DimmingValue),
						ColorDraw:     "0xff010000",
						ColorDrawOff:  "0xff000000",
						SetPassword:   "",
						ColorText:     "",
					})
				}
			}
		}
	}

	for _, act := range actions {

		var b []byte
		var err error
		if b, err = json.Marshal(act); err != nil {
			return err
		}

		if debug {
			msu.Info(ctx,
				zap.String("request", "controller"),
				zap.Any("uri", host+"?"+encode(encryptKey, string(b))),
				zap.Any("req.object", act))
		}

		msu.Info(ctx,
			zap.String("request", "controller"),
			zap.Any("uri", host+"?"+encode(encryptKey, string(b))))

		req, err := http.NewRequest("GET", host+"?"+encode(encryptKey, string(b)), nil)
		if err != nil {
			return err
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var body []byte
		if body, err = ioutil.ReadAll(resp.Body); err != nil {
			return err
		}
		defer resp.Body.Close()

		if debug {
			msu.Info(ctx,
				zap.String("response", "controller"),
				zap.Any("uri", host+"?"+encode(encryptKey, string(b))),
				zap.Any("body", string(body)),
				zap.Any("resp.object", decode(encryptKey, strings.TrimSpace(string(body)))))
		}

		msu.Info(ctx,
			zap.String("response", "controller"),
			zap.Any("uri", host+"?"+encode(encryptKey, string(b))),
			zap.Any("body", string(body)))
	}

	return nil
}
