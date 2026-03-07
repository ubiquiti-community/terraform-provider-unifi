package unifi

// UnifiHost represents a UniFi host/console device.
type UnifiHost struct {
	ID                        string `json:"id,omitempty"`
	HardwareID                string `json:"hardwareId,omitempty"`
	Type                      string `json:"type,omitempty"`
	IPAddress                 string `json:"ipAddress,omitempty"`
	Owner                     bool   `json:"owner,omitempty"`
	IsBlocked                 bool   `json:"isBlocked,omitempty"`
	RegistrationTime          string `json:"registrationTime,omitempty"`
	LastConnectionStateChange string `json:"lastConnectionStateChange,omitempty"`
	LatestBackupTime          string `json:"latestBackupTime,omitempty"`
}

// UnifiHostList represents a list of UniFi hosts from the API.
type UnifiHostList struct {
	Data           []UnifiHost `json:"data,omitempty"`
	HTTPStatusCode int         `json:"httpStatusCode,omitempty"`
	TraceID        string      `json:"traceId,omitempty"`
}
