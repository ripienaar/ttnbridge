package ttn

import (
	"encoding/json"
	"time"
)

// https://www.thethingsnetwork.org/docs/applications/http/

type PushMetadata struct {
	Time       time.Time `json:"time"`
	Frequency  float32   `json:"frequency"`  // frequency the message was sent at
	Modulation string    `json:"modulation"` // LORA or FSK
	DataRate   string    `json:"data_rate"`  // Data rate that was used - if LORA modulation
	BitRate    int       `json:"bit_rate"`   // Bit rate that was used - if FSK modulation
	CodingRate string    `json:"coding_rate"`
	Gateways   []Gateway `json:"gateways"`
	Latitude   int       `json:"latitude"`  // Device lat
	Longitude  int       `json:"longitude"` // Device lon
	Altitude   int       `json:"altitude"`  // Device alt
}

type Gateway struct {
	GatewayID   string    `json:"gtw_id"`
	TimeSeconds int       `json:"timestamp"` // Unix timestamp when the gateway received the message
	Timestamp   time.Time `json:"time"`      // Time when the gateway received the message - left out when gateway does not have synchronized time
	Channel     int       `json:"channel"`   // Radio channel where the gateway received the message
	RSSI        int       `json:"rssi"`      // Signal strength of the received message
	SNR         float32   `json:"snr"`       // Signal to noise ratio of the received message
	RFChain     int       `json:"rf_chain"`  // RF chain where the gateway received the message
	Latitude    int       `json:"latitude"`  // Gateway Lat
	Longitude   int       `json:"longitude"` // Gateway Lon
	Altitude    int       `json:"altitude"`  // Gateway Alt
}

type WebhookIntegrationPush struct {
	AppID          string          `json:"app_id"`
	DeviceID       string          `json:"dev_id"`
	HardwareSerial string          `json:"hardware_serial"` // LoRa Device EUI
	Port           int             `json:"port"`            // LoRaWAN FPort
	Counter        int             `json:"counter"`         // LoRaWAN frame counter
	IsRetry        bool            `json:"is_retry"`
	IsConfirmed    bool            `json:"confirmed"` // Is set to true if this message was a confirmed message
	PayloadRaw     []byte          `json:"payload_raw"`
	PayloadFields  json.RawMessage `json:"payload_fields"` // Output from the payload functions
	Metadata       PushMetadata    `json:"metadata"`
	DownlinkURL    string          `json:"downlink_url"`
}
