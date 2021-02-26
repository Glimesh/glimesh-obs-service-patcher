package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
)

type obsServicesFile struct {
	FormatVersion int          `json:"format_version"`
	Services      []obsService `json:"services"`
}

type obsService struct {
	Name    string `json:"name"`
	Servers []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"servers"`
	Recommended struct {
		Keyint          int    `json:"keyint"`
		Output          string `json:"output"`
		MaxAudioBitrate int    `json:"max audio bitrate"`
		MaxVideoBitrate int    `json:"max video bitrate"`
		Bframes         int    `json:"bframes"`
		X264Opts        string `json:"x264opts"`
	} `json:"recommended"`
}

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(string(bytes))
}

func panicAndPause(v ...interface{}) {
	log.Print(v...)
	fmt.Println("Glimesh OBS Service Patcher Failed!\nPress the Enter key or close this window.")
	fmt.Scanln()
	os.Exit(1)
}

func main() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))

	glimeshServiceEntry := getGlimeshServiceContents("https://glimesh-static-assets.nyc3.digitaloceanspaces.com/obs-glimesh-service.json")

	var glimeshService obsService
	err := json.Unmarshal([]byte(glimeshServiceEntry), &glimeshService)
	if err != nil {
		panicAndPause("Problem unmarshalling Glimesh JSON entry.")
	}

	log.Println()

	servicesFiles := findObsDirectories()

	log.Println()

	for _, serviceFile := range servicesFiles {
		patchFile(serviceFile, glimeshService)
	}

	log.Println()

	fmt.Println("Glimesh OBS Service Patcher Completed!\nPress the Enter key or close this window.")
	fmt.Scanln()
}

func getGlimeshServiceContents(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panicAndPause(err)
	}

	if resp.StatusCode != http.StatusOK {
		panicAndPause("Got an error code from the CDN")
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panicAndPause(err)
	}

	log.Printf("💽 Downloaded Glimesh Service Definition from %s\n", url)

	return data
}

func patchFile(filePath string, newService obsService) {
	servicesFile, err := os.Open(filePath)
	if err != nil {
		panicAndPause(err)
	}

	var services obsServicesFile
	byteValue, err := ioutil.ReadAll(servicesFile)
	json.Unmarshal(byteValue, &services)

	foundGlimesh := false
	for i := 0; i < len(services.Services); i++ {
		if services.Services[i].Name == "Glimesh" {
			foundGlimesh = true
		}
	}

	servicesFile.Close()

	if foundGlimesh == false {
		services.Services = append(services.Services, newService)

		whatever, err := json.MarshalIndent(services, "", "    ")
		err = os.WriteFile(filePath, whatever, 0644)
		if err != nil {
			log.Printf("⛔️ Failed to patch file: %s", filePath)
			panicAndPause("⛔️ Please try running the program as an Administrator")
		}

		log.Printf("✅ Patched services file: %s", filePath)
	} else {
		log.Printf("✅ Glimesh already exists in: %s", filePath)
	}
}

func findObsDirectories() (services []string) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		panicAndPause(err)
	}

	// Traditional config based path, that beautiful open source projects like OBS Studio use
	obsPath := path.Join(configDir, "obs-studio", "plugin_config", "rtmp-services", "services.json")
	slobsPath := path.Join(configDir, "slobs-client", "plugin_config", "rtmp-services", "services.json")

	if _, err := os.Stat(obsPath); err == nil {
		// OBS Studio Exists
		log.Printf("🔍 Detected OBS Studio at: %s\n", obsPath)
		services = append(services, obsPath)
	}

	if _, err := os.Stat(slobsPath); err == nil {
		// Streamlabs OBS Exists
		log.Printf("🔍 Detected Streamlabs OBS at: %s\n", slobsPath)
		services = append(services, slobsPath)
	}

	// Gross electron packaged non-config directories that we have to inject into
	if runtime.GOOS == "windows" {
		// Weird compiled electron path for Windows SLOBS
		// C:\Program Files\Streamlabs OBS\resources\app.asar.unpacked\node_modules\obs-studio-node\data\obs-plugins\rtmp-services
		slobs32bitPath := path.Join(os.Getenv("programfiles(x86)"), "Streamlabs OBS", "resources", "app.asar.unpacked", "node_modules", "obs-studio-node", "data", "obs-plugins", "rtmp-services", "services.json")

		slobs64bitPath := path.Join(os.Getenv("programfiles"), "Streamlabs OBS", "resources", "app.asar.unpacked", "node_modules", "obs-studio-node", "data", "obs-plugins", "rtmp-services", "services.json")

		if _, err := os.Stat(slobs32bitPath); err == nil {
			// OBS Studio Exists
			log.Printf("🔍 Detected SLOBS Electron 32-bit at: %s\n", slobs32bitPath)
			services = append(services, slobs32bitPath)
		}

		if _, err := os.Stat(slobs64bitPath); err == nil {
			// OBS Studio Exists
			log.Printf("🔍 Detected SLOBS Electron 64-bit at: %s\n", slobs64bitPath)
			services = append(services, slobs64bitPath)
		}

	}

	if runtime.GOOS == "darwin" {
		// Weird compiled electron path for Mac SLOBS
		// /Applications/Streamlabs OBS.app/Contents/Resources/app.asar.unpacked/node_modules/obs-studio-node/data/obs-plugins/rtmp-services/services.json
		slobsAppPath := path.Join("/", "Applications", "Streamlabs OBS.app", "Contents", "Resources", "app.asar.unpacked", "node_modules", "obs-studio-node", "data", "obs-plugins", "rtmp-services", "services.json")

		if _, err := os.Stat(slobsAppPath); err == nil {
			// OBS Studio Exists
			log.Printf("🔍 Detected SLOBS Electron at: %s\n", slobsAppPath)
			services = append(services, slobsAppPath)
		}
	}

	return services
}
