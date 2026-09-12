package devices

import "time"

type Device struct {
	ID              string    `json:"id"`
	MerchantID      string    `json:"merchant_id"`
	Name            string    `json:"name"`
	Model           string    `json:"model"`
	AndroidVersion  string    `json:"android_version"`
	AppVersion      string    `json:"app_version"`
	Status          string    `json:"status"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at,omitempty"`
}

type PairRequest struct {
	PairingToken   string `json:"pairing_token"`
	DeviceName     string `json:"device_name"`
	Model          string `json:"model"`
	AndroidVersion string `json:"android_version"`
	AppVersion     string `json:"app_version"`
}

type PairResponse struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
	Status       string `json:"status"`
}

type SMSItem struct {
	ClientID    string    `json:"client_id"`
	Sender      string    `json:"sender"`
	Body        string    `json:"body"`
	SMSAt       time.Time `json:"sms_time"`
	SimSlot     *int32    `json:"sim_slot,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	AccountType string    `json:"account_type,omitempty"`
}

type SMSRequest struct {
	Messages []SMSItem `json:"messages"`
	SMS      *SMSItem  `json:"sms,omitempty"`
}

type SMSAck struct {
	ClientID string `json:"client_id"`
	SMSID    string `json:"sms_id,omitempty"`
	Status   string `json:"status"`
}
type HeartbeatRequest struct {
	BatteryLevel int32  `json:"battery_level"`
	AppVersion   string `json:"app_version"`
	QueueDepth   int32  `json:"queue_depth"`
}
type HeartbeatResponse struct {
	Status string `json:"status"`
}
type ConfigResponse struct {
	LatestAppVersion string              `json:"latest_app_version,omitempty"`
	APKURL           string              `json:"apk_url,omitempty"`
	MandatoryUpdate  bool                `json:"mandatory_update"`
	ParserVersion    string              `json:"parser_version,omitempty"`
	SenderIDs        map[string][]string `json:"sender_ids"`
}
