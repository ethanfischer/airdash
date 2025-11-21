package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type AirGradientConfig struct {
	Token      string `yaml:"token"`
	LocationID int    `yaml:"locationId"`
}

type MonitorConfig struct {
	PushoverUserKey      string  `yaml:"pushoverUserKey"`
	PushoverAPIToken     string  `yaml:"pushoverApiToken"`
	PM25ThresholdWarning float64 `yaml:"pm25ThresholdWarning"`
	PM25ThresholdUrgent  float64 `yaml:"pm25ThresholdUrgent"`
	PM25ThresholdEmergency float64 `yaml:"pm25ThresholdEmergency"`
	CO2Threshold         float64 `yaml:"co2Threshold"`
	TVOCThreshold        float64 `yaml:"tvocThreshold"`
	CheckInterval        int     `yaml:"checkInterval"` // minutes
}

type AirGradientMeasures struct {
	LocationID   int       `yaml:"locationId"`
	LocationName string    `yaml:"locationName"`
	Pm02         float64   `yaml:"pm02"`
	Atmp         float64   `yaml:"atmp"`
	Rhum         float64   `yaml:"rhum"`
	Rco2         float64   `yaml:"rco2"`
	Tvoc         float64   `yaml:"tvoc"`
	Timestamp    time.Time `yaml:"timestamp"`
}

var pm25AlertLevel = "" // "", "warning", "urgent", "emergency"
var co2AlertActive = false
var tvocAlertActive = false

func main() {
	log.Println("AirDash Monitor starting...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// Load AirGradient config
	agConfig, err := loadAirGradientConfig(homeDir + "/.airdash/config.yaml")
	if err != nil {
		log.Fatal("Failed to load AirGradient config:", err)
	}

	// Load monitor config
	monConfig, err := loadMonitorConfig(homeDir + "/.airdash/monitor-config.yaml")
	if err != nil {
		log.Fatal("Failed to load monitor config:", err)
	}

	log.Printf("Monitoring PM2.5 levels - Warning: %.1f, Urgent: %.1f, Emergency: %.1f μg/m³",
		monConfig.PM25ThresholdWarning, monConfig.PM25ThresholdUrgent, monConfig.PM25ThresholdEmergency)
	log.Printf("Monitoring CO2 levels. Threshold: %.0f ppm", monConfig.CO2Threshold)
	log.Printf("Monitoring TVOC levels. Threshold: %.0f ppb", monConfig.TVOCThreshold)
	log.Printf("Checking every %d minutes", monConfig.CheckInterval)

	// Run immediately, then on interval
	checkAirQuality(agConfig, monConfig)

	ticker := time.NewTicker(time.Duration(monConfig.CheckInterval) * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		checkAirQuality(agConfig, monConfig)
	}
}

func checkAirQuality(agConfig *AirGradientConfig, monConfig *MonitorConfig) {
	measures, err := fetchAirGradientData(agConfig)
	if err != nil {
		log.Printf("Error fetching air quality data: %v", err)
		return
	}

	log.Printf("Current PM2.5: %.1f μg/m³, CO2: %.0f ppm, TVOC: %.0f ppb", measures.Pm02, measures.Rco2, measures.Tvoc)

	// Check PM2.5 with tiered alerts
	var newPM25Level string
	if measures.Pm02 >= monConfig.PM25ThresholdEmergency {
		newPM25Level = "emergency"
	} else if measures.Pm02 >= monConfig.PM25ThresholdUrgent {
		newPM25Level = "urgent"
	} else if measures.Pm02 >= monConfig.PM25ThresholdWarning {
		newPM25Level = "warning"
	} else {
		newPM25Level = ""
	}

	// Only send alert if level changed
	if newPM25Level != pm25AlertLevel {
		if newPM25Level == "emergency" {
			sendPushoverNotification(
				monConfig,
				"🚨 PM2.5 EMERGENCY",
				fmt.Sprintf("PM2.5 has reached EMERGENCY levels at %.1f μg/m³\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
					measures.Pm02, measures.Pm02, measures.Rco2, measures.Tvoc),
				2, // emergency priority
			)
			log.Printf("🚨 EMERGENCY: PM2.5 at critical levels!")
		} else if newPM25Level == "urgent" {
			sendPushoverNotification(
				monConfig,
				"⚠️ PM2.5 URGENT",
				fmt.Sprintf("PM2.5 is at URGENT levels at %.1f μg/m³\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
					measures.Pm02, measures.Pm02, measures.Rco2, measures.Tvoc),
				1, // high priority
			)
			log.Printf("⚠️  URGENT: PM2.5 at elevated levels!")
		} else if newPM25Level == "warning" {
			sendPushoverNotification(
				monConfig,
				"⚡ PM2.5 Warning",
				fmt.Sprintf("PM2.5 has exceeded warning threshold at %.1f μg/m³\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
					measures.Pm02, measures.Pm02, measures.Rco2, measures.Tvoc),
				1, // high priority
			)
			log.Printf("⚡ WARNING: PM2.5 above warning threshold!")
		} else if pm25AlertLevel != "" {
			// Was in alert, now back to normal
			sendPushoverNotification(
				monConfig,
				"✓ PM2.5 Normal",
				fmt.Sprintf("PM2.5 has returned to safe levels at %.1f μg/m³\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
					measures.Pm02, measures.Pm02, measures.Rco2, measures.Tvoc),
				0, // normal priority
			)
			log.Printf("✓ All clear: PM2.5 back to normal")
		}
		pm25AlertLevel = newPM25Level
	}

	// Check CO2
	if measures.Rco2 > monConfig.CO2Threshold && !co2AlertActive {
		// CO2 exceeded threshold - send alert
		co2AlertActive = true
		sendPushoverNotification(
			monConfig,
			"CO2 Alert",
			fmt.Sprintf("CO2 is elevated at %.0f ppm (threshold: %.0f)\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
				measures.Rco2, monConfig.CO2Threshold, measures.Pm02, measures.Rco2, measures.Tvoc),
			1, // high priority
		)
		log.Printf("⚠️  ALERT: CO2 exceeded threshold!")
	} else if measures.Rco2 <= monConfig.CO2Threshold && co2AlertActive {
		// CO2 back to normal - send all clear
		co2AlertActive = false
		sendPushoverNotification(
			monConfig,
			"CO2 Normal",
			fmt.Sprintf("CO2 has returned to safe levels at %.0f ppm\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
				measures.Rco2, measures.Pm02, measures.Rco2, measures.Tvoc),
			0, // normal priority
		)
		log.Printf("✓ All clear: CO2 back to normal")
	}

	// Check TVOC
	if measures.Tvoc > monConfig.TVOCThreshold && !tvocAlertActive {
		// TVOC exceeded threshold - send alert
		tvocAlertActive = true
		sendPushoverNotification(
			monConfig,
			"TVOC Alert",
			fmt.Sprintf("TVOC is elevated at %.0f ppb (threshold: %.0f)\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
				measures.Tvoc, monConfig.TVOCThreshold, measures.Pm02, measures.Rco2, measures.Tvoc),
			1, // high priority
		)
		log.Printf("⚠️  ALERT: TVOC exceeded threshold!")
	} else if measures.Tvoc <= monConfig.TVOCThreshold && tvocAlertActive {
		// TVOC back to normal - send all clear
		tvocAlertActive = false
		sendPushoverNotification(
			monConfig,
			"TVOC Normal",
			fmt.Sprintf("TVOC has returned to safe levels at %.0f ppb\n\nCurrent readings:\n• PM2.5: %.1f μg/m³\n• CO2: %.0f ppm\n• TVOC: %.0f ppb",
				measures.Tvoc, measures.Pm02, measures.Rco2, measures.Tvoc),
			0, // normal priority
		)
		log.Printf("✓ All clear: TVOC back to normal")
	}
}

func fetchAirGradientData(config *AirGradientConfig) (*AirGradientMeasures, error) {
	apiURL := fmt.Sprintf("https://api.airgradient.com/public/api/v1/locations/%d/measures/current", config.LocationID)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("token", config.Token)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// API can return single object or array
	var measures AirGradientMeasures
	if err := json.Unmarshal(body, &measures); err != nil {
		// Try array format
		var measuresArray []AirGradientMeasures
		if err := json.Unmarshal(body, &measuresArray); err != nil {
			return nil, err
		}
		if len(measuresArray) == 0 {
			return nil, fmt.Errorf("no measurements returned")
		}
		measures = measuresArray[0]
	}

	return &measures, nil
}

func sendPushoverNotification(config *MonitorConfig, title, message string, priority int) {
	data := url.Values{}
	data.Set("token", config.PushoverAPIToken)
	data.Set("user", config.PushoverUserKey)
	data.Set("title", title)
	data.Set("message", message)
	data.Set("priority", fmt.Sprintf("%d", priority))

	// Emergency priority (2) requires retry and expire parameters
	// Notification will repeat every 60 seconds for up to 1 hour until acknowledged
	if priority == 2 {
		data.Set("retry", "60")    // Retry every 60 seconds
		data.Set("expire", "3600") // Stop retrying after 1 hour
		data.Set("sound", "siren") // Use siren sound for emergency
	}

	resp, err := http.PostForm("https://api.pushover.net/1/messages.json", data)
	if err != nil {
		log.Printf("Failed to send notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Pushover API error (status %d): %s", resp.StatusCode, body)
		return
	}

	log.Println("✓ Push notification sent successfully")
}

func loadAirGradientConfig(path string) (*AirGradientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config AirGradientConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func loadMonitorConfig(path string) (*MonitorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config MonitorConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set defaults
	if config.PM25ThresholdWarning == 0 {
		config.PM25ThresholdWarning = 12
	}
	if config.PM25ThresholdUrgent == 0 {
		config.PM25ThresholdUrgent = 35
	}
	if config.PM25ThresholdEmergency == 0 {
		config.PM25ThresholdEmergency = 55
	}
	if config.CO2Threshold == 0 {
		config.CO2Threshold = 750
	}
	if config.TVOCThreshold == 0 {
		config.TVOCThreshold = 250
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = 5
	}

	return &config, nil
}
