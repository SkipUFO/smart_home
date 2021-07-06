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

type deviceResponseYandex struct {
	RequestID string `json:"request_id"`
	Payload   struct {
		UserID  string         `json:"user_id"`
		Devices []deviceYandex `json:"devices"`
	} `json:"payload"`
}

type deviceYandex struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Room         string        `json:"room"`
	Type         string        `json:"type"`
	CustomData   interface{}   `json:"custom_data,omitempty"`
	Capabilities []interface{} `json:"capabilities,omitempty"`
	Properties   []interface{} `json:"properties,omitempty"`
	DeviceInfo   struct {
		Manufacturer string `json:"manufacturer"`
		Model        string `json:"model"`
		HWVersion    string `json:"hw_version"`
		SWVersion    string `json:"sw_version"`
	} `json:"device_info"`
}

type deviceSmartHome struct {
	ID             string `json:"guid"`
	Name           string `json:"name"`
	RoomID         int    `json:"id_rooms"`
	RoomName       string `json:"rooms_name"`
	DeviceTypeID   int    `json:"id_devices"`
	DeviceTypeName string `json:"device_types"`
	FloorID        int    `json:"id_floor"`
	FloorName      string `json:"floor_name"`
	Line           int    `json:"line"`
	LineID         int    `json:"id_line"`
	LineIndex      int    `json:"index_line"`
	Active         int    `json:"active"`
	Dimming        int    `json:"dimming"`
	TurnOn         int    `json:"turn_on"`
	DimmingValue   int    `json:"dimming_value"`
	host           string
	username       string
	password       string
}

func getUserDevices(c context.Context, requestID string, token string) (string, error) {
	ctx := c
	var err error

	response := deviceResponseYandex{}
	response.RequestID = requestID

	// TODO
	response.Payload.UserID = "user"

	// http://192.168.10.17:9010
	// http://188.226.37.223:9010

	// http://185.180.125.234:9010

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

	// temp, err := getUserDevicesFromSmartHome(ctx, "", "", "http://188.226.37.223:9010")
	// if err != nil {
	// 	msu.Error(ctx, err)
	// 	return "", err
	// }
	// devices = append(devices, temp...)

	if response.Payload.Devices, err = toYandexDevices(ctx, devices); err != nil {
		msu.Error(ctx, err)
		return "", err
	}

	var result []byte

	if result, err = json.Marshal(response); err != nil {
		return "", err
	}

	return string(result), nil
}

func toYandexDevices(c context.Context, devices []deviceSmartHome) ([]deviceYandex, error) {
	//ctx := c
	devicesYandex := make([]deviceYandex, 0)

	for _, val := range devices {
		typeYandexID, err := typeYandex(val.DeviceTypeID)
		if typeYandexID == "devices.types.openable" || typeYandexID == "devices.types.openable.curtain" {
			if val.LineIndex == 1 {
				continue
			}
		}
		if err != nil {
			continue
		}

		devicesYandex = append(devicesYandex,
			deviceYandex{
				ID:          val.ID,
				Name:        val.Name,
				Description: "",
				Room:        val.RoomName,
				Type:        typeYandexID,
				//CustomData:   interface{},
				Capabilities: capabilitiesYandex(typeYandexID, val.Dimming),
				//Properties:   []interface{}{propertiesYandex(val.DeviceTypeID)},
				DeviceInfo: struct {
					Manufacturer string "json:\"manufacturer\""
					Model        string "json:\"model\""
					HWVersion    string "json:\"hw_version\""
					SWVersion    string "json:\"sw_version\""
				}{
					Manufacturer: "",
					Model:        "",
					HWVersion:    "",
					SWVersion:    "",
				},
			})
	}

	return devicesYandex, nil
}

func getUserDevicesFromSmartHome(c context.Context, username string, password, host string) ([]deviceSmartHome, error) {
	ctx := c

	request := `getalldevices=` + encode(encryptKey, `{"login":"`+username+`","password":"`+password+`"}`)

	if debug {
		msu.Info(ctx,
			zap.String("request", "controller"),
			zap.Any("uri", host+"?"+request),
			zap.Any("req.object", request))
	}

	// msu.Info(ctx,
	// 	zap.String("request", "controller"),
	// 	zap.Any("uri", host+"?"+request))

	// TODO сделать контекст с дедлайном, чтобы дропать соединения
	req, err := http.NewRequest("GET", host+"?"+request, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []byte
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	devices := make([]deviceSmartHome, 0)

	// msu.Info(ctx, zap.Any("object", strings.TrimSpace(string(body))))
	// msu.Info(ctx, zap.Any("object", decode(encryptKey, strings.TrimSpace(string(body)))))

	if err = json.Unmarshal([]byte(decode(encryptKey, strings.TrimSpace(string(body)))), &devices); err != nil {
		return nil, err
	}

	for index, _ := range devices {
		devices[index].host = host
		devices[index].username = username
		devices[index].password = password
	}

	if debug {
		msu.Info(ctx,
			zap.String("response", "controller"),
			zap.Any("body", string(body)),
			zap.Any("resp.object", devices))
	}

	// msu.Info(ctx,
	// 	zap.String("response", "controller"),
	// 	zap.Any("body", string(body)))

	return devices, nil
}

func capabilitiesYandex(yandexTypeID string, dimming int) []interface{} {
	switch yandexTypeID {
	case "devices.types.light":
		if dimming == 0 {
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
				Parameters: struct {
					Split bool `json:"split"`
				}{Split: false},
			}}
		} else {
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
				Parameters: struct {
					Split bool `json:"split"`
				}{Split: false},
			}, struct {
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
					Instance:     "brightness",
					Unit:         "unit.percent",
					RandomAccess: true,
					Range: struct {
						Min       float32 "json:\"min\""
						Max       float32 "json:\"max\""
						Precision float32 "json:\"precision\""
					}{
						Min:       0,
						Max:       100,
						Precision: 1,
					},
				},
			}}
		}
	case "devices.types.socket":
		return []interface{}{struct {
			Type       string `json:"type"`
			Retrivable bool   `json:"retrivable"`
		}{
			Type:       "devices.capabilities.on_off",
			Retrivable: true,
		}}
	case "devices.types.thermostat.ac":
		return []interface{}{
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Parameters struct {
					Instance     string `json:"instance"`
					RandomAccess bool   `json:"random_access"`
					Range        struct {
						Max       int `json:"max"`
						Min       int `json:"min"`
						Precision int `json:"precision"`
					} `json:"range"`
					Unit string `json:"unit"`
				} `json:"parameters"`
			}{
				Type:       "devices.capabilities.range",
				Retrivable: true,
				Parameters: struct {
					Instance     string "json:\"instance\""
					RandomAccess bool   "json:\"random_access\""
					Range        struct {
						Max       int "json:\"max\""
						Min       int "json:\"min\""
						Precision int "json:\"precision\""
					} "json:\"range\""
					Unit string "json:\"unit\""
				}{
					Instance:     "temperature",
					RandomAccess: true,
					Range: struct {
						Max       int "json:\"max\""
						Min       int "json:\"min\""
						Precision int "json:\"precision\""
					}{
						Max:       33,
						Min:       18,
						Precision: 1,
					},
					Unit: "unit.temperature.celsius",
				},
			},
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Parameters struct {
					Instance string `json:"instance"`
					Modes    []struct {
						Value string `json:"value"`
					} `json:"modes"`
				} `json:"parameters"`
			}{
				Type:       "devices.capabilities.mode",
				Retrivable: true,
				Parameters: struct {
					Instance string "json:\"instance\""
					Modes    []struct {
						Value string "json:\"value\""
					} "json:\"modes\""
				}{
					Instance: "fan_speed",
					Modes: []struct {
						Value string "json:\"value\""
					}{
						{Value: "high"},
						{Value: "medium"},
						{Value: "low"},
						{Value: "auto"},
					},
				},
			},
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Parameters struct {
					Instance string `json:"instance"`
					Modes    []struct {
						Value string `json:"value"`
					} `json:"modes"`
				}
			}{
				Type:       "devices.capabilities.mode",
				Retrivable: true,
				Parameters: struct {
					Instance string "json:\"instance\""
					Modes    []struct {
						Value string "json:\"value\""
					} "json:\"modes\""
				}{
					Instance: "thermostat",
					Modes: []struct {
						Value string "json:\"value\""
					}{
						{Value: "fan_only"},
						{Value: "heat"},
						{Value: "cool"},
						{Value: "dry"},
						{Value: "auto"},
					},
				},
			},
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
			}{
				Type:       "devices.capabilities.on_off",
				Retrivable: true,
			},
		}
	case "devices.types.openable.curtain":
		return []interface{}{
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
				Parameters struct {
					Split bool `json:"split"`
				} `json:"parameters"`
			}{
				Type:       "devices.capabilities.on_off",
				Retrivable: false,
				Parameters: struct {
					Split bool "json:\"split\""
				}{
					Split: true,
				},
			},
		}
	case "devices.types.openable":
		return []interface{}{
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
			}{
				Type:       "devices.capabilities.on_off",
				Retrivable: true,
			},
		}
	case "devices.types.other":
		return []interface{}{
			struct {
				Type       string `json:"type"`
				Retrivable bool   `json:"retrivable"`
			}{
				Type:       "devices.capabilities.on_off",
				Retrivable: true,
			},
		}
	}

	return make([]interface{}, 0)
}

func propertiesYandex(smartHomeTypeID int) []interface{} {
	// switch smartHomeTypeID {
	// case 14:
	// }

	return make([]interface{}, 0)
}

func typeYandex(smartHomeTypeID int) (string, error) {
	switch smartHomeTypeID {
	case 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18:
		return "devices.types.light", nil
	/*case 19, 25, 36, 42, 43, 44, 46, 48, 49, 50, 51, 52, 53, 54, 55, 56, 58, 59:
		return "devices.types.socket", nil
	case 28:
		return "devices.types.thermostat.ac", nil*/
	case 20, 21, 22, 23, 34, 35:
		return "devices.types.openable.curtain", nil
		/*case 24, 29, 30, 31, 32, 41:
			return "devices.types.openable", nil
		default:
			return "devices.types.other", nil
		*/
	}

	return "", errors.New("type not found")
}
