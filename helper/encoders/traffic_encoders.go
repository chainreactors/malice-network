package encoders

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/encoders/traffic"
	"io/fs"
	insecureRand "math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	TrafficEncoderFS = PassthroughEncoderFS{}
	PNG              = PNGEncoder{}
	Nop              = NoEncoder{}
)

func init() {
	// TODO set English dictionary
	//SetEnglishDictionary(assets.English())
	TrafficEncoderFS = PassthroughEncoderFS{
		rootDir: filepath.Join(assets.GetRootAppDir(), "traffic-encoders"),
	}
	loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		// TODO - log this to the server log trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// EncoderMap - A map of all available encoders (native and traffic/wasm)
var EncoderMap = map[uint64]Encoder{
	Base64EncoderID:  Base64{},
	Base58EncoderID:  Base58{},
	Base32EncoderID:  Base32{},
	HexEncoderID:     Hex{},
	EnglishEncoderID: English{},
	GzipEncoderID:    Gzip{},
	PNGEncoderID:     PNG,
}

// TrafficEncoderMap - Keeps track of the loaded traffic encoders (i.e., wasm-based encoder functions)
var TrafficEncoderMap = map[uint64]*traffic.TrafficEncoder{}

// FastEncoderMap - Keeps track of fast native encoders that can be used for large messages
var FastEncoderMap = map[uint64]Encoder{
	Base64EncoderID: Base64{},
	Base58EncoderID: Base58{},
	Base32EncoderID: Base32{},
	HexEncoderID:    Hex{},
	GzipEncoderID:   Gzip{},
}

// SaveTrafficEncoder - Save a traffic encoder to the filesystem
func SaveTrafficEncoder(name string, wasmBin []byte) error {
	if !strings.HasSuffix(name, ".wasm") {
		return fmt.Errorf("invalid encoder name, must end with .wasm")
	}
	// TODO get wasmFilePath from TrafficEncoderFS
	// test
	wasmFilePath := filepath.Join("test", filepath.Base(name))
	err := os.WriteFile(wasmFilePath, wasmBin, 0600)
	if err != nil {
		return err
	}
	return loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		// TODO - log this to the server log trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// RemoveTrafficEncoder - Save a traffic encoder to the filesystem
func RemoveTrafficEncoder(name string) error {
	if !strings.HasSuffix(name, ".wasm") {
		return fmt.Errorf("invalid encoder name, must end with .wasm")
	}
	// TODO get wasmFilePath from TrafficEncoderFS
	// test
	wasmFilePath := filepath.Join("test", filepath.Base(name))
	info, err := os.Stat(wasmFilePath)
	if os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		panic("wasmFilePath is a directory, this should never happen")
	}
	err = os.Remove(wasmFilePath)
	if err != nil {
		return err
	}
	return loadTrafficEncodersFromFS(TrafficEncoderFS, func(msg string) {
		// TODO - log this to the server log trafficEncoderLog.Debugf("[traffic-encoder] %s", msg)
	})
}

// loadTrafficEncodersFromFS - Loads the wasm traffic encoders from the filesystem, for the
// server these will be loaded from: <app root>/traffic-encoders/*.wasm
func loadTrafficEncodersFromFS(encodersFS EncoderFS, logger func(string)) error {

	// Reset references pointing to traffic encoders
	for _, encoder := range TrafficEncoderMap {
		delete(EncoderMap, encoder.ID)
	}
	TrafficEncoderMap = map[uint64]*traffic.TrafficEncoder{}

	// Load WASM encoders
	// TODO - log this to the server log encodersLog.Info("initializing traffic encoder map...")
	wasmEncoderFiles, err := encodersFS.ReadDir("traffic-encoders")
	if err != nil {
		return err
	}
	for _, wasmEncoderFile := range wasmEncoderFiles {
		// TODO - log this to the server log encodersLog.Debugf("checking file: %s", wasmEncoderFile.Name())
		if wasmEncoderFile.IsDir() {
			continue
		}
		if !strings.HasSuffix(wasmEncoderFile.Name(), ".wasm") {
			continue
		}
		// WASM Module name should be equal to file name without the extension
		wasmEncoderModuleName := strings.TrimSuffix(wasmEncoderFile.Name(), ".wasm")
		wasmEncoderData, err := encodersFS.ReadFile(path.Join("traffic-encoders", wasmEncoderFile.Name()))
		if err != nil {
			// TODO - log this to the server log encodersLog.Errorf(fmt.Sprintf("failed to read file %s (%s)", wasmEncoderModuleName, err.Error()))
			return err
		}
		wasmEncoderID := traffic.CalculateWasmEncoderID(wasmEncoderData)
		trafficEncoder, err := traffic.CreateTrafficEncoder(wasmEncoderModuleName, wasmEncoderData, logger)
		if err != nil {
			// TODO - log this to the server log encodersLog.Errorf(fmt.Sprintf("failed to create traffic encoder from '%s': %s", wasmEncoderModuleName, err.Error()))
			return err
		}
		trafficEncoder.FileName = wasmEncoderFile.Name()
		if _, ok := EncoderMap[uint64(wasmEncoderID)]; ok {
			// TODO - log this to the server log encodersLog.Errorf(fmt.Sprintf("duplicate encoder id: %d", wasmEncoderID))
			return fmt.Errorf("duplicate encoder id: %d", wasmEncoderID)
		}
		EncoderMap[uint64(wasmEncoderID)] = trafficEncoder
		TrafficEncoderMap[uint64(wasmEncoderID)] = trafficEncoder
		// TODO - log this to the server log encodersLog.Info(fmt.Sprintf("loaded %s (id: %d, bytes: %d)", wasmEncoderModuleName, wasmEncoderID, len(wasmEncoderData)))
	}
	//TODO - log this to the server log encodersLog.Info(fmt.Sprintf("loaded %d native encoders", len(EncoderMap)-len(TrafficEncoderMap)))
	return nil
}

// EncoderFromNonce - Convert a nonce into an encoder
func EncoderFromNonce(nonce uint64) (uint64, Encoder, error) {
	encoderID := uint64(nonce) % EncoderModulus
	if encoderID == 0 {
		return 0, new(NoEncoder), nil
	}
	if encoder, ok := EncoderMap[encoderID]; ok {
		return encoderID, encoder, nil
	}
	return 0, nil, fmt.Errorf("invalid encoder id: %d", encoderID)
}

// RandomEncoder - Get a random nonce identifier and a matching encoder
func RandomEncoder() (uint64, Encoder) {
	keys := make([]uint64, 0, len(EncoderMap))
	for k := range EncoderMap {
		keys = append(keys, k)
	}
	encoderID := keys[insecureRand.Intn(len(keys))]
	nonce := (randomUint64(MaxN) * EncoderModulus) + encoderID
	return nonce, EncoderMap[encoderID]
}

func randomUint64(max uint64) uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf) % max
}

// PassthroughEncoderFS - Creates an encoder.EncoderFS object from a single local directory
type PassthroughEncoderFS struct {
	rootDir string
}

func (p PassthroughEncoderFS) Open(name string) (fs.File, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if stat, err := os.Stat(localPath); os.IsNotExist(err) || stat.IsDir() {
		return nil, os.ErrNotExist
	}
	return os.Open(localPath)
}

func (p PassthroughEncoderFS) ReadDir(_ string) ([]fs.DirEntry, error) {
	if _, err := os.Stat(p.rootDir); os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	ls, err := os.ReadDir(p.rootDir)
	if err != nil {
		return nil, err
	}
	var entries []fs.DirEntry
	for _, entry := range ls {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".wasm") {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (p PassthroughEncoderFS) ReadFile(name string) ([]byte, error) {
	localPath := filepath.Join(p.rootDir, filepath.Base(name))
	if !strings.HasSuffix(localPath, ".wasm") {
		return nil, os.ErrNotExist
	}
	if stat, err := os.Stat(localPath); os.IsNotExist(err) || stat.IsDir() {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(localPath)
}
