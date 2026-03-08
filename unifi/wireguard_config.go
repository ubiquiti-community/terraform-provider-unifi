package unifi

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// wireguardConfigParsed contains the parsed fields from a WireGuard .conf file.
type wireguardConfigParsed struct {
	// Interface section
	PrivateKey string
	Address    string
	DNS        []string

	// Peer section
	PublicKey    string
	Endpoint     string
	EndpointIP   string
	EndpointPort int64
	AllowedIPs   string
	PresharedKey string
}

// parseWireGuardConfig parses a standard WireGuard configuration file content
// and extracts the relevant fields.
func parseWireGuardConfig(content string) (*wireguardConfigParsed, error) {
	result := &wireguardConfigParsed{}
	var section string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Section headers
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		// Parse key = value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch section {
		case "interface":
			switch strings.ToLower(key) {
			case "privatekey":
				result.PrivateKey = value
			case "address":
				result.Address = value
			case "dns":
				for _, dns := range strings.Split(value, ",") {
					dns = strings.TrimSpace(dns)
					if dns != "" {
						result.DNS = append(result.DNS, dns)
					}
				}
			}
		case "peer":
			switch strings.ToLower(key) {
			case "publickey":
				result.PublicKey = value
			case "endpoint":
				result.Endpoint = value
				if ip, port, err := parseEndpoint(value); err == nil {
					result.EndpointIP = ip
					result.EndpointPort = port
				}
			case "allowedips":
				result.AllowedIPs = value
			case "presharedkey":
				result.PresharedKey = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading WireGuard configuration: %w", err)
	}

	if result.PublicKey == "" {
		return nil, fmt.Errorf("WireGuard configuration missing [Peer] PublicKey")
	}

	if result.Endpoint == "" {
		return nil, fmt.Errorf("WireGuard configuration missing [Peer] Endpoint")
	}

	return result, nil
}

// parseEndpoint extracts IP and port from a WireGuard endpoint string (e.g., "192.0.2.1:51820").
func parseEndpoint(endpoint string) (string, int64, error) {
	lastColon := strings.LastIndex(endpoint, ":")
	if lastColon < 0 {
		return "", 0, fmt.Errorf("invalid endpoint format: %s", endpoint)
	}

	ip := endpoint[:lastColon]
	portStr := endpoint[lastColon+1:]

	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		ip = ip[1 : len(ip)-1]
	}

	port, err := strconv.ParseInt(portStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in endpoint %s: %w", endpoint, err)
	}

	return ip, port, nil
}

// parseWireGuardBase64Config decodes and parses a base64-encoded WireGuard config.
func parseWireGuardBase64Config(b64Content string) (*wireguardConfigParsed, error) {
	decoded, err := base64.StdEncoding.DecodeString(b64Content)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encoding: %w", err)
	}

	return parseWireGuardConfig(string(decoded))
}
